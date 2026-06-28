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
let workspacePath = process.cwd();

// ---------------------------------------------------------------------------
// CLI helpers
// ---------------------------------------------------------------------------

function execp(bin, args, cwd) {
  return new Promise((resolve, reject) => {
    execFile(bin, args, {
      cwd,
      env: { ...process.env, NO_COLOR: "1", GH_NO_UPDATE_NOTIFIER: "1" },
      maxBuffer: 10 * 1024 * 1024,
    }, (err, stdout, stderr) => {
      if (err) reject(Object.assign(err, { stderr: stderr ?? "" }));
      else resolve(stdout);
    });
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

async function getRuns(count = 50) {
  const key = `runs:${count}`;
  const hit = getCached(key);
  if (hit) return hit;
  const raw = await runGhAw(["logs", "--json", "-c", String(count)]);
  const logsData = JSON.parse(raw);
  const runs = logsData.runs ?? [];
  setCached(key, runs);
  return runs;
}

// ---------------------------------------------------------------------------
// Command runner for the Commands panel
// ---------------------------------------------------------------------------

function parseGhAwArgs(raw) {
  const m = raw.trim().match(/^(?:gh\s+aw\s+)(.+)$/);
  return m ? m[1].trim().split(/\s+/) : null;
}

async function execCommand(rawCmd) {
  const args = parseGhAwArgs(rawCmd);
  if (!args) {
    return { command: rawCmd, output: "Only 'gh aw <subcommand>' commands are supported.", error: true };
  }
  try {
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
  return {
    items: items.slice(start, start + pageSize),
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
        const [html, css] = await Promise.all([
          readFile(join(__dirname, "web", "index.html"), "utf8"),
          readFile(join(__dirname, "web", "styles.css"), "utf8"),
        ]);
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
      } else if (pathname === "/api/runs") {
        const count = parseInt(reqUrl.searchParams.get("count") ?? "50", 10);
        sendJson(await getRuns(count));
      } else if (pathname === "/api/run-command") {
        const cmd = reqUrl.searchParams.get("cmd") ?? "";
        sendJson(await execCommand(cmd));
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
- \`gh aw logs --json -c <N>\` — list recent workflow runs (run_id, workflow_name, status, conclusion, duration, token_usage, turns, error_count)

**Dev build** (when gh-aw is not installed as a gh extension):
1. Run \`make build\` in the repository root to compile \`./gh-aw\` (or \`./gh-aw.exe\` on Windows)
2. The canvas auto-detects the dev binary and uses it before falling back to \`gh aw\`

**Canvas actions available to the agent:**
- \`listDefinitions\` — calls \`gh aw status --json\`, returns paged results
- \`listRuns\` — calls \`gh aw logs --json\`, returns paged results
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
          description: "List recent workflow runs via gh aw logs --json, with paging.",
          inputSchema: {
            type: "object",
            properties: {
              page: { type: "number", minimum: 1 },
              pageSize: { type: "number", minimum: 1, maximum: 100 },
              count: { type: "number", minimum: 1, maximum: 200, description: "Max runs to fetch from the CLI." },
            },
            additionalProperties: false,
          },
          handler: async ctx => {
            const runs = await getRuns(Number(ctx.input?.count ?? 50));
            return paginate(runs, Number(ctx.input?.page ?? 1), Number(ctx.input?.pageSize ?? 20));
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
            const runs = await getRuns(200);
            return { run: runs.find(r => r.run_id === Number(ctx.input?.run_id)) ?? null };
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
          handler: async ctx => execCommand(String(ctx.input?.command ?? "")),
        },
        {
          name: "refresh",
          description: "Clear the data cache so the next listDefinitions/listRuns fetches fresh data from the CLI.",
          inputSchema: { type: "object", additionalProperties: false },
          handler: () => { cache.clear(); return { ok: true }; },
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
