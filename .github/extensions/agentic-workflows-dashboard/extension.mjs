import { createServer } from "node:http";
import { execFile } from "node:child_process";
import { access, readFile } from "node:fs/promises";
import { constants as fsConstants } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";

import { createCanvas, joinSession } from "@github/copilot-sdk/extension";

const __dirname = dirname(fileURLToPath(import.meta.url));
const servers = new Map();
const cache = new Map(); // key → { data, expiresAt }
const CACHE_TTL_MS = 60_000;
const DEFAULT_LOG_TIMEOUT_MINUTES = 1;
const DEFAULT_RUN_COUNT = 100;
const MAX_LOG_CONTINUATIONS = 6;
const REPORT_WINDOWS = {
  "3d": { id: "3d", label: "3 days", startDate: "-3d", days: 3 },
  "7d": { id: "7d", label: "7 days", startDate: "-1w", days: 7 },
  "1mo": { id: "1mo", label: "1 month", startDate: "-1mo", days: 30 },
};
let workspacePath = process.cwd();

// ---------------------------------------------------------------------------
// CLI helpers
// ---------------------------------------------------------------------------

function execp(bin, args, cwd) {
  return new Promise((resolve, reject) => {
    execFile(
      bin,
      args,
      {
        cwd,
        env: { ...process.env, NO_COLOR: "1", GH_NO_UPDATE_NOTIFIER: "1" },
        maxBuffer: 10 * 1024 * 1024,
      },
      (err, stdout, stderr) => {
        if (err) reject(Object.assign(err, { stderr: stderr ?? "" }));
        else resolve(stdout);
      }
    );
  });
}

async function runGhAw(args) {
  const cwd = workspacePath;
  const isWin = process.platform === "win32";
  const devBin = join(cwd, isWin ? "gh-aw.exe" : "gh-aw");
  try {
    await access(devBin, fsConstants.X_OK);
    return await execp(devBin, args, cwd);
  } catch {
    return await execp("gh", ["aw", ...args], cwd);
  }
}

// ---------------------------------------------------------------------------
// Cache
// ---------------------------------------------------------------------------

function getCached(key) {
  const entry = cache.get(key);
  return entry && Date.now() < entry.expiresAt ? entry.data : null;
}
function setCached(key, data) {
  cache.set(key, { data, expiresAt: Date.now() + CACHE_TTL_MS });
}

// ---------------------------------------------------------------------------
// Data fetchers — both call the CLI, never Go code
// ---------------------------------------------------------------------------

async function getDefinitions() {
  const hit = getCached("definitions");
  if (hit) return hit;
  const raw = await runGhAw(["status", "--json"]);
  const data = JSON.parse(raw);
  setCached("definitions", data);
  return data;
}

async function getExperiments() {
  const hit = getCached("experiments");
  if (hit) return hit;
  const raw = await runGhAw(["experiments", "list", "--json"]);
  const data = JSON.parse(raw);
  const experiments = Array.isArray(data) ? data : [];
  setCached("experiments", experiments);
  return experiments;
}

function getReportWindow(windowId) {
  return REPORT_WINDOWS[windowId] ?? REPORT_WINDOWS["7d"];
}

function normalizeLogsOptions(options = {}) {
  const window = getReportWindow(options.window);
  const count = Number.parseInt(String(options.count ?? DEFAULT_RUN_COUNT), 10);
  const timeout = Number.parseInt(String(options.timeout ?? DEFAULT_LOG_TIMEOUT_MINUTES), 10);
  const artifacts = Array.isArray(options.artifacts) && options.artifacts.length > 0 ? options.artifacts : ["usage"];

  return {
    window,
    count: Number.isFinite(count) && count > 0 ? count : DEFAULT_RUN_COUNT,
    timeout: Number.isFinite(timeout) && timeout > 0 ? timeout : DEFAULT_LOG_TIMEOUT_MINUTES,
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

async function fetchLogsBatches(initialOptions, initialArgs = null) {
  let current = initialOptions;
  let logsFetches = 0;
  let runs = [];
  let continuation = null;
  let summary = null;
  let firstBatch = null;

  while (current && logsFetches < MAX_LOG_CONTINUATIONS) {
    const raw = await runGhAw(logsFetches === 0 && initialArgs ? initialArgs : buildLogsArgs(current));
    let data;
    try {
      data = JSON.parse(raw);
    } catch (error) {
      throw new Error(`Failed to parse logs batch ${logsFetches + 1}: ${error.message}`);
    }
    if (!firstBatch) {
      firstBatch = data;
    }
    runs = mergeRuns(runs, Array.isArray(data?.runs) ? data.runs : []);
    continuation = data?.continuation ?? null;
    summary = data?.summary ?? summary;
    logsFetches += 1;

    if (!continuation) {
      break;
    }

    current = continuationToLogsOptions(continuation, initialOptions);
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

async function getRuns(options = {}) {
  return getLogsData(options);
}

function toNumber(value) {
  const numeric = Number(value ?? 0);
  return Number.isFinite(numeric) ? numeric : 0;
}

function buildUsageSummary(runs, window) {
  const usageByWorkflow = new Map();
  const effectiveDays = Number(window?.days ?? 0);
  if (!Number.isFinite(effectiveDays) || effectiveDays <= 0) {
    throw new Error(`report window '${window?.id ?? "unknown"}' is missing a valid positive day count.`);
  }

  for (const run of runs) {
    const workflowName = String(run?.workflow_name ?? "").trim();
    if (!workflowName) continue;

    const aic = toNumber(run?.aic);
    const entry = usageByWorkflow.get(workflowName) ?? {
      workflow_name: workflowName,
      run_count: 0,
      total_aic: 0,
      cost_per_run: 0,
      daily_aic: 0,
      monthly_forecast_aic: 0,
      last_run_at: "",
    };

    entry.run_count += 1;
    entry.total_aic += aic;
    const createdAt = typeof run?.created_at === "string" ? run.created_at : "";
    if (createdAt && (!entry.last_run_at || createdAt > entry.last_run_at)) {
      entry.last_run_at = createdAt;
    }

    usageByWorkflow.set(workflowName, entry);
  }

  return Array.from(usageByWorkflow.values())
    .map(entry => {
      const costPerRun = entry.run_count > 0 ? entry.total_aic / entry.run_count : 0;
      const dailyAIC = entry.total_aic / effectiveDays;
      return {
        ...entry,
        cost_per_run: costPerRun,
        daily_aic: dailyAIC,
        monthly_forecast_aic: dailyAIC * 30,
      };
    })
    .sort((a, b) => {
      const dailyDelta = b.daily_aic - a.daily_aic;
      if (dailyDelta !== 0) return dailyDelta;
      return b.cost_per_run - a.cost_per_run;
    });
}

async function getUsage(options = {}) {
  const logsData = await getLogsData(options);
  return {
    items: buildUsageSummary(logsData.runs, logsData.window),
    window: logsData.window,
    timeout: logsData.timeout,
    logsFetches: logsData.logsFetches,
    partial: logsData.partial,
    continuation: logsData.continuation,
    total_runs: logsData.runs.length,
  };
}

// ---------------------------------------------------------------------------
// Command runner for the Commands panel
// ---------------------------------------------------------------------------

function parseGhAwArgs(raw) {
  const m = raw.trim().match(/^(?:gh\s+aw\s+)(.+)$/);
  return m ? m[1].trim().split(/\s+/) : null;
}

function hasFlag(args, longFlag, shortFlag = "") {
  return args.some(arg => {
    if (arg.startsWith(`${longFlag}=`)) {
      return true;
    }
    if (shortFlag && arg.startsWith(`${shortFlag}=`)) {
      return true;
    }
    if (arg === longFlag || (shortFlag && arg === shortFlag)) {
      return true;
    }
    return false;
  });
}

function logsCommandUsesJSON(args) {
  return hasFlag(args, "--json");
}

function normalizeLogsCommandArgs(args, windowId, timeout) {
  const nextArgs = [...args];
  if (!hasFlag(nextArgs, "--start-date") && !hasFlag(nextArgs, "--end-date") && !hasFlag(nextArgs, "--after-run-id") && !hasFlag(nextArgs, "--before-run-id")) {
    nextArgs.push("--start-date", getReportWindow(windowId).startDate);
  }
  if (!hasFlag(nextArgs, "--timeout")) {
    nextArgs.push("--timeout", String(timeout));
  }
  if (!hasFlag(nextArgs, "--artifacts")) {
    nextArgs.push("--artifacts", "usage");
  }
  return nextArgs;
}

async function execCommand(rawCmd, options = {}) {
  const args = parseGhAwArgs(rawCmd);
  if (!args) {
    return { command: rawCmd, output: "Only 'gh aw <subcommand>' commands are supported.", error: true };
  }
  try {
    if (args[0] === "logs" && logsCommandUsesJSON(args)) {
      const commandArgs = normalizeLogsCommandArgs(args, options.window, options.timeout ?? DEFAULT_LOG_TIMEOUT_MINUTES);
      const logsOptions = normalizeLogsOptions({ window: options.window, timeout: options.timeout });
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

    const output = await runGhAw(args);
    return { command: rawCmd, output };
  } catch (err) {
    return { command: rawCmd, output: err.stderr || err.message, error: true };
  }
}

// ---------------------------------------------------------------------------
// Pagination utility
// ---------------------------------------------------------------------------

function paginate(items, page = 1, pageSize = 20) {
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

// ---------------------------------------------------------------------------
// Loopback HTTP server per canvas instance
// ---------------------------------------------------------------------------

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
        const [html, css] = await Promise.all([readFile(join(__dirname, "web", "index.html"), "utf8"), readFile(join(__dirname, "web", "styles.css"), "utf8")]);
        res.setHeader("Content-Type", "text/html; charset=utf-8");
        res.end(html.replace("/*__APP_CSS__*/", css));
      } else if (pathname === "/app.js") {
        res.setHeader("Content-Type", "application/javascript; charset=utf-8");
        res.end(await readFile(join(__dirname, "web", "app.js"), "utf8"));
      } else if (pathname === "/pagination.js") {
        res.setHeader("Content-Type", "application/javascript; charset=utf-8");
        res.end(await readFile(join(__dirname, "web", "pagination.js"), "utf8"));
      } else if (pathname === "/api/status") {
        sendJson(await getDefinitions());
      } else if (pathname === "/api/experiments") {
        sendJson(await getExperiments());
      } else if (pathname === "/api/runs") {
        sendJson(
          await getRuns({
            count: parseInt(reqUrl.searchParams.get("count") ?? String(DEFAULT_RUN_COUNT), 10),
            window: reqUrl.searchParams.get("window") ?? "7d",
            timeout: parseInt(reqUrl.searchParams.get("timeout") ?? String(DEFAULT_LOG_TIMEOUT_MINUTES), 10),
          })
        );
      } else if (pathname === "/api/usage") {
        sendJson(
          await getUsage({
            count: parseInt(reqUrl.searchParams.get("count") ?? String(DEFAULT_RUN_COUNT), 10),
            window: reqUrl.searchParams.get("window") ?? "7d",
            timeout: parseInt(reqUrl.searchParams.get("timeout") ?? String(DEFAULT_LOG_TIMEOUT_MINUTES), 10),
          })
        );
      } else if (pathname === "/api/run-command") {
        const cmd = reqUrl.searchParams.get("cmd") ?? "";
        sendJson(
          await execCommand(cmd, {
            window: reqUrl.searchParams.get("window") ?? "7d",
            timeout: parseInt(reqUrl.searchParams.get("timeout") ?? String(DEFAULT_LOG_TIMEOUT_MINUTES), 10),
          })
        );
      } else if (pathname === "/api/refresh") {
        cache.clear();
        sendJson({ ok: true });
      } else {
        res.writeHead(404);
        res.end("Not found");
      }
    } catch (err) {
      sendJson({ error: err.message }, 500);
    }
  });
  await new Promise(r => server.listen(0, "127.0.0.1", r));
  const { port } = server.address();
  return { server, url: `http://127.0.0.1:${port}/` };
}

// ---------------------------------------------------------------------------
// Session
// ---------------------------------------------------------------------------

const session = await joinSession({
  systemMessage: {
    mode: "append",
    content: `## Agentic Workflows Dashboard

This canvas shows live data from the current repository using the gh-aw CLI.
It never calls Go code directly — all data is fetched by running CLI subcommands.

**CLI commands used by this canvas:**
- \`gh aw status --json\` — list agentic workflow definitions (workflow, engine_id, compiled, labels, status, time_remaining)
- \`gh aw logs --json -c <N> --start-date <window> --timeout <minutes>\` — list recent workflow runs and follow continuation batches progressively
- \`gh aw experiments list --json\` — list experiment workflow branches (workflow_id, branch, experiments, total_runs, last_run)

**Dev build** (when gh-aw is not installed as a gh extension):
1. Run \`make build\` in the repository root to compile \`./gh-aw\` (or \`./gh-aw.exe\` on Windows)
2. The canvas auto-detects the dev binary and uses it before falling back to \`gh aw\`

**Canvas actions available to the agent:**
- \`listDefinitions\` — calls \`gh aw status --json\`, returns paged results
- \`listRuns\` — calls \`gh aw logs --json\` with a selected report window, timeout, and continuation handling
- \`listUsage\` — aggregates workflow AIC usage, daily burn, and monthly forecast from the same logs window
- \`listExperiments\` — calls \`gh aw experiments list --json\`, returns paged results
- \`getRun\` — looks up a single run by \`run_id\`
- \`runCommand\` — executes any \`gh aw <subcommand>\` and returns stdout
- \`refresh\` — clears the 60-second cache so the next call fetches fresh data
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
            const defs = await getDefinitions();
            return paginate(defs, Number(ctx.input?.page ?? 1), Number(ctx.input?.pageSize ?? 20));
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
            const logsData = await getRuns({
              count: Number(ctx.input?.count ?? DEFAULT_RUN_COUNT),
              window: String(ctx.input?.window ?? "7d"),
              timeout: Number(ctx.input?.timeout ?? DEFAULT_LOG_TIMEOUT_MINUTES),
            });
            return {
              ...paginate(logsData.runs, Number(ctx.input?.page ?? 1), Number(ctx.input?.pageSize ?? 20)),
              partial: logsData.partial,
              logsFetches: logsData.logsFetches,
              window: logsData.window,
            };
          },
        },
        {
          name: "listUsage",
          description: "Aggregate workflow AIC usage, daily burn, and monthly forecast from gh aw logs.",
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
            const usage = await getUsage({
              count: Number(ctx.input?.count ?? DEFAULT_RUN_COUNT),
              window: String(ctx.input?.window ?? "7d"),
              timeout: Number(ctx.input?.timeout ?? DEFAULT_LOG_TIMEOUT_MINUTES),
            });
            return {
              ...paginate(usage.items, Number(ctx.input?.page ?? 1), Number(ctx.input?.pageSize ?? 20)),
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
            const experiments = await getExperiments();
            return paginate(experiments, Number(ctx.input?.page ?? 1), Number(ctx.input?.pageSize ?? 20));
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
            const logsData = await getRuns({ count: 200, window: "1mo", timeout: DEFAULT_LOG_TIMEOUT_MINUTES });
            return { run: logsData.runs.find(r => r.run_id === Number(ctx.input?.run_id)) ?? null };
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
          handler: async ctx => execCommand(String(ctx.input?.command ?? ""), { window: "7d", timeout: DEFAULT_LOG_TIMEOUT_MINUTES }),
        },
        {
          name: "refresh",
          description: "Clear the data cache so the next listDefinitions/listRuns fetches fresh data from the CLI.",
          inputSchema: { type: "object", additionalProperties: false },
          handler: () => {
            cache.clear();
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
        return { title: "Agentic Workflows Dashboard", status: "Live · gh aw", url: entry.url };
      },
      onClose: async ctx => {
        const entry = servers.get(ctx.instanceId);
        if (entry) {
          servers.delete(ctx.instanceId);
          await new Promise(r => entry.server.close(r));
        }
      },
    }),
  ],
});

workspacePath = session.workspacePath ?? process.cwd();
