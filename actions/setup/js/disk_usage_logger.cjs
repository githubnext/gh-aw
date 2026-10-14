// @ts-check

const fs = require("fs");
const path = require("path");
const childProcess = require("child_process");
const { DISK_USAGE_JSONL_PATH, TMP_GH_AW_PATH } = require("./constants.cjs");
const { getErrorMessage } = require("./error_helpers.cjs");

const SAMPLE_INTERVAL_MS = 30_000;
const MAX_MONITOR_DURATION_MS = 72 * 60 * 60 * 1000;

/** @param {string} message */
function warning(message) {
  console.warn(`::warning::disk_usage_logger: ${message}`);
}

/** @returns {string} */
function controlPath() {
  if (!process.env.RUNNER_TEMP) throw new Error("RUNNER_TEMP is required for disk monitoring");
  // The agent cannot modify the runner-side setup tree to control the monitor.
  return path.join(process.env.RUNNER_TEMP, "gh-aw", "disk-usage-monitor");
}

/**
 * Snapshot filesystem capacity without scanning files or invoking external tools.
 * Resource labels, rather than directory contents or environment values, are logged.
 * @param {string} operation
 * @returns {boolean}
 */
function logDiskUsage(operation) {
  const resources = [
    ["root", path.parse(process.cwd()).root],
    ["runtime", TMP_GH_AW_PATH],
    ["workspace", process.env.GITHUB_WORKSPACE],
    ["runner_temp", process.env.RUNNER_TEMP],
  ];
  const timestamp = new Date().toISOString();
  const entries = [];
  for (const [resource, directory] of resources) {
    if (!directory) continue;
    try {
      const stats = fs.statfsSync(directory);
      const totalBytes = stats.blocks * stats.bsize;
      const freeBytes = stats.bfree * stats.bsize;
      const availableBytes = stats.bavail * stats.bsize;
      if (![totalBytes, freeBytes, availableBytes, stats.files, stats.ffree].every(value => Number.isSafeInteger(value) && value >= 0)) {
        throw new Error("filesystem statistics exceed the supported non-negative integer range");
      }
      entries.push({
        timestamp,
        source: "statfs",
        operation,
        resource,
        totalBytes,
        freeBytes,
        availableBytes,
        usedBytes: totalBytes - freeBytes,
        // Filesystems that do not report inode capacity return zero.
        totalInodes: stats.files || null,
        freeInodes: stats.files ? stats.ffree : null,
      });
    } catch (error) {
      warning(`failed to sample ${resource}: ${getErrorMessage(error)}`);
    }
  }
  if (entries.length === 0) return false;
  try {
    fs.mkdirSync(path.dirname(DISK_USAGE_JSONL_PATH), { recursive: true });
    fs.appendFileSync(DISK_USAGE_JSONL_PATH, entries.map(entry => JSON.stringify(entry) + "\n").join(""));
    return true;
  } catch (error) {
    warning(`failed to write disk usage: ${getErrorMessage(error)}`);
    return false;
  }
}

/** @returns {void} */
function startDiskUsageMonitor() {
  const control = controlPath();
  if (fs.existsSync(control)) {
    warning("disk monitor is already running");
    return;
  }
  fs.mkdirSync(path.dirname(control), { recursive: true });
  if (!logDiskUsage("setup")) return;
  // Do not retain credentials in the long-lived sampler's environment.
  const env = Object.fromEntries(["RUNNER_TEMP", "GITHUB_WORKSPACE", "RUNNER_TRACKING_ID"].flatMap(key => (process.env[key] === undefined ? [] : [[key, process.env[key]]])));
  const log = fs.openSync(path.join(TMP_GH_AW_PATH, "disk_usage_monitor.log"), "a");
  try {
    fs.writeFileSync(control, "");
    const child = childProcess.spawn(process.execPath, [__filename, "monitor"], { detached: true, stdio: ["ignore", log, log], env });
    child.on("error", error => {
      fs.rmSync(control, { force: true });
      warning(`failed to start disk monitor: ${getErrorMessage(error)}`);
    });
    child.unref();
  } catch (error) {
    fs.rmSync(control, { force: true });
    throw error;
  } finally {
    fs.closeSync(log);
  }
}

/**
 * The stop request is a file, not a PID supplied by the agent. The worker
 * acknowledges shutdown by removing it before artifact redaction/upload.
 * @returns {Promise<void>}
 */
async function stopDiskUsageMonitor() {
  const control = controlPath();
  if (!fs.existsSync(control)) return;
  fs.writeFileSync(control, "stop");
  const deadline = Date.now() + 5000;
  while (fs.existsSync(control) && Date.now() < deadline) {
    await new Promise(resolve => setTimeout(resolve, 100));
  }
  if (fs.existsSync(control)) warning("disk monitor did not acknowledge shutdown within 5 seconds");
}

/** @returns {void} */
function runDiskUsageMonitor() {
  const control = controlPath();
  const startedAt = Date.now();
  let sampledAt = startedAt;
  const timer = setInterval(() => {
    try {
      const stopping = !fs.existsSync(control) || fs.readFileSync(control, "utf8") === "stop" || Date.now() - startedAt >= MAX_MONITOR_DURATION_MS;
      if (stopping) {
        logDiskUsage("shutdown");
        fs.rmSync(control, { force: true });
        clearInterval(timer);
      } else if (Date.now() - sampledAt >= SAMPLE_INTERVAL_MS) {
        logDiskUsage("interval");
        sampledAt = Date.now();
      }
    } catch (error) {
      warning(`disk monitor failed: ${getErrorMessage(error)}`);
      clearInterval(timer);
      fs.rmSync(control, { force: true });
    }
  }, 1000);
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
        default:
          throw new Error("expected start, stop, or monitor");
      }
    } catch (error) {
      warning(getErrorMessage(error));
      process.exitCode = 1;
    }
  })();
}

module.exports = { logDiskUsage, startDiskUsageMonitor, stopDiskUsageMonitor, runDiskUsageMonitor, SAMPLE_INTERVAL_MS };
