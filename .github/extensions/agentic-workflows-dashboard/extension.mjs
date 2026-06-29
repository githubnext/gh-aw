import { createServer } from "node:http";
import { readFile } from "node:fs/promises";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";

import { createCanvas, joinSession } from "@github/copilot-sdk/extension";

import { createGhAwRunnerWithStatus } from "./dashboard-cli.mjs";
import { DEFAULT_LOG_TIMEOUT_MINUTES, DEFAULT_RUN_COUNT } from "./dashboard-config.mjs";
import { createDashboardDataAccess } from "./dashboard-data.mjs";

const __dirname = dirname(fileURLToPath(import.meta.url));
const servers = new Map();
let workspacePath = process.cwd();
const runGhAw = createGhAwRunnerWithStatus({ getWorkspacePath: () => workspacePath });
const dataAccess = createDashboardDataAccess({ runGhAw });

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
        sendJson(await dataAccess.getDefinitions());
      } else if (pathname === "/api/cli-status") {
        sendJson(await runGhAw.getStatus());
      } else if (pathname === "/api/experiments") {
        sendJson(await dataAccess.getExperiments());
      } else if (pathname === "/api/runs") {
        sendJson(
          await dataAccess.getRuns({
            count: parseInt(reqUrl.searchParams.get("count") ?? String(DEFAULT_RUN_COUNT), 10),
            window: reqUrl.searchParams.get("window") ?? "7d",
            timeout: parseInt(reqUrl.searchParams.get("timeout") ?? String(DEFAULT_LOG_TIMEOUT_MINUTES), 10),
          })
        );
      } else if (pathname === "/api/usage") {
        sendJson(
          await dataAccess.getUsage({
            count: parseInt(reqUrl.searchParams.get("count") ?? String(DEFAULT_RUN_COUNT), 10),
            window: reqUrl.searchParams.get("window") ?? "7d",
            timeout: parseInt(reqUrl.searchParams.get("timeout") ?? String(DEFAULT_LOG_TIMEOUT_MINUTES), 10),
          })
        );
      } else if (pathname === "/api/audit") {
        const runId = reqUrl.searchParams.get("run_id") ?? "";
        if (!runId) {
          sendJson({ error: "run_id is required" }, 400);
        } else {
          sendJson(await dataAccess.getAudit(runId));
        }
      } else if (pathname === "/api/run-command") {
        const cmd = reqUrl.searchParams.get("cmd") ?? "";
        sendJson(
          await dataAccess.execCommand(cmd, {
            window: reqUrl.searchParams.get("window") ?? "7d",
            timeout: parseInt(reqUrl.searchParams.get("timeout") ?? String(DEFAULT_LOG_TIMEOUT_MINUTES), 10),
          })
        );
      } else if (pathname === "/api/refresh") {
        dataAccess.clearCache();
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
- \`listUsage\` — aggregates workflow AIC usage from logs and fills monthly forecast via \`gh aw forecast --json\`
- \`listExperiments\` — calls \`gh aw experiments list --json\`, returns paged results
- \`getRun\` — looks up a single run by \`run_id\`
- \`auditRun\` — calls \`gh aw audit <run_id> --json\` and returns structured audit data (overview, metrics, key_findings, recommendations, jobs, tool_usage, errors, warnings, firewall_analysis)
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
            const defs = await dataAccess.getDefinitions();
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
            const logsData = await dataAccess.getRuns({
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
            const usage = await dataAccess.getUsage({
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
            const experiments = await dataAccess.getExperiments();
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
            const logsData = await dataAccess.getRuns({ count: 200, window: "1mo", timeout: DEFAULT_LOG_TIMEOUT_MINUTES });
            return { run: logsData.runs.find(r => r.run_id === Number(ctx.input?.run_id)) ?? null };
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
            const runId = String(ctx.input?.run_id ?? "").trim();
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
          handler: async ctx => dataAccess.execCommand(String(ctx.input?.command ?? ""), { window: "7d", timeout: DEFAULT_LOG_TIMEOUT_MINUTES }),
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
