// extension.ts
import { createServer } from "node:http";
import { readFile } from "node:fs/promises";
import { dirname, join as join2 } from "node:path";
import { fileURLToPath } from "node:url";
import { createCanvas, joinSession } from "@github/copilot-sdk/extension";

// dashboard-cli.ts
import { spawn } from "node:child_process";
import { constants as fsConstants } from "node:fs";
import { access } from "node:fs/promises";
import { join } from "node:path";
var INSTALL_COMMAND = "gh extension install github/gh-aw";
var GH_INSTALL_URL = "https://cli.github.com";
function combineOutput(stdout, stderr) {
  return [stdout, stderr].filter(Boolean).join("\n").trim();
}
function buildCommandError(message, details = {}) {
  return Object.assign(new Error(message), details);
}
function spawnExecFile(file, args, options = {}, callback) {
  const { env, cwd, maxBuffer = 10 * 1024 * 1024 } = options;
  const proc = spawn(file, args, { env, cwd, stdio: ["ignore", "pipe", "pipe"], detached: true });
  const stdoutChunks = [];
  const stderrChunks = [];
  let stdoutLen = 0;
  let stderrLen = 0;
  let overflowed = false;
  proc.stdout?.on("data", chunk => {
    const buffer = Buffer.isBuffer(chunk) ? chunk : Buffer.from(String(chunk));
    stdoutLen += buffer.length;
    if (stdoutLen > maxBuffer) {
      overflowed = true;
      return;
    }
    stdoutChunks.push(buffer);
  });
  proc.stderr?.on("data", chunk => {
    const buffer = Buffer.isBuffer(chunk) ? chunk : Buffer.from(String(chunk));
    stderrLen += buffer.length;
    if (stderrLen > maxBuffer) {
      overflowed = true;
      return;
    }
    stderrChunks.push(buffer);
  });
  proc.on("error", error => callback(error, "", ""));
  proc.on("close", code => {
    const stdout = Buffer.concat(stdoutChunks).toString("utf8");
    const stderr = Buffer.concat(stderrChunks).toString("utf8");
    if (overflowed) {
      callback(buildCommandError("stdout/stderr maxBuffer exceeded", { code: "ERR_CHILD_PROCESS_STDIO_MAXBUFFER" }), stdout, stderr);
      return;
    }
    if (code !== 0) {
      callback(buildCommandError(`Command failed with exit code ${code ?? "unknown"}`, { code: code ?? "unknown" }), stdout, stderr);
      return;
    }
    callback(null, stdout, stderr);
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
        maxBuffer: 10 * 1024 * 1024,
      },
      (error, stdout, stderr) => {
        const output = combineOutput(stdout, stderr);
        if (error) {
          reject(Object.assign(error, { stderr, stdout, output }));
          return;
        }
        resolve(combineIO ? output : stdout);
      }
    );
  });
}
function parseVersionFromOutput(output) {
  const trimmed = String(output).trim();
  if (!trimmed) return "";
  const match = trimmed.match(/gh(?:-aw| aw) version ([^\r\n]+)/i);
  return match?.[1]?.trim() ?? "";
}
function isMissingGh(error) {
  return typeof error === "object" && error !== null && error.code === "ENOENT" && error.syscall === "spawn" && error.path === "gh";
}
function isMissingGhAwExtension(error) {
  const output = typeof error === "object" && error !== null ? String(error.output ?? error.stderr ?? error.message ?? "") : "";
  return /extension not found:\s*aw/i.test(output) || /unknown command ["']aw["'] for ["']gh["']/i.test(output);
}
async function findDevBinary(cwd, accessFn = access, platform = process.platform) {
  const devBin = join(cwd, platform === "win32" ? "gh-aw.exe" : "gh-aw");
  try {
    await accessFn(devBin, fsConstants.X_OK);
    return devBin;
  } catch {
    return null;
  }
}
function createGhAwRunner({ getWorkspacePath, accessFn = access, execFileFn = spawnExecFile, platform = process.platform, env = process.env }) {
  async function runExec(bin, args, cwd, options) {
    return execp(bin, args, cwd, { ...options, execFileFn, env });
  }
  return async function runGhAw2(args) {
    const cwd = getWorkspacePath();
    const devBin = await findDevBinary(cwd, accessFn, platform);
    if (devBin) {
      return runExec(devBin, args, cwd);
    }
    return runExec("gh", ["aw", ...args], cwd);
  };
}
function createGhAwRunnerWithStatus(options) {
  const runGhAw2 = createGhAwRunner(options);
  runGhAw2.getStatus = async () => {
    const cwd = options.getWorkspacePath();
    const devBin = await findDevBinary(cwd, options.accessFn ?? access, options.platform ?? process.platform);
    if (devBin) {
      const output = await execp(devBin, ["version"], cwd, {
        combineIO: true,
        execFileFn: options.execFileFn ?? spawnExecFile,
        env: options.env ?? process.env,
      });
      return {
        available: true,
        source: "dev-binary",
        version: parseVersionFromOutput(output) || "unknown",
        command: `${devBin} version`,
        installCommand: INSTALL_COMMAND,
      };
    }
    try {
      const output = await execp("gh", ["aw", "version"], cwd, {
        combineIO: true,
        execFileFn: options.execFileFn ?? spawnExecFile,
        env: options.env ?? process.env,
      });
      return {
        available: true,
        source: "gh-extension",
        version: parseVersionFromOutput(output) || "unknown",
        command: "gh aw version",
        installCommand: INSTALL_COMMAND,
      };
    } catch (error) {
      if (isMissingGh(error)) {
        return {
          available: false,
          source: "gh-not-found",
          version: "",
          command: "gh aw version",
          installCommand: INSTALL_COMMAND,
          installUrl: GH_INSTALL_URL,
          message: "Install the GitHub CLI to use this dashboard.",
        };
      }
      if (isMissingGhAwExtension(error)) {
        return {
          available: false,
          source: "missing",
          version: "",
          command: "gh aw version",
          installCommand: INSTALL_COMMAND,
          message: "gh aw is not installed. Install the GitHub CLI extension to use the dashboard outside a local dev build.",
        };
      }
      const output = typeof error === "object" && error !== null ? String(error.output ?? error.stderr ?? error.message ?? "Failed to detect gh aw.") : "Failed to detect gh aw.";
      return {
        available: false,
        source: "error",
        version: "",
        command: "gh aw version",
        installCommand: INSTALL_COMMAND,
        message: output,
      };
    }
  };
  return runGhAw2;
}

// dashboard-config.ts
var CACHE_TTL_MS = 6e4;
var DEFAULT_LOG_TIMEOUT_MINUTES = 1;
var DEFAULT_REPORT_WINDOW_ID = "7d";
var DEFAULT_RUN_COUNT = 100;
var MAX_LOG_CONTINUATIONS = 6;
var REPORT_WINDOWS = {
  "3d": { id: "3d", label: "3 days", startDate: "-3d", days: 3 },
  "7d": { id: "7d", label: "7 days", startDate: "-1w", days: 7 },
  "1mo": { id: "1mo", label: "1 month", startDate: "-1mo", days: 30 },
};
function getReportWindow(windowId) {
  return REPORT_WINDOWS[windowId] ?? REPORT_WINDOWS[DEFAULT_REPORT_WINDOW_ID];
}

// dashboard-logs.ts
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
    artifacts,
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
    artifacts: fallback.artifacts,
  });
}
function mergeRuns(existingRuns, nextRuns) {
  const merged = new Map(existingRuns.map(run => [run.run_id, run]));
  for (const run of nextRuns) {
    if (run?.run_id != null) {
      merged.set(run.run_id, run);
    }
  }
  return Array.from(merged.values()).sort((a, b) => Number(b.run_id ?? 0) - Number(a.run_id ?? 0));
}
function parseGhAwArgs(raw) {
  const match = raw.trim().match(/^(?:gh\s+aw\s+)(.+)$/);
  const subcommand = match?.[1];
  return subcommand ? subcommand.trim().split(/\s+/) : null;
}
function hasFlag(args, longFlag, shortFlag = "") {
  return args.some(arg => {
    if (arg.startsWith(`${longFlag}=`)) {
      return true;
    }
    if (shortFlag && arg.startsWith(`${shortFlag}=`)) {
      return true;
    }
    return arg === longFlag || (shortFlag !== "" && arg === shortFlag);
  });
}
function logsCommandUsesJSON(args) {
  return hasFlag(args, "--json", "-j");
}
function normalizeLogsCommandArgs(args, windowId, timeoutMinutes = DEFAULT_LOG_TIMEOUT_MINUTES) {
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
  const options = {};
  const fallbackWindow = typeof fallback.window === "string" ? fallback.window : fallback.window?.id;
  if (fallbackWindow) options.window = fallbackWindow;
  if (fallback.count !== void 0) options.count = fallback.count;
  if (fallback.timeout !== void 0) options.timeout = fallback.timeout;
  if (fallback.startDate !== void 0) options.startDate = fallback.startDate;
  if (fallback.endDate !== void 0) options.endDate = fallback.endDate;
  if (fallback.beforeRunID !== void 0) options.beforeRunID = fallback.beforeRunID;
  if (fallback.afterRunID !== void 0) options.afterRunID = fallback.afterRunID;
  if (fallback.workflowName !== void 0) options.workflowName = fallback.workflowName;
  if (fallback.engine !== void 0) options.engine = fallback.engine;
  if (fallback.branch !== void 0) options.branch = fallback.branch;
  if (fallback.artifacts !== void 0) options.artifacts = fallback.artifacts;
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
      options.artifacts = value
        .split(",")
        .map(item => item.trim())
        .filter(Boolean);
      index = nextIndex;
    }
  }
  return normalizeLogsOptions(options);
}

// usage-forecast.ts
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
  const forecastEntries = forecastWorkflows.map(forecast => [normalizeWorkflowID(forecast.workflow_id || forecast.workflow_path), getForecastMonthlyAIC(forecast)]).filter(([workflowID]) => Boolean(workflowID));
  const forecastByWorkflow = new Map(forecastEntries);
  return items.map(item => ({
    ...item,
    monthly_forecast_aic: forecastByWorkflow.get(item.workflow_id) ?? 0,
  }));
}
function buildUsageSummary(runs, window, forecastWorkflows = []) {
  const usageByWorkflow = /* @__PURE__ */ new Map();
  const effectiveDays = Number(window.days ?? 0);
  if (!Number.isFinite(effectiveDays) || effectiveDays <= 0) {
    throw new Error(`report window '${window.id ?? "unknown"}' is missing a valid positive day count.`);
  }
  for (const run of runs) {
    const workflowPath = typeof run.workflow_path === "string" ? run.workflow_path.trim() : "";
    const workflowID = normalizeWorkflowID(workflowPath || run.workflow_name);
    if (!workflowID) continue;
    const workflowName = String(run.workflow_name ?? workflowID).trim() || workflowID;
    const aic = toNumber(run.aic);
    const entry = usageByWorkflow.get(workflowID) ?? {
      workflow_id: workflowID,
      workflow_name: workflowName,
      workflow_path: workflowPath,
      run_count: 0,
      total_aic: 0,
      cost_per_run: 0,
      daily_aic: 0,
      monthly_forecast_aic: 0,
      last_run_at: "",
    };
    entry.run_count += 1;
    entry.total_aic += aic;
    if (!entry.workflow_path && workflowPath) {
      entry.workflow_path = workflowPath;
    }
    if (!entry.workflow_name && workflowName) {
      entry.workflow_name = workflowName;
    }
    const createdAt = typeof run.created_at === "string" ? run.created_at : "";
    if (createdAt && (!entry.last_run_at || createdAt > entry.last_run_at)) {
      entry.last_run_at = createdAt;
    }
    usageByWorkflow.set(workflowID, entry);
  }
  const items = Array.from(usageByWorkflow.values())
    .map(entry => {
      const costPerRun = entry.run_count > 0 ? entry.total_aic / entry.run_count : 0;
      const dailyAIC = entry.total_aic / effectiveDays;
      return {
        ...entry,
        cost_per_run: costPerRun,
        daily_aic: dailyAIC,
        monthly_forecast_aic: 0,
      };
    })
    .sort((a, b) => {
      const dailyDelta = b.daily_aic - a.daily_aic;
      if (dailyDelta !== 0) return dailyDelta;
      return b.cost_per_run - a.cost_per_run;
    });
  return applyForecastToUsageSummary(items, forecastWorkflows);
}

// dashboard-data.ts
function parseJSON(raw, errorPrefix, snippetLength = 200) {
  try {
    return JSON.parse(raw);
  } catch (error) {
    const snippet = String(raw ?? "")
      .replace(/\s+/g, " ")
      .slice(0, snippetLength);
    const message = error instanceof Error ? error.message : String(error);
    throw new Error(`${errorPrefix}: ${message}${snippet ? ` (output: ${snippet})` : ""}`);
  }
}
function getErrorMessage(error) {
  if (typeof error === "object" && error !== null) {
    const commandError = error;
    return commandError.stderr || commandError.message || "Unknown error";
  }
  return String(error ?? "Unknown error");
}
function createDashboardDataAccess({ runGhAw: runGhAw2, cacheTTL = CACHE_TTL_MS }) {
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
    if (hit) return hit;
    const raw = await runGhAw2(["status", "--json"]);
    const data = parseJSON(raw, "Failed to parse workflow definitions");
    setCached("definitions", data);
    return data;
  }
  async function getExperiments() {
    const hit = getCached("experiments");
    if (hit) return hit;
    const raw = await runGhAw2(["experiments", "list", "--json"]);
    const data = parseJSON(raw, "Failed to parse experiments list");
    const experiments = Array.isArray(data) ? data : [];
    setCached("experiments", experiments);
    return experiments;
  }
  async function fetchLogsBatches(initialOptions, initialArgs = null) {
    let current = initialOptions;
    let logsFetches = 0;
    let runs = [];
    let continuation = null;
    let summary = null;
    let firstBatch = null;
    while (current && logsFetches < MAX_LOG_CONTINUATIONS) {
      const raw = await runGhAw2(logsFetches === 0 && initialArgs ? initialArgs : buildLogsArgs(current));
      const data = parseJSON(raw, `Failed to parse logs batch ${logsFetches + 1}`);
      if (!firstBatch) {
        firstBatch = data;
      }
      runs = mergeRuns(runs, Array.isArray(data.runs) ? data.runs : []);
      continuation = data.continuation ?? null;
      summary = data.summary ?? summary;
      logsFetches += 1;
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
      continuation,
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
      artifacts: normalized.artifacts,
    })}`;
    const hit = getCached(key);
    if (hit) return hit;
    const logsResult = await fetchLogsBatches(normalized);
    const result = {
      runs: logsResult.runs,
      summary: logsResult.summary,
      window: normalized.window,
      timeout: normalized.timeout,
      logsFetches: logsResult.logsFetches,
      partial: logsResult.partial,
      continuation: logsResult.continuation,
    };
    setCached(key, result);
    return result;
  }
  async function getForecastData(workflowIDs, window, timeout) {
    if (workflowIDs.length === 0) {
      return [];
    }
    const args = ["forecast", "--json", "--period", "month", "--days", String(forecastDaysForWindow(window)), "--timeout", String(timeout), ...workflowIDs];
    const raw = await runGhAw2(args);
    const data = parseJSON(raw, "Failed to parse forecast output");
    return Array.isArray(data.workflows) ? data.workflows : [];
  }
  async function getRuns(options = {}) {
    return getLogsData(options);
  }
  async function getUsage(options = {}) {
    const normalized = normalizeLogsOptions(options);
    const key = `usage:${JSON.stringify({
      window: normalized.window.id,
      count: normalized.count,
      timeout: normalized.timeout,
    })}`;
    const hit = getCached(key);
    if (hit) return hit;
    const logsData = await getLogsData(normalized);
    const usageItems = buildUsageSummary(logsData.runs, logsData.window);
    const workflowIDs = usageItems.map(item => item.workflow_id).filter(Boolean);
    const forecastWorkflows = await getForecastData(workflowIDs, logsData.window, logsData.timeout);
    const result = {
      items: applyForecastToUsageSummary(usageItems, forecastWorkflows),
      window: logsData.window,
      timeout: logsData.timeout,
      logsFetches: logsData.logsFetches,
      partial: logsData.partial,
      continuation: logsData.continuation,
      total_runs: logsData.runs.length,
      forecast_history_days: forecastDaysForWindow(logsData.window),
    };
    setCached(key, result);
    return result;
  }
  async function execCommand(rawCmd, options = {}) {
    const args = parseGhAwArgs(rawCmd);
    if (!args) {
      return { command: rawCmd, output: "Only 'gh aw <subcommand>' commands are supported.", error: true };
    }
    try {
      if (args[0] === "logs" && logsCommandUsesJSON(args)) {
        const commandArgs = normalizeLogsCommandArgs(args, options.window, options.timeout ?? DEFAULT_LOG_TIMEOUT_MINUTES);
        const logsFallback = {};
        if (options.window) logsFallback.window = options.window;
        if (options.timeout != null) logsFallback.timeout = options.timeout;
        const logsOptions = logsArgsToOptions(commandArgs, logsFallback);
        const logsResult = await fetchLogsBatches(logsOptions, commandArgs);
        return {
          command: `gh aw ${commandArgs.join(" ")}`,
          output: JSON.stringify(
            {
              ...(logsResult.firstBatch ?? {}),
              runs: logsResult.runs,
              partial: logsResult.partial,
              logs_fetches: logsResult.logsFetches,
              continuation: logsResult.continuation,
            },
            null,
            2
          ),
        };
      }
      const output = await runGhAw2(args);
      return { command: rawCmd, output };
    } catch (error) {
      return { command: rawCmd, output: getErrorMessage(error), error: true };
    }
  }
  async function getAudit(runId) {
    if (!runId) return null;
    const key = `audit:${runId}`;
    const hit = getCached(key);
    if (hit) return hit;
    const raw = await runGhAw2(["audit", String(runId), "--json"]);
    const data = parseJSON(raw, `Failed to parse audit output for run ${runId}`, 100);
    setCached(key, data);
    return data;
  }
  return {
    clearCache: () => cache.clear(),
    execCommand,
    getAudit,
    getDefinitions,
    getExperiments,
    getRuns,
    getUsage,
  };
}

// src/pagination.ts
function paginate(items, page, pageSize) {
  const totalItems = items.length;
  const totalPages = Math.max(1, Math.ceil(totalItems / pageSize));
  const safePage = Math.min(Math.max(1, page), totalPages);
  const start = (safePage - 1) * pageSize;
  const end = start + pageSize;
  return {
    items: items.slice(start, end),
    page: safePage,
    pageSize,
    totalItems,
    totalPages,
    hasNextPage: safePage < totalPages,
    hasPreviousPage: safePage > 1,
  };
}

// extension.ts
var __dirname = dirname(fileURLToPath(import.meta.url));
var servers = /* @__PURE__ */ new Map();
var workspacePath = process.cwd();
var runGhAw = createGhAwRunnerWithStatus({ getWorkspacePath: () => workspacePath });
var dataAccess = createDashboardDataAccess({ runGhAw });
function sanitizePayload(payload) {
  if (payload instanceof Error) {
    return { error: "Request failed." };
  }
  if (payload === null) {
    return null;
  }
  if (typeof payload === "boolean" || typeof payload === "number" || typeof payload === "string") {
    return payload;
  }
  if (Array.isArray(payload)) {
    return payload.map(item => sanitizePayload(item));
  }
  if (typeof payload === "object") {
    return Object.fromEntries(
      Object.entries(payload)
        .filter(([key]) => key !== "stack")
        .map(([key, value]) => [key, sanitizePayload(value)])
    );
  }
  return String(payload);
}
function parseQueryInt(value, fallback) {
  const parsed = Number.parseInt(value ?? "", 10);
  return Number.isFinite(parsed) ? parsed : fallback;
}
async function startServer() {
  const server = createServer(async (req, res) => {
    const reqUrl = new URL(req.url ?? "/", "http://localhost");
    const pathname = reqUrl.pathname;
    const sendJson = (payload, status = 200) => {
      res.writeHead(status, { "Content-Type": "application/json; charset=utf-8" });
      res.end(JSON.stringify(payload));
    };
    try {
      if (pathname === "/" || pathname === "/index.html") {
        const [html, css] = await Promise.all([readFile(join2(__dirname, "web", "index.html"), "utf8"), readFile(join2(__dirname, "web", "styles.css"), "utf8")]);
        res.setHeader("Content-Type", "text/html; charset=utf-8");
        res.end(html.replace("/*__APP_CSS__*/", css));
      } else if (pathname === "/app.js") {
        res.setHeader("Content-Type", "application/javascript; charset=utf-8");
        res.end(await readFile(join2(__dirname, "web", "app.js"), "utf8"));
      } else if (pathname === "/api/status") {
        sendJson(sanitizePayload(await dataAccess.getDefinitions()));
      } else if (pathname === "/api/cli-status") {
        sendJson(sanitizePayload(await runGhAw.getStatus()));
      } else if (pathname === "/api/experiments") {
        sendJson(sanitizePayload(await dataAccess.getExperiments()));
      } else if (pathname === "/api/runs") {
        sendJson(
          sanitizePayload(
            await dataAccess.getRuns({
              count: parseQueryInt(reqUrl.searchParams.get("count"), DEFAULT_RUN_COUNT),
              window: reqUrl.searchParams.get("window") ?? "7d",
              timeout: parseQueryInt(reqUrl.searchParams.get("timeout"), DEFAULT_LOG_TIMEOUT_MINUTES),
            })
          )
        );
      } else if (pathname === "/api/usage") {
        sendJson(
          sanitizePayload(
            await dataAccess.getUsage({
              count: parseQueryInt(reqUrl.searchParams.get("count"), DEFAULT_RUN_COUNT),
              window: reqUrl.searchParams.get("window") ?? "7d",
              timeout: parseQueryInt(reqUrl.searchParams.get("timeout"), DEFAULT_LOG_TIMEOUT_MINUTES),
            })
          )
        );
      } else if (pathname === "/api/audit") {
        const runId = reqUrl.searchParams.get("run_id") ?? "";
        if (!runId) {
          sendJson({ error: "run_id is required" }, 400);
        } else {
          sendJson(sanitizePayload(await dataAccess.getAudit(runId)));
        }
      } else if (pathname === "/api/run-command") {
        const cmd = reqUrl.searchParams.get("cmd") ?? "";
        sendJson(
          sanitizePayload(
            await dataAccess.execCommand(cmd, {
              window: reqUrl.searchParams.get("window") ?? "7d",
              timeout: parseQueryInt(reqUrl.searchParams.get("timeout"), DEFAULT_LOG_TIMEOUT_MINUTES),
            })
          )
        );
      } else if (pathname === "/api/refresh") {
        dataAccess.clearCache();
        sendJson({ ok: true });
      } else {
        res.writeHead(404);
        res.end("Not found");
      }
    } catch {
      sendJson({ error: "Request failed." }, 500);
    }
  });
  await new Promise(resolve => server.listen(0, "127.0.0.1", resolve));
  const address = server.address();
  if (!address || typeof address === "string") {
    throw new Error("Failed to determine loopback server address.");
  }
  return { server, url: `http://127.0.0.1:${address.port}/` };
}
function paginateDefinitions(definitions, page, pageSize) {
  return paginate(definitions, page, pageSize);
}
function paginateRuns(runs, page, pageSize) {
  return paginate(runs, page, pageSize);
}
function paginateUsage(items, page, pageSize) {
  return paginate(items, page, pageSize);
}
function paginateExperiments(items, page, pageSize) {
  return paginate(items, page, pageSize);
}
var session = await joinSession({
  systemMessage: {
    mode: "append",
    content: `## Agentic Workflows Dashboard

This canvas shows live data from the current repository using the gh-aw CLI.
It never calls Go code directly \u2014 all data is fetched by running CLI subcommands.

**CLI commands used by this canvas:**
- \`gh aw status --json\` \u2014 list agentic workflow definitions (workflow, engine_id, compiled, labels, status, time_remaining)
- \`gh aw logs --json -c <N> --start-date <window> --timeout <minutes>\` \u2014 list recent workflow runs and follow continuation batches progressively
- \`gh aw experiments list --json\` \u2014 list experiment workflow branches (workflow_id, branch, experiments, total_runs, last_run)

**Dev build** (when gh-aw is not installed as a gh extension):
1. Run \`make build\` in the repository root to compile \`./gh-aw\` (or \`./gh-aw.exe\` on Windows)
2. The canvas auto-detects the dev binary and uses it before falling back to \`gh aw\`

**Canvas actions available to the agent:**
- \`listDefinitions\` \u2014 calls \`gh aw status --json\`, returns paged results
- \`listRuns\` \u2014 calls \`gh aw logs --json\` with a selected report window, timeout, and continuation handling
- \`listUsage\` \u2014 aggregates workflow AIC usage from logs and fills monthly forecast via \`gh aw forecast --json\`
- \`listExperiments\` \u2014 calls \`gh aw experiments list --json\`, returns paged results
- \`getRun\` \u2014 looks up a single run by \`run_id\`
- \`auditRun\` \u2014 calls \`gh aw audit <run_id> --json\` and returns structured audit data (overview, metrics, key_findings, recommendations, jobs, tool_usage, errors, warnings, firewall_analysis)
- \`runCommand\` \u2014 executes any \`gh aw <subcommand>\` and returns stdout
- \`refresh\` \u2014 clears the 60-second cache so the next call fetches fresh data
`,
  },
  canvases: [
    createCanvas({
      id: "agentic-workflows-dashboard",
      displayName: "Agentic Workflows Dashboard",
      description: "Live dashboard for agentic workflow definitions and runs, powered by gh aw status and gh aw logs.",
      actions: [
        {
          name: "listDefinitions",
          description: "List workflow definitions via gh aw status --json, with paging.",
          inputSchema: {
            type: "object",
            properties: {
              page: { type: "number", minimum: 1 },
              pageSize: { type: "number", minimum: 1, maximum: 100 },
            },
            additionalProperties: false,
          },
          handler: async ctx => {
            const input = ctx.input ?? {};
            const defs = await dataAccess.getDefinitions();
            return paginateDefinitions(defs, Number(input.page ?? 1), Number(input.pageSize ?? 20));
          },
        },
        {
          name: "listRuns",
          description: "List recent workflow runs via gh aw logs --json, with paging and continuation handling.",
          inputSchema: {
            type: "object",
            properties: {
              page: { type: "number", minimum: 1 },
              pageSize: { type: "number", minimum: 1, maximum: 100 },
              count: { type: "number", minimum: 1, maximum: 200, description: "Max runs to fetch from the CLI." },
              window: { type: "string", enum: ["3d", "7d", "1mo"], description: "Report window preset for gh aw logs." },
              timeout: { type: "number", minimum: 1, maximum: 10, description: "Per-request timeout in minutes for progressive logs retrieval." },
            },
            additionalProperties: false,
          },
          handler: async ctx => {
            const input = ctx.input ?? {};
            const logsData = await dataAccess.getRuns({
              count: Number(input.count ?? DEFAULT_RUN_COUNT),
              window: String(input.window ?? "7d"),
              timeout: Number(input.timeout ?? DEFAULT_LOG_TIMEOUT_MINUTES),
            });
            return {
              ...paginateRuns(logsData.runs, Number(input.page ?? 1), Number(input.pageSize ?? 20)),
              partial: logsData.partial,
              logsFetches: logsData.logsFetches,
              window: logsData.window,
            };
          },
        },
        {
          name: "listUsage",
          description: "Aggregate workflow AIC usage from gh aw logs and monthly forecast costs from gh aw forecast.",
          inputSchema: {
            type: "object",
            properties: {
              page: { type: "number", minimum: 1 },
              pageSize: { type: "number", minimum: 1, maximum: 100 },
              count: { type: "number", minimum: 1, maximum: 200, description: "Max runs to fetch from the CLI." },
              window: { type: "string", enum: ["3d", "7d", "1mo"], description: "Report window preset for gh aw logs." },
              timeout: { type: "number", minimum: 1, maximum: 10, description: "Per-request timeout in minutes for progressive logs retrieval." },
            },
            additionalProperties: false,
          },
          handler: async ctx => {
            const input = ctx.input ?? {};
            const usage = await dataAccess.getUsage({
              count: Number(input.count ?? DEFAULT_RUN_COUNT),
              window: String(input.window ?? "7d"),
              timeout: Number(input.timeout ?? DEFAULT_LOG_TIMEOUT_MINUTES),
            });
            return {
              ...paginateUsage(usage.items, Number(input.page ?? 1), Number(input.pageSize ?? 20)),
              partial: usage.partial,
              logsFetches: usage.logsFetches,
              totalRuns: usage.total_runs,
              window: usage.window,
            };
          },
        },
        {
          name: "listExperiments",
          description: "List experiment workflow branches via gh aw experiments list --json, with paging.",
          inputSchema: {
            type: "object",
            properties: {
              page: { type: "number", minimum: 1 },
              pageSize: { type: "number", minimum: 1, maximum: 100 },
            },
            additionalProperties: false,
          },
          handler: async ctx => {
            const input = ctx.input ?? {};
            const experiments = await dataAccess.getExperiments();
            return paginateExperiments(experiments, Number(input.page ?? 1), Number(input.pageSize ?? 20));
          },
        },
        {
          name: "getRun",
          description: "Get a single workflow run by its run_id.",
          inputSchema: {
            type: "object",
            required: ["run_id"],
            properties: { run_id: { type: "number" } },
            additionalProperties: false,
          },
          handler: async ctx => {
            const input = ctx.input ?? {};
            const logsData = await dataAccess.getRuns({ count: 200, window: "1mo", timeout: DEFAULT_LOG_TIMEOUT_MINUTES });
            return { run: logsData.runs.find(run => run.run_id === Number(input.run_id)) ?? null };
          },
        },
        {
          name: "auditRun",
          description: "Run gh aw audit for a specific workflow run by run_id, returning structured audit data.",
          inputSchema: {
            type: "object",
            required: ["run_id"],
            properties: { run_id: { type: "string", description: "The workflow run ID to audit (numeric string)." } },
            additionalProperties: false,
          },
          handler: async ctx => {
            const input = ctx.input ?? {};
            const runId = String(input.run_id ?? "").trim();
            if (!runId || !/^\d+$/.test(runId)) {
              throw new Error("run_id must be a non-empty numeric string");
            }
            return dataAccess.getAudit(runId);
          },
        },
        {
          name: "runCommand",
          description: "Execute a gh aw subcommand (e.g. 'gh aw status', 'gh aw logs -c 5') and return its stdout.",
          inputSchema: {
            type: "object",
            required: ["command"],
            properties: { command: { type: "string", description: "Full command string starting with 'gh aw'." } },
            additionalProperties: false,
          },
          handler: async ctx => {
            const input = ctx.input ?? {};
            return dataAccess.execCommand(String(input.command ?? ""), { window: "7d", timeout: DEFAULT_LOG_TIMEOUT_MINUTES });
          },
        },
        {
          name: "refresh",
          description: "Clear the data cache so the next listDefinitions/listRuns fetches fresh data from the CLI.",
          inputSchema: { type: "object", additionalProperties: false },
          handler: () => {
            dataAccess.clearCache();
            return { ok: true };
          },
        },
      ],
      open: async ctx => {
        let entry = servers.get(ctx.instanceId);
        if (!entry) {
          entry = await startServer();
          servers.set(ctx.instanceId, entry);
        }
        return { title: "Agentic Workflows Dashboard", status: "Live \xB7 gh aw", url: entry.url };
      },
      onClose: async ctx => {
        const entry = servers.get(ctx.instanceId);
        if (entry) {
          servers.delete(ctx.instanceId);
          await new Promise(resolve => entry.server.close(() => resolve()));
        }
      },
    }),
  ],
});
workspacePath = session.workspacePath ?? process.cwd();
