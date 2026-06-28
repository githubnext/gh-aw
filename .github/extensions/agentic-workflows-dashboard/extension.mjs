import { createServer } from "node:http";
import { readFile } from "node:fs/promises";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";

import { createCanvas, joinSession } from "@github/copilot-sdk/extension";

const __dirname = dirname(fileURLToPath(import.meta.url));

const definitions = buildDefinitions(240);
const runs = buildRuns(420, definitions);

function nowIso() {
  return new Date().toISOString();
}

function isoHoursAgo(hours) {
  return new Date(Date.now() - hours * 60 * 60 * 1000).toISOString();
}

function buildDefinitions(count) {
  return Array.from({ length: count }, (_, i) => {
    const index = i + 1;
    return {
      id: `wf-${String(index).padStart(3, "0")}`,
      name: `agentic-workflow-${index}`,
      description: `Automated workflow #${index} for triage, reporting, and repository automation.`,
      inputSchema: {
        type: "object",
        properties: {
          issue: { type: "number" },
          branch: { type: "string" },
        },
        additionalProperties: false,
      },
      enabled: index % 9 !== 0,
    };
  });
}

function buildStep(runNumber, idx, status) {
  return {
    id: `step-${runNumber}-${idx}`,
    title: ["Resolve context", "Execute engine", "Publish summary", "Finalize run"][idx - 1] ?? `Step ${idx}`,
    status,
    summaryMarkdown: `### ${status === "failed" ? "Action required" : "Step complete"}\n- Run: **${runNumber}**\n- Step: **${idx}**\n- Status: \`${status}\`\n\n[View workflow docs](https://github.com/github/gh-aw/tree/main/docs)`,
  };
}

function buildRuns(count, sourceDefinitions) {
  const statuses = ["queued", "running", "completed", "failed"];
  const stepStatusMap = {
    queued: ["pending", "pending", "pending", "pending"],
    running: ["done", "running", "pending", "pending"],
    completed: ["done", "done", "done", "done"],
    failed: ["done", "failed", "pending", "pending"],
  };

  return Array.from({ length: count }, (_, i) => {
    const index = i + 1;
    const status = statuses[index % statuses.length] ?? "queued";
    const definition = sourceDefinitions[index % sourceDefinitions.length];
    const stepStatuses = stepStatusMap[status] ?? stepStatusMap.queued;

    return {
      id: `run-${String(index).padStart(5, "0")}`,
      definitionId: definition.id,
      status,
      createdAt: isoHoursAgo(index * 2),
      updatedAt: isoHoursAgo(index),
      steps: stepStatuses.map((stepStatus, stepIndex) => buildStep(index, stepIndex + 1, stepStatus)),
    };
  });
}

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

function findRun(runId) {
  return runs.find(run => run.id === runId) ?? null;
}

function dispatchWorkflow(definitionId, inputs = {}) {
  const definition = definitions.find(item => item.id === definitionId);
  if (!definition) {
    return { ok: false, error: `Unknown workflow definition: ${definitionId}` };
  }

  const sequence = runs.length + 1;
  const now = nowIso();
  const run = {
    id: `run-${String(sequence).padStart(5, "0")}`,
    definitionId,
    status: "queued",
    createdAt: now,
    updatedAt: now,
    steps: [buildStep(sequence, 1, "pending"), buildStep(sequence, 2, "pending"), buildStep(sequence, 3, "pending"), buildStep(sequence, 4, "pending")],
    inputs,
  };

  runs.unshift(run);

  return {
    ok: true,
    run,
  };
}

function parseRunFlag(argsText) {
  const directMatch = argsText.match(/--run(?:=|\s+)(run-\d{5})/);
  return directMatch?.[1] ?? null;
}

function runGhAwLogs(args = "") {
  const argsText = typeof args === "string" ? args : JSON.stringify(args);
  const runId = parseRunFlag(argsText);

  if (runId) {
    const run = findRun(runId);
    if (!run) {
      return { command: `gh aw logs ${argsText}`.trim(), output: `Run ${runId} not found.` };
    }
    return {
      command: `gh aw logs --run ${runId}`,
      output: `Logs for ${run.id}\n- definition: ${run.definitionId}\n- status: ${run.status}\n- updatedAt: ${run.updatedAt}`,
    };
  }

  const latest = runs[0];
  return {
    command: "gh aw logs",
    output: `Showing logs for ${latest?.id ?? "n/a"}.\n- status: ${latest?.status ?? "n/a"}\n- updatedAt: ${latest?.updatedAt ?? "n/a"}`,
  };
}

function runGhAwAudit(args = "") {
  const argsText = typeof args === "string" ? args : JSON.stringify(args);
  const runId = parseRunFlag(argsText);

  if (runId) {
    const run = findRun(runId);
    if (!run) {
      return { command: `gh aw audit ${argsText}`.trim(), output: `Run ${runId} not found.` };
    }
    const failedSteps = run.steps.filter(step => step.status === "failed").length;
    return {
      command: `gh aw audit --run ${runId}`,
      output: `Audit for ${run.id}\n- definition: ${run.definitionId}\n- status: ${run.status}\n- failed steps: ${failedSteps}`,
    };
  }

  const completed = runs.filter(run => run.status === "completed").length;
  const failed = runs.filter(run => run.status === "failed").length;

  return {
    command: "gh aw audit",
    output: `Audit summary\n- total runs: ${runs.length}\n- completed: ${completed}\n- failed: ${failed}`,
  };
}

function runGhAwCompile(args = "") {
  const argsText = typeof args === "string" ? args : JSON.stringify(args);
  return {
    command: `gh aw compile ${argsText}`.trim(),
    output: `Compile summary\n- definitions loaded: ${definitions.length}\n- runs indexed: ${runs.length}\n- status: success`,
  };
}

function runGhAwAuditDiff(args = "") {
  const argsText = typeof args === "string" ? args : JSON.stringify(args);
  const referencedRuns = argsText.match(/run-\d{5}/g) ?? [];
  if (referencedRuns.length < 2) {
    return {
      command: `gh aw audit-diff ${argsText}`.trim(),
      output: "Need two valid runs for diff. Example: gh aw audit-diff run-00002 run-00003",
    };
  }

  const baseRun = findRun(referencedRuns[0]);
  const compareRun = findRun(referencedRuns[1]);

  if (!baseRun || !compareRun) {
    return {
      command: `gh aw audit-diff ${argsText}`.trim(),
      output: "Need two valid runs for diff. Example: gh aw audit-diff run-00002 run-00003",
    };
  }

  return {
    command: `gh aw audit-diff ${baseRun.id} ${compareRun.id}`,
    output: `Audit diff\n- base: ${baseRun.id} (${baseRun.status})\n- compare: ${compareRun.id} (${compareRun.status})\n- step delta: ${compareRun.steps.length - baseRun.steps.length}`,
  };
}

const servers = new Map();

async function startServer() {
  const server = createServer(async (req, res) => {
    const pathname = new URL(req.url ?? "/", "http://localhost").pathname;
    try {
      if (pathname === "/" || pathname === "/index.html") {
        const [html, css] = await Promise.all([
          readFile(join(__dirname, "web", "index.html"), "utf8"),
          readFile(join(__dirname, "web", "styles.css"), "utf8"),
        ]);
        res.setHeader("Content-Type", "text/html; charset=utf-8");
        res.end(html.replace("/*__APP_CSS__*/", css));
      } else if (pathname === "/app.js") {
        const content = await readFile(join(__dirname, "web", "app.js"), "utf8");
        res.setHeader("Content-Type", "application/javascript; charset=utf-8");
        res.end(content);
      } else if (pathname === "/pagination.js") {
        const content = await readFile(join(__dirname, "web", "pagination.js"), "utf8");
        res.setHeader("Content-Type", "application/javascript; charset=utf-8");
        res.end(content);
      } else {
        res.writeHead(404);
        res.end("Not found");
      }
    } catch {
      res.writeHead(500);
      res.end("Internal server error");
    }
  });
  await new Promise(r => server.listen(0, "127.0.0.1", r));
  const { port } = server.address();
  return { server, url: `http://127.0.0.1:${port}/` };
}

await joinSession({
  canvases: [
    createCanvas({
      id: "agentic-workflows-dashboard",
      displayName: "Agentic Workflows Dashboard",
      description: "A minimal dashboard for browsing workflow definitions and workflow runs.",
      actions: [
        {
          name: "listDefinitions",
          description: "List workflow definitions with paging support.",
          inputSchema: {
            type: "object",
            properties: {
              page: { type: "number", minimum: 1 },
              pageSize: { type: "number", minimum: 1, maximum: 100 },
            },
            additionalProperties: false,
          },
          handler: ctx => {
            const page = Number(ctx.input?.page ?? 1);
            const pageSize = Number(ctx.input?.pageSize ?? 20);
            return paginate(definitions, page, pageSize);
          },
        },
        {
          name: "listRuns",
          description: "List workflow runs with paging support.",
          inputSchema: {
            type: "object",
            properties: {
              page: { type: "number", minimum: 1 },
              pageSize: { type: "number", minimum: 1, maximum: 100 },
            },
            additionalProperties: false,
          },
          handler: ctx => {
            const page = Number(ctx.input?.page ?? 1);
            const pageSize = Number(ctx.input?.pageSize ?? 20);
            return paginate(runs, page, pageSize);
          },
        },
        {
          name: "getRun",
          description: "Get a workflow run by run id.",
          inputSchema: {
            type: "object",
            required: ["id"],
            properties: {
              id: { type: "string" },
            },
            additionalProperties: false,
          },
          handler: ctx => {
            const id = String(ctx.input?.id ?? "");
            return { run: findRun(id) };
          },
        },
        {
          name: "dispatchWorkflow",
          description: "Dispatch an existing predefined workflow.",
          inputSchema: {
            type: "object",
            required: ["definitionId"],
            properties: {
              definitionId: { type: "string" },
              inputs: { type: "object", additionalProperties: true },
            },
            additionalProperties: false,
          },
          handler: ctx => {
            const definitionId = String(ctx.input?.definitionId ?? "");
            const inputs = ctx.input?.inputs && typeof ctx.input.inputs === "object" ? ctx.input.inputs : {};
            return dispatchWorkflow(definitionId, inputs);
          },
        },
        {
          name: "runGhAwLogs",
          description: "Run gh aw logs command behavior.",
          inputSchema: {
            type: "object",
            properties: {
              args: { type: "string" },
            },
            additionalProperties: false,
          },
          handler: ctx => runGhAwLogs(String(ctx.input?.args ?? "")),
        },
        {
          name: "runGhAwAudit",
          description: "Run gh aw audit command behavior.",
          inputSchema: {
            type: "object",
            properties: {
              args: { type: "string" },
            },
            additionalProperties: false,
          },
          handler: ctx => runGhAwAudit(String(ctx.input?.args ?? "")),
        },
        {
          name: "runGhAwCompile",
          description: "Run gh aw compile command behavior.",
          inputSchema: {
            type: "object",
            properties: {
              args: { type: "string" },
            },
            additionalProperties: false,
          },
          handler: ctx => runGhAwCompile(String(ctx.input?.args ?? "")),
        },
        {
          name: "runGhAwAuditDiff",
          description: "Run gh aw audit-diff command behavior.",
          inputSchema: {
            type: "object",
            properties: {
              args: { type: "string" },
            },
            additionalProperties: false,
          },
          handler: ctx => runGhAwAuditDiff(String(ctx.input?.args ?? "")),
        },
      ],
      open: async (ctx) => {
        let entry = servers.get(ctx.instanceId);
        if (!entry) {
          entry = await startServer();
          servers.set(ctx.instanceId, entry);
        }
        return {
          title: "Agentic Workflows Dashboard",
          status: `${definitions.length} workflows · ${runs.length} runs`,
          url: entry.url,
        };
      },
      onClose: async (ctx) => {
        const entry = servers.get(ctx.instanceId);
        if (entry) {
          servers.delete(ctx.instanceId);
          await new Promise(r => entry.server.close(r));
        }
      },
    }),
  ],
});
