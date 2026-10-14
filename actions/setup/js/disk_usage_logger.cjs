// @ts-check

const fs = require("fs");
const path = require("path");
const childProcess = require("child_process");
const { TMP_GH_AW_PATH } = require("./constants.cjs");
const { spaceBand, inodeBand, normalizeDiskUsage, diskTimestamp, diskWarning, diskErrorCode, sanitizeDiskDiagnostics, openDiskDirectory, openDiskFile, closeDiskFile, removeDiskControl } = require("./disk_usage.cjs");

const SAMPLE_INTERVAL_MS = 30_000;
const MAX_MONITOR_DURATION_MS = 72 * 60 * 60 * 1000;
const MAX_TELEMETRY_BYTES = 16 * 1024 * 1024;

/** @returns {{fd: number, anchor: string}} */
function controlDirectory() {
  if (!process.env.RUNNER_TEMP) throw new Error("runner temp unavailable");
  const temp = openDiskDirectory(process.env.RUNNER_TEMP);
  try {
    const directory = path.join(temp.anchor, "gh-aw");
    try {
      fs.mkdirSync(directory, { mode: 0o700 });
    } catch (error) {
      if (diskErrorCode(error) !== "EEXIST") throw error;
    }
    return openDiskDirectory(directory);
  } finally {
    closeDiskFile(temp.fd);
  }
}

/** @param {string} operation @returns {boolean} */
function logDiskUsage(operation) {
  let directory;
  let output;
  try {
    if (!["setup", "interval", "shutdown"].includes(operation)) throw new Error("invalid operation");
    const resources = [
      ["root", path.parse(process.cwd()).root],
      ["runtime", TMP_GH_AW_PATH],
      ["workspace", process.env.GITHUB_WORKSPACE],
      ["runner_temp", process.env.RUNNER_TEMP],
    ];
    const timestamp = new Date().toISOString();
    const entries = [];
    for (const [resource, location] of resources) {
      if (!location) continue;
      let availableSpaceBand = "unavailable";
      let inodeHeadroom = "unavailable";
      try {
        // statfs follows legitimate directory symlinks; no target names are persisted.
        const stats = fs.statfsSync(location);
        const available = stats.bavail * stats.bsize;
        if (!Number.isSafeInteger(stats.bsize) || stats.bsize <= 0 || ![available, stats.files, stats.ffree].every(value => Number.isSafeInteger(value) && value >= 0) || stats.ffree > stats.files) throw new Error("invalid statistics");
        availableSpaceBand = spaceBand(available);
        inodeHeadroom = inodeBand(stats.files, stats.ffree);
      } catch (error) {
        diskWarning(`failed to sample ${resource}`, error);
      }
      entries.push({ timestamp, source: "statfs", operation, resource, availableSpaceBand, inodeHeadroom });
    }
    directory = openDiskDirectory(TMP_GH_AW_PATH);
    output = openDiskFile(path.join(directory.anchor, "disk_usage.jsonl"), fs.constants.O_WRONLY | fs.constants.O_CREAT | fs.constants.O_APPEND);
    const content = entries.map(entry => JSON.stringify(entry) + "\n").join("");
    if (fs.fstatSync(output).size + Buffer.byteLength(content) > MAX_TELEMETRY_BYTES) {
      diskWarning("telemetry size limit reached");
      return false;
    }
    fs.writeFileSync(output, content);
    return true;
  } catch (error) {
    diskWarning("failed to write disk usage", error);
    return false;
  } finally {
    closeDiskFile(output);
    closeDiskFile(directory?.fd);
  }
}

/** @returns {void} */
function startDiskUsageMonitor() {
  let directory;
  let control;
  let runtime;
  let output;
  let log;
  let claimed = false;
  let launched = false;
  try {
    directory = controlDirectory();
    control = path.join(directory.anchor, "disk-usage-monitor");
    const fd = openDiskFile(control, fs.constants.O_WRONLY | fs.constants.O_CREAT | fs.constants.O_EXCL);
    claimed = true;
    closeDiskFile(fd);
    runtime = openDiskDirectory(TMP_GH_AW_PATH);
    output = openDiskFile(path.join(runtime.anchor, "disk_usage.jsonl"), fs.constants.O_WRONLY | fs.constants.O_CREAT);
    fs.ftruncateSync(output, 0);
    closeDiskFile(output);
    output = undefined;
    if (!logDiskUsage("setup")) return;
    log = openDiskFile(path.join(runtime.anchor, "disk_usage_monitor.log"), fs.constants.O_WRONLY | fs.constants.O_CREAT);
    fs.ftruncateSync(log, 0);
    const env = Object.fromEntries(["RUNNER_TEMP", "GITHUB_WORKSPACE", "RUNNER_TRACKING_ID"].flatMap(key => (process.env[key] === undefined ? [] : [[key, process.env[key]]])));
    const child = childProcess.spawn(process.execPath, [__filename, "monitor"], { detached: true, stdio: ["ignore", log, "ignore"], env });
    child.on("exit", code => {
      if (code !== 0) diskWarning("disk monitor exited unexpectedly");
    });
    child.on("error", error => {
      diskWarning("failed to start disk monitor", error);
      let cleanup;
      try {
        cleanup = controlDirectory();
        removeDiskControl(path.join(cleanup.anchor, "disk-usage-monitor"));
      } catch (cleanupError) {
        diskWarning("failed to clean up monitor control", cleanupError);
      } finally {
        closeDiskFile(cleanup?.fd);
      }
    });
    child.unref();
    launched = true;
  } catch (error) {
    diskWarning("failed to start disk monitor", error);
  } finally {
    if (claimed && !launched && control) removeDiskControl(control);
    closeDiskFile(log);
    closeDiskFile(output);
    closeDiskFile(runtime?.fd);
    closeDiskFile(directory?.fd);
  }
}

/** @returns {Promise<void>} */
async function stopDiskUsageMonitor() {
  let directory;
  let fd;
  let control;
  try {
    directory = controlDirectory();
    control = path.join(directory.anchor, "disk-usage-monitor");
    try {
      fd = openDiskFile(control, fs.constants.O_WRONLY);
    } catch (error) {
      if (diskErrorCode(error) === "ENOENT") return;
      throw error;
    }
    logDiskUsage("shutdown");
    fs.writeFileSync(fd, "stop");
    const deadline = Date.now() + 5000;
    while (fs.fstatSync(fd).nlink !== 0 && Date.now() < deadline) await new Promise(resolve => setTimeout(resolve, 100));
    if (fs.fstatSync(fd).nlink !== 0) diskWarning("disk monitor did not acknowledge shutdown within 5 seconds");
  } catch (error) {
    diskWarning("failed to stop disk monitor", error);
    if (control) removeDiskControl(control);
  } finally {
    closeDiskFile(fd);
    closeDiskFile(directory?.fd);
    collectDiskUsageArtifact(false);
  }
}

/** @returns {void} */
function runDiskUsageMonitor() {
  let directory;
  let fd;
  let watcher;
  let timer;
  let finished = false;
  const startedAt = Date.now();
  const finish = (sample = false) => {
    if (finished) return;
    finished = true;
    clearInterval(timer);
    try {
      watcher?.close();
    } catch (error) {
      diskWarning("failed to close monitor watcher", error);
    }
    if (sample) logDiskUsage("shutdown");
    if (directory) removeDiskControl(path.join(directory.anchor, "disk-usage-monitor"));
    closeDiskFile(fd);
    closeDiskFile(directory?.fd);
  };
  const check = () => {
    try {
      const buffer = Buffer.alloc(4);
      const size = fs.readSync(fd, buffer, 0, 4, 0);
      if (fs.fstatSync(fd).nlink === 0 || buffer.toString("utf8", 0, size) === "stop") finish();
      else if (Date.now() - startedAt >= MAX_MONITOR_DURATION_MS) finish(true);
    } catch (error) {
      diskWarning("disk monitor control unavailable", error);
      finish();
    }
  };
  try {
    directory = controlDirectory();
    fd = openDiskFile(path.join(directory.anchor, "disk-usage-monitor"), fs.constants.O_RDONLY);
    watcher = fs.watch(`/proc/self/fd/${fd}`, check);
    watcher.on("error", error => {
      diskWarning("disk monitor watcher unavailable", error);
      finish();
    });
    timer = setInterval(() => {
      check();
      if (!finished && !logDiskUsage("interval")) finish();
    }, SAMPLE_INTERVAL_MS);
    check();
  } catch (error) {
    diskWarning("failed to run disk monitor", error);
    finish();
  }
}

/** Sanitize both primary/fallback and usage artifacts; never copy raw legacy data.
 * @param {boolean} [copyToUsage]
 * @returns {void}
 */
function collectDiskUsageArtifact(copyToUsage = true) {
  let directory;
  let source;
  let usage;
  let destination;
  try {
    directory = openDiskDirectory(TMP_GH_AW_PATH);
    try {
      source = openDiskFile(path.join(directory.anchor, "disk_usage.jsonl"), fs.constants.O_RDWR);
    } catch (error) {
      if (diskErrorCode(error) === "ENOENT") return;
      throw error;
    }
    if (fs.fstatSync(source).size > MAX_TELEMETRY_BYTES) throw new Error("telemetry size limit");
    const entries = [];
    for (const line of fs.readFileSync(source, "utf8").split(/\r?\n/)) {
      if (!line.trim()) continue;
      try {
        const record = JSON.parse(line);
        const data = normalizeDiskUsage(record);
        if (!data) throw new Error("invalid disk record");
        const timestamp = diskTimestamp(record.timestamp);
        entries.push({ ...(timestamp ? { timestamp } : {}), ...data });
      } catch (error) {
        diskWarning("discarded invalid disk record", error);
      }
    }
    const content = entries.map(entry => JSON.stringify(entry) + "\n").join("");
    fs.ftruncateSync(source, 0);
    fs.writeSync(source, content, 0, "utf8");
    if (!copyToUsage) return;
    usage = openDiskDirectory(path.join(directory.anchor, "usage"));
    destination = openDiskFile(path.join(usage.anchor, "disk_usage.jsonl"), fs.constants.O_WRONLY | fs.constants.O_CREAT);
    fs.ftruncateSync(destination, 0);
    fs.writeFileSync(destination, content);
  } catch (error) {
    diskWarning("failed to collect disk usage artifact", error);
  } finally {
    if (directory) {
      let diagnostic;
      try {
        diagnostic = openDiskFile(path.join(directory.anchor, "disk_usage_monitor.log"), fs.constants.O_RDWR);
        if (fs.fstatSync(diagnostic).size > MAX_TELEMETRY_BYTES) throw new Error("diagnostic size limit");
        const content = sanitizeDiskDiagnostics(fs.readFileSync(diagnostic, "utf8"));
        fs.ftruncateSync(diagnostic, 0);
        fs.writeSync(diagnostic, content, 0, "utf8");
      } catch (error) {
        if (diskErrorCode(error) !== "ENOENT") diskWarning("failed to sanitize monitor diagnostics", error);
      } finally {
        closeDiskFile(diagnostic);
      }
    }
    closeDiskFile(destination);
    closeDiskFile(usage?.fd);
    closeDiskFile(source);
    closeDiskFile(directory?.fd);
  }
}

if (require.main === module) {
  (async () => {
    try {
      switch (process.argv[2]) {
        case "start":
          startDiskUsageMonitor();
          break;
        case "stop":
          await stopDiskUsageMonitor();
          break;
        case "monitor":
          runDiskUsageMonitor();
          break;
        case "collect":
          collectDiskUsageArtifact();
          break;
        default:
          diskWarning("invalid monitor operation");
      }
    } catch (error) {
      diskWarning("monitor unavailable", error);
    }
  })();
}

module.exports = { logDiskUsage, startDiskUsageMonitor, stopDiskUsageMonitor, runDiskUsageMonitor, collectDiskUsageArtifact, SAMPLE_INTERVAL_MS };
