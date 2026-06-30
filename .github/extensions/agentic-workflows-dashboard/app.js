// src/dashboard-cli.ts
import { spawn } from "node:child_process";
import { constants as fsConstants } from "node:fs";
import { access } from "node:fs/promises";
import { join } from "node:path";
var INSTALL_COMMAND = "gh extension install github/gh-aw";
var GH_INSTALL_URL = "https://cli.github.com";
var LOG = "[dashboard-cli]";
function combineOutput(stdout, stderr) {
  return [stdout, stderr].filter(Boolean).join("\n").trim();
}
function spawnExecFile(file, args, options, callback) {
  const { env, cwd, maxBuffer = 10 * 1024 * 1024 } = options ?? {};
  const spawnOptions = { env, cwd, stdio: ["ignore", "pipe", "pipe"], windowsHide: true, detached: true };
  console.error(`${LOG} spawn file=${file} args=${JSON.stringify(args)} cwd=${cwd}`);
  const proc = spawn(file, args, spawnOptions);
  const stdoutChunks = [];
  const stderrChunks = [];
  let stdoutLen = 0;
  let stderrLen = 0;
  let overflowed = false;
  proc.stdout?.on("data", (chunk) => {
    stdoutLen += chunk.length;
    if (stdoutLen > maxBuffer) {
      overflowed = true;
      return;
    }
    stdoutChunks.push(chunk);
  });
  proc.stderr?.on("data", (chunk) => {
    stderrLen += chunk.length;
    if (stderrLen > maxBuffer) {
      overflowed = true;
      return;
    }
    stderrChunks.push(chunk);
  });
  proc.on("error", (err) => {
    console.error(`${LOG} spawn error file=${file} args=${JSON.stringify(args)}: ${err.message}`);
    callback(err, "", "");
  });
  proc.on("close", (code) => {
    const stdout = Buffer.concat(stdoutChunks).toString("utf8");
    const stderr = Buffer.concat(stderrChunks).toString("utf8");
    if (code !== 0 || stderr) {
      console.error(`${LOG} spawn close file=${file} code=${code} stdout=${stdout.length}B stderr=${stderr.length}B${stderr ? ` stderr: ${stderr.slice(0, 300)}` : ""}`);
    }
    if (overflowed) {
      const err = new Error("stdout/stderr maxBuffer exceeded");
      err.code = "ERR_CHILD_PROCESS_STDIO_MAXBUFFER";
      callback(err, stdout, stderr);
    } else if (code !== 0) {
      const err = new Error(`Command failed with exit code ${code}`);
      err.code = code ?? 1;
      callback(err, stdout, stderr);
    } else {
      callback(null, stdout, stderr);
    }
  });
}
function execp(bin, args, cwd, { combineIO = false, execFileFn = spawnExecFile, env = process.env } = {}) {
  return new Promise((resolve, reject) => {
    execFileFn(
      bin,
      args,
      {
        cwd,
        env: { ...env, CI: "1", NO_COLOR: "1", GH_NO_UPDATE_NOTIFIER: "1" },
        maxBuffer: 10 * 1024 * 1024
      },
      (err, stdout, stderr) => {
        const output = combineOutput(stdout ?? "", stderr ?? "");
        if (err) {
          reject(Object.assign(err, { stderr: stderr ?? "", stdout: stdout ?? "", output }));
          return;
        }
        resolve(combineIO ? output : stdout);
      }
    );
  });
}
function parseVersionFromOutput(output) {
  const trimmed = String(output ?? "").trim();
  if (!trimmed) return "";
  const match = trimmed.match(/gh(?:-aw| aw) version ([^\r\n]+)/i);
  return match?.[1]?.trim() ?? "";
}
function isMissingGh(error) {
  const e = error;
  return e?.code === "ENOENT" && e?.syscall === "spawn" && e?.path === "gh";
}
function isMissingGhAwExtension(error) {
  const e = error;
  const output = String(e?.output ?? e?.stderr ?? e?.message ?? "");
  return /extension not found:\s*aw/i.test(output) || /unknown command ["']aw["'] for ["']gh["']/i.test(output);
}
async function findDevBinary(cwd, accessFn = access, platform = process.platform) {
  const devBin = join(cwd, platform === "win32" ? "gh-aw.exe" : "gh-aw");
  try {
    await accessFn(devBin, fsConstants.X_OK);
    console.error(`${LOG} findDevBinary found: ${devBin}`);
    return devBin;
  } catch {
    console.error(`${LOG} findDevBinary not found at ${devBin}, falling back to gh extension`);
    return null;
  }
}
function createGhAwRunner({ getWorkspacePath, accessFn = access, execFileFn = spawnExecFile, platform = process.platform, env = process.env, resolveBin }) {
  const binCache = /* @__PURE__ */ new Map();
  const _resolveBin = resolveBin ?? (() => {
    const cwd = getWorkspacePath();
    if (!binCache.has(cwd)) {
      binCache.set(cwd, findDevBinary(cwd, accessFn, platform));
    }
    return binCache.get(cwd);
  });
  function runExec(bin, args, cwd, options) {
    return execp(bin, args, cwd, { ...options, execFileFn, env });
  }
  return async function runGhAw(args) {
    const cwd = getWorkspacePath();
    const devBin = await _resolveBin();
    if (devBin) {
      console.error(`${LOG} runGhAw using dev-binary: ${devBin} args=${JSON.stringify(args)} cwd=${cwd}`);
      return runExec(devBin, args, cwd);
    }
    console.error(`${LOG} runGhAw using gh extension args=${JSON.stringify(args)} cwd=${cwd}`);
    return runExec("gh", ["aw", ...args], cwd);
  };
}
function createGhAwRunnerWithStatus(options) {
  const binCache = /* @__PURE__ */ new Map();
  const resolveBin = () => {
    const cwd = options.getWorkspacePath();
    if (!binCache.has(cwd)) {
      binCache.set(cwd, findDevBinary(cwd, options.accessFn ?? access, options.platform ?? process.platform));
    }
    return binCache.get(cwd);
  };
  const runGhAw = createGhAwRunner({ ...options, resolveBin });
  const getStatus = async () => {
    const cwd = options.getWorkspacePath();
    const devBin = await resolveBin();
    if (devBin) {
      const output = await execp(devBin, ["version"], cwd, {
        combineIO: true,
        execFileFn: options.execFileFn ?? spawnExecFile,
        env: options.env ?? process.env
      });
      const status = {
        available: true,
        source: "dev-binary",
        version: parseVersionFromOutput(output) || "unknown",
        command: `${devBin} version`,
        installCommand: INSTALL_COMMAND
      };
      console.error(`${LOG} getStatus: available=${status.available} source=${status.source} version=${status.version} cwd=${cwd}`);
      return status;
    }
    try {
      const output = await execp("gh", ["aw", "version"], cwd, {
        combineIO: true,
        execFileFn: options.execFileFn ?? spawnExecFile,
        env: options.env ?? process.env
      });
      const status = {
        available: true,
        source: "gh-extension",
        version: parseVersionFromOutput(output) || "unknown",
        command: "gh aw version",
        installCommand: INSTALL_COMMAND
      };
      console.error(`${LOG} getStatus: available=${status.available} source=${status.source} version=${status.version} cwd=${cwd}`);
      return status;
    } catch (error) {
      if (isMissingGh(error)) {
        console.error(`${LOG} getStatus error: gh not found in PATH cwd=${cwd}`);
        return {
          available: false,
          source: "gh-not-found",
          version: "",
          command: "gh aw version",
          installCommand: INSTALL_COMMAND,
          installUrl: GH_INSTALL_URL,
          message: "Install the GitHub CLI to use this dashboard."
        };
      }
      if (isMissingGhAwExtension(error)) {
        console.error(`${LOG} getStatus error: gh aw extension not installed cwd=${cwd}`);
        return {
          available: false,
          source: "missing",
          version: "",
          command: "gh aw version",
          installCommand: INSTALL_COMMAND,
          message: "gh aw is not installed. Install the GitHub CLI extension to use the dashboard outside a local dev build."
        };
      }
      const e = error;
      const message = String(e?.output ?? e?.stderr ?? e?.message ?? "Failed to detect gh aw.");
      console.error(`${LOG} getStatus error: ${message} cwd=${cwd}`);
      return {
        available: false,
        source: "error",
        version: "",
        command: "gh aw version",
        installCommand: INSTALL_COMMAND,
        message
      };
    }
  };
  runGhAw.getStatus = getStatus;
  return runGhAw;
}

// src/dashboard-config.ts
var CACHE_TTL_MS = 6e4;
var DEFAULT_LOG_TIMEOUT_MINUTES = 1;
var DEFAULT_REPORT_WINDOW_ID = "7d";
var DEFAULT_RUN_COUNT = 100;
var MAX_LOG_CONTINUATIONS = 6;
var REPORT_WINDOWS = {
  "3d": { id: "3d", label: "3 days", startDate: "-3d", days: 3 },
  "7d": { id: "7d", label: "7 days", startDate: "-1w", days: 7 },
  "1mo": { id: "1mo", label: "1 month", startDate: "-1mo", days: 30 }
};
function getReportWindow(windowId) {
  if (windowId && windowId in REPORT_WINDOWS) {
    return REPORT_WINDOWS[windowId];
  }
  return REPORT_WINDOWS[DEFAULT_REPORT_WINDOW_ID];
}

// src/dashboard-logs.ts
function parsePositiveInt(value, fallback) {
  const numeric = Number.parseInt(String(value ?? fallback), 10);
  return Number.isFinite(numeric) && numeric > 0 ? numeric : fallback;
}
function readFlagValue(args, index, arg) {
  const equalsIndex = arg.indexOf("=");
  if (equalsIndex >= 0) {
    return { value: arg.slice(equalsIndex + 1), nextIndex: index };
  }
  return { value: args[index + 1] ?? "", nextIndex: index + 1 };
}
function normalizeLogsOptions(options = {}) {
  const windowId = typeof options.window === "string" ? options.window : options.window?.id;
  const window = getReportWindow(windowId);
  const artifacts = Array.isArray(options.artifacts) && options.artifacts.length > 0 ? options.artifacts : ["usage"];
  return {
    window,
    count: parsePositiveInt(options.count, DEFAULT_RUN_COUNT),
    timeout: parsePositiveInt(options.timeout, DEFAULT_LOG_TIMEOUT_MINUTES),
    startDate: typeof options.startDate === "string" && options.startDate.trim() ? options.startDate.trim() : window.startDate,
    endDate: typeof options.endDate === "string" && options.endDate.trim() ? options.endDate.trim() : "",
    beforeRunID: Number.isFinite(Number(options.beforeRunID)) && Number(options.beforeRunID) > 0 ? Number(options.beforeRunID) : 0,
    afterRunID: Number.isFinite(Number(options.afterRunID)) && Number(options.afterRunID) > 0 ? Number(options.afterRunID) : 0,
    workflowName: typeof options.workflowName === "string" ? options.workflowName.trim() : "",
    engine: typeof options.engine === "string" ? options.engine.trim() : "",
    branch: typeof options.branch === "string" ? options.branch.trim() : "",
    artifacts
  };
}
function buildLogsArgs(options) {
  const args = ["logs", "--json", "-c", String(options.count), "--timeout", String(options.timeout)];
  if (options.workflowName) args.push(options.workflowName);
  if (options.startDate) args.push("--start-date", options.startDate);
  if (options.endDate) args.push("--end-date", options.endDate);
  if (options.engine) args.push("--engine", options.engine);
  if (options.branch) args.push("--ref", options.branch);
  if (options.beforeRunID > 0) args.push("--before-run-id", String(options.beforeRunID));
  if (options.afterRunID > 0) args.push("--after-run-id", String(options.afterRunID));
  if (options.artifacts.length > 0) args.push("--artifacts", options.artifacts.join(","));
  return args;
}
function continuationToLogsOptions(continuation, fallback) {
  if (!continuation) return null;
  return normalizeLogsOptions({
    window: fallback.window.id,
    workflowName: continuation.workflow_name || fallback.workflowName,
    count: continuation.count || fallback.count,
    startDate: continuation.start_date || fallback.startDate,
    endDate: continuation.end_date || fallback.endDate,
    engine: continuation.engine || fallback.engine,
    branch: continuation.branch || fallback.branch,
    afterRunID: continuation.after_run_id || fallback.afterRunID,
    beforeRunID: continuation.before_run_id || fallback.beforeRunID,
    timeout: continuation.timeout || fallback.timeout,
    artifacts: fallback.artifacts
  });
}
function mergeRuns(existingRuns, nextRuns) {
  const merged = new Map(existingRuns.map((run) => [run.run_id, run]));
  for (const run of nextRuns) {
    if (run?.run_id != null) {
      merged.set(run.run_id, run);
    }
  }
  return Array.from(merged.values()).sort((a, b) => Number(b.run_id ?? 0) - Number(a.run_id ?? 0));
}
function parseGhAwArgs(raw) {
  const match = raw.trim().match(/^(?:gh\s+aw\s+)(.+)$/);
  return match?.[1] ? match[1].trim().split(/\s+/) : null;
}
function hasFlag(args, longFlag, shortFlag = "") {
  return args.some((arg) => {
    if (arg.startsWith(`${longFlag}=`)) return true;
    if (shortFlag && arg.startsWith(`${shortFlag}=`)) return true;
    return arg === longFlag || shortFlag !== "" && arg === shortFlag;
  });
}
function logsCommandUsesJSON(args) {
  return hasFlag(args, "--json", "-j");
}
function normalizeLogsCommandArgs(args, windowId, timeoutMinutes) {
  const nextArgs = [...args];
  if (!hasFlag(nextArgs, "--start-date") && !hasFlag(nextArgs, "--end-date") && !hasFlag(nextArgs, "--after-run-id") && !hasFlag(nextArgs, "--before-run-id")) {
    nextArgs.push("--start-date", getReportWindow(windowId).startDate);
  }
  if (!hasFlag(nextArgs, "--timeout")) {
    nextArgs.push("--timeout", String(timeoutMinutes));
  }
  if (!hasFlag(nextArgs, "--artifacts")) {
    nextArgs.push("--artifacts", "usage");
  }
  return nextArgs;
}
function logsArgsToOptions(args, fallback = {}) {
  const options = {
    window: typeof fallback.window === "string" ? fallback.window : fallback.window?.id,
    count: fallback.count,
    timeout: fallback.timeout,
    startDate: fallback.startDate,
    endDate: fallback.endDate,
    beforeRunID: fallback.beforeRunID,
    afterRunID: fallback.afterRunID,
    workflowName: fallback.workflowName,
    engine: fallback.engine,
    branch: fallback.branch,
    artifacts: fallback.artifacts
  };
  for (let index = 1; index < args.length; index += 1) {
    const arg = args[index] ?? "";
    if (!arg.startsWith("-")) {
      if (!options.workflowName) {
        options.workflowName = arg;
      }
      continue;
    }
    if (arg === "--json" || arg === "-j") {
      continue;
    }
    if (arg === "-c" || arg.startsWith("-c=") || arg === "--count" || arg.startsWith("--count=")) {
      const { value, nextIndex } = readFlagValue(args, index, arg);
      options.count = value;
      index = nextIndex;
      continue;
    }
    if (arg === "--timeout" || arg.startsWith("--timeout=")) {
      const { value, nextIndex } = readFlagValue(args, index, arg);
      options.timeout = value;
      index = nextIndex;
      continue;
    }
    if (arg === "--start-date" || arg.startsWith("--start-date=")) {
      const { value, nextIndex } = readFlagValue(args, index, arg);
      options.startDate = value;
      index = nextIndex;
      continue;
    }
    if (arg === "--end-date" || arg.startsWith("--end-date=")) {
      const { value, nextIndex } = readFlagValue(args, index, arg);
      options.endDate = value;
      index = nextIndex;
      continue;
    }
    if (arg === "--before-run-id" || arg.startsWith("--before-run-id=")) {
      const { value, nextIndex } = readFlagValue(args, index, arg);
      options.beforeRunID = value;
      index = nextIndex;
      continue;
    }
    if (arg === "--after-run-id" || arg.startsWith("--after-run-id=")) {
      const { value, nextIndex } = readFlagValue(args, index, arg);
      options.afterRunID = value;
      index = nextIndex;
      continue;
    }
    if (arg === "--engine" || arg.startsWith("--engine=") || arg === "-e" || arg.startsWith("-e=")) {
      const { value, nextIndex } = readFlagValue(args, index, arg);
      options.engine = value;
      index = nextIndex;
      continue;
    }
    if (arg === "--ref" || arg.startsWith("--ref=")) {
      const { value, nextIndex } = readFlagValue(args, index, arg);
      options.branch = value;
      index = nextIndex;
      continue;
    }
    if (arg === "--artifacts" || arg.startsWith("--artifacts=")) {
      const { value, nextIndex } = readFlagValue(args, index, arg);
      options.artifacts = value.split(",").map((item) => item.trim()).filter(Boolean);
      index = nextIndex;
    }
  }
  return normalizeLogsOptions(options);
}

// src/usage-forecast.ts
import { basename } from "node:path";
function toNumber(value) {
  const numeric = Number(value ?? 0);
  return Number.isFinite(numeric) ? numeric : 0;
}
function normalizeWorkflowID(value) {
  const raw = String(value ?? "").trim();
  if (!raw) return "";
  let name = basename(raw);
  const lowerName = name.toLowerCase();
  for (const suffix of [".lock.yml", ".yml", ".yaml", ".md"]) {
    if (lowerName.endsWith(suffix)) {
      name = name.slice(0, -suffix.length);
      break;
    }
  }
  return name.trim();
}
function forecastDaysForWindow(window) {
  return window?.id === "1mo" ? 30 : 7;
}
function getForecastMonthlyAIC(forecast) {
  if (!forecast || typeof forecast !== "object") return 0;
  const monteCarloP50 = toNumber(forecast.monthly_monte_carlo?.p50_projected_aic);
  if (monteCarloP50 > 0) return monteCarloP50;
  return toNumber(forecast.monthly_projected_aic);
}
function applyForecastToUsageSummary(items, forecastWorkflows = []) {
  const forecastEntries = forecastWorkflows.map((forecast) => [normalizeWorkflowID(forecast?.workflow_id || forecast?.workflow_path), getForecastMonthlyAIC(forecast)]).filter(([workflowID]) => Boolean(workflowID));
  const forecastByWorkflow = new Map(forecastEntries);
  return items.map((item) => ({
    ...item,
    monthly_forecast_aic: forecastByWorkflow.get(item.workflow_id) ?? 0
  }));
}
function buildUsageSummary(runs, window, forecastWorkflows = []) {
  const usageByWorkflow = /* @__PURE__ */ new Map();
  const effectiveDays = Number(window?.days ?? 0);
  if (!Number.isFinite(effectiveDays) || effectiveDays <= 0) {
    throw new Error(`report window '${window?.id ?? "unknown"}' is missing a valid positive day count.`);
  }
  for (const run of runs) {
    const workflowPath = typeof run?.workflow_path === "string" ? run.workflow_path.trim() : "";
    const workflowID = normalizeWorkflowID(workflowPath || run?.workflow_name);
    if (!workflowID) continue;
    const workflowName = String(run?.workflow_name ?? workflowID).trim() || workflowID;
    const aic = toNumber(run?.aic);
    const entry = usageByWorkflow.get(workflowID) ?? {
      workflow_id: workflowID,
      workflow_name: workflowName,
      workflow_path: workflowPath,
      run_count: 0,
      total_aic: 0,
      cost_per_run: 0,
      daily_aic: 0,
      monthly_forecast_aic: 0,
      last_run_at: ""
    };
    entry.run_count += 1;
    entry.total_aic += aic;
    if (!entry.workflow_path && workflowPath) {
      entry.workflow_path = workflowPath;
    }
    if (!entry.workflow_name && workflowName) {
      entry.workflow_name = workflowName;
    }
    const createdAt = typeof run?.created_at === "string" ? run.created_at : "";
    if (createdAt && (!entry.last_run_at || createdAt > entry.last_run_at)) {
      entry.last_run_at = createdAt;
    }
    usageByWorkflow.set(workflowID, entry);
  }
  const items = Array.from(usageByWorkflow.values()).map((entry) => {
    const costPerRun = entry.run_count > 0 ? entry.total_aic / entry.run_count : 0;
    const dailyAIC = entry.total_aic / effectiveDays;
    return {
      ...entry,
      cost_per_run: costPerRun,
      daily_aic: dailyAIC,
      monthly_forecast_aic: 0
    };
  }).sort((a, b) => {
    const dailyDelta = b.daily_aic - a.daily_aic;
    if (dailyDelta !== 0) return dailyDelta;
    return b.cost_per_run - a.cost_per_run;
  });
  return applyForecastToUsageSummary(items, forecastWorkflows);
}

// src/dashboard-data.ts
var LOG2 = "[dashboard-data]";
function asError(value) {
  if (value instanceof Error) {
    return value;
  }
  return new Error(String(value));
}
function parseJsonOutput(raw, context) {
  const trimmed = (raw ?? "").trim();
  if (!trimmed) {
    console.error(`${LOG2} parseJsonOutput: no output for context="${context}"`);
    throw new Error(`${context}: command produced no output`);
  }
  try {
    return JSON.parse(trimmed);
  } catch {
    const jsonStart = trimmed.search(/[{[]/);
    if (jsonStart > 0) {
      try {
        return JSON.parse(trimmed.slice(jsonStart));
      } catch {
      }
    }
    const snippet = trimmed.replace(/\s+/g, " ").slice(0, 200);
    console.error(`${LOG2} parseJsonOutput: JSON parse failed context="${context}" snippet=${snippet}`);
    throw new Error(`${context}: failed to parse JSON (output: ${snippet})`);
  }
}
function createDashboardDataAccess({ runGhAw, cacheTTL = CACHE_TTL_MS, logsOutputDir }) {
  const cache = /* @__PURE__ */ new Map();
  function getCached(key) {
    const entry = cache.get(key);
    return entry && Date.now() < entry.expiresAt ? entry.data : null;
  }
  function setCached(key, data) {
    cache.set(key, { data, expiresAt: Date.now() + cacheTTL });
  }
  async function getDefinitions() {
    const hit = getCached("definitions");
    if (hit) {
      console.error(`${LOG2} getDefinitions: cache hit count=${hit.length}`);
      return hit;
    }
    console.error(`${LOG2} getDefinitions: fetching from CLI`);
    try {
      const raw = await runGhAw(["status", "--json"]);
      const parsed = parseJsonOutput(raw, "gh aw status --json");
      const data = Array.isArray(parsed) ? parsed : [];
      setCached("definitions", data);
      console.error(`${LOG2} getDefinitions: fetched count=${data.length}`);
      return data;
    } catch (err) {
      console.error(`${LOG2} getDefinitions error: ${asError(err).message}`);
      throw err;
    }
  }
  async function getExperiments() {
    const hit = getCached("experiments");
    if (hit) {
      console.error(`${LOG2} getExperiments: cache hit count=${hit.length}`);
      return hit;
    }
    console.error(`${LOG2} getExperiments: fetching from CLI`);
    try {
      const raw = await runGhAw(["experiments", "list", "--json"]);
      const parsed = parseJsonOutput(raw, "gh aw experiments list --json");
      const experiments = Array.isArray(parsed) ? parsed : [];
      setCached("experiments", experiments);
      console.error(`${LOG2} getExperiments: fetched count=${experiments.length}`);
      return experiments;
    } catch (err) {
      console.error(`${LOG2} getExperiments error: ${asError(err).message}`);
      throw err;
    }
  }
  async function fetchLogsBatches(initialOptions, initialArgs = null) {
    let current = initialOptions;
    let logsFetches = 0;
    let runs = [];
    let continuation = null;
    let summary = null;
    let firstBatch = null;
    while (current && logsFetches < MAX_LOG_CONTINUATIONS) {
      let batchArgs = logsFetches === 0 && initialArgs ? initialArgs : buildLogsArgs(current);
      if (logsOutputDir && !hasFlag(batchArgs, "--output", "-o")) {
        batchArgs = [...batchArgs, "--output", logsOutputDir];
      }
      console.error(`${LOG2} fetchLogsBatches: batch=${logsFetches + 1} args=${JSON.stringify(batchArgs)}`);
      const raw = await runGhAw(batchArgs);
      let data;
      try {
        data = parseJsonOutput(raw, `logs batch ${logsFetches + 1}`);
      } catch (error) {
        console.error(`${LOG2} fetchLogsBatches: parse error on batch ${logsFetches + 1}: ${asError(error).message}`);
        throw asError(error);
      }
      if (!firstBatch) {
        firstBatch = data;
      }
      const newRuns = Array.isArray(data.runs) ? data.runs : [];
      runs = mergeRuns(runs, newRuns);
      continuation = data.continuation ?? null;
      summary = data.summary ?? summary;
      logsFetches += 1;
      console.error(`${LOG2} fetchLogsBatches: batch=${logsFetches} newRuns=${newRuns.length} totalRuns=${runs.length} hasContinuation=${Boolean(continuation)}`);
      if (!continuation) {
        break;
      }
      current = continuationToLogsOptions(continuation, current);
    }
    return {
      firstBatch,
      runs,
      summary,
      logsFetches,
      partial: Boolean(continuation),
      continuation
    };
  }
  async function getLogsData(options = {}) {
    const normalized = normalizeLogsOptions(options);
    const key = `logs:${JSON.stringify({
      window: normalized.window.id,
      count: normalized.count,
      timeout: normalized.timeout,
      startDate: normalized.startDate,
      endDate: normalized.endDate,
      beforeRunID: normalized.beforeRunID,
      afterRunID: normalized.afterRunID,
      workflowName: normalized.workflowName,
      engine: normalized.engine,
      branch: normalized.branch,
      artifacts: normalized.artifacts
    })}`;
    const hit = getCached(key);
    if (hit) {
      console.error(`${LOG2} getLogsData: cache hit runs=${hit.runs.length} window=${hit.window.id}`);
      return hit;
    }
    console.error(`${LOG2} getLogsData: fetching window=${normalized.window.id} count=${normalized.count} timeout=${normalized.timeout}`);
    const logsResult = await fetchLogsBatches(normalized);
    console.error(`${LOG2} getLogsData: fetched runs=${logsResult.runs.length} fetches=${logsResult.logsFetches} partial=${logsResult.partial}`);
    const result = {
      runs: logsResult.runs,
      summary: logsResult.summary,
      window: normalized.window,
      timeout: normalized.timeout,
      logsFetches: logsResult.logsFetches,
      partial: logsResult.partial,
      continuation: logsResult.continuation
    };
    setCached(key, result);
    return result;
  }
  async function getForecastData(workflowIDs, window, timeout) {
    if (workflowIDs.length === 0) {
      return [];
    }
    const args = ["forecast", "--json", "--period", "month", "--days", String(forecastDaysForWindow(window)), "--timeout", String(timeout), ...workflowIDs];
    console.error(`${LOG2} getForecastData: workflowIDs=${workflowIDs.length} window=${window.id} days=${forecastDaysForWindow(window)}`);
    try {
      const raw = await runGhAw(args);
      const data = parseJsonOutput(raw, "gh aw forecast --json");
      const workflows = Array.isArray(data.workflows) ? data.workflows : [];
      console.error(`${LOG2} getForecastData: fetched workflows=${workflows.length}`);
      return workflows;
    } catch (err) {
      console.error(`${LOG2} getForecastData error: ${asError(err).message}`);
      throw err;
    }
  }
  async function getRuns(options = {}) {
    return getLogsData(options);
  }
  async function getUsage(options = {}) {
    const normalized = normalizeLogsOptions(options);
    const key = `usage:${JSON.stringify({
      window: normalized.window.id,
      count: normalized.count,
      timeout: normalized.timeout
    })}`;
    const hit = getCached(key);
    if (hit) return hit;
    const logsData = await getLogsData(normalized);
    const usageItems = buildUsageSummary(logsData.runs, logsData.window);
    const workflowIDs = usageItems.map((item) => item.workflow_id).filter(Boolean);
    const forecastWorkflows = await getForecastData(workflowIDs, logsData.window, logsData.timeout);
    const result = {
      items: applyForecastToUsageSummary(usageItems, forecastWorkflows),
      window: logsData.window,
      timeout: logsData.timeout,
      logsFetches: logsData.logsFetches,
      partial: logsData.partial,
      continuation: logsData.continuation,
      total_runs: logsData.runs.length,
      forecast_history_days: forecastDaysForWindow(logsData.window)
    };
    setCached(key, result);
    return result;
  }
  async function execCommand(rawCmd, options = {}) {
    console.error(`${LOG2} execCommand: cmd="${rawCmd}"`);
    const args = parseGhAwArgs(rawCmd);
    if (!args) {
      console.error(`${LOG2} execCommand: rejected unsupported command "${rawCmd}"`);
      return { command: rawCmd, output: "Only 'gh aw <subcommand>' commands are supported.", error: true };
    }
    try {
      if (args[0] === "logs" && logsCommandUsesJSON(args)) {
        const commandArgs = normalizeLogsCommandArgs(args, options.window, options.timeout ?? DEFAULT_LOG_TIMEOUT_MINUTES);
        const fallback = {};
        if (options.window) {
          fallback.window = options.window;
        }
        if (options.timeout != null) {
          fallback.timeout = options.timeout;
        }
        const logsOptions = logsArgsToOptions(commandArgs, fallback);
        const logsResult = await fetchLogsBatches(logsOptions, commandArgs);
        return {
          command: `gh aw ${commandArgs.join(" ")}`,
          output: JSON.stringify(
            {
              ...logsResult.firstBatch ?? {},
              runs: logsResult.runs,
              partial: logsResult.partial,
              logs_fetches: logsResult.logsFetches,
              continuation: logsResult.continuation
            },
            null,
            2
          )
        };
      }
      const output = await runGhAw(args);
      return { command: rawCmd, output };
    } catch (err) {
      const e = asError(err);
      const msg = e.stderr || e.message || "Unknown error";
      console.error(`${LOG2} execCommand error cmd="${rawCmd}": ${msg}`);
      return { command: rawCmd, output: msg, error: true };
    }
  }
  async function getAudit(runId) {
    if (!runId) return null;
    const key = `audit:${runId}`;
    const hit = getCached(key);
    if (hit) {
      console.error(`${LOG2} getAudit: cache hit runId=${runId}`);
      return hit;
    }
    console.error(`${LOG2} getAudit: fetching runId=${runId}`);
    try {
      const auditArgs = ["audit", String(runId), "--json"];
      if (logsOutputDir) {
        auditArgs.push("--output", logsOutputDir);
      }
      const raw = await runGhAw(auditArgs);
      const data = parseJsonOutput(raw, `gh aw audit ${runId} --json`);
      setCached(key, data);
      console.error(`${LOG2} getAudit: fetched runId=${runId}`);
      return data;
    } catch (err) {
      console.error(`${LOG2} getAudit error runId=${runId}: ${asError(err).message}`);
      throw err;
    }
  }
  return {
    clearCache: () => cache.clear(),
    execCommand,
    getAudit,
    getDefinitions,
    getExperiments,
    getRuns,
    getUsage
  };
}
export {
  DEFAULT_LOG_TIMEOUT_MINUTES,
  DEFAULT_RUN_COUNT,
  createDashboardDataAccess,
  createGhAwRunnerWithStatus
};
