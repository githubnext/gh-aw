# AW Harness Design Plan

## Problem Statement

Design a **replacement for `copilot_harness.cjs`** — the Node.js wrapper that gh-aw uses to launch coding agent CLIs inside the GitHub Actions container. Today's harness is a thin retry loop around a single CLI invocation. The new harness — called **"aw"** — adds **multi-step orchestration**, **multi-agent**, **multi-model**, **context engineering**, **cost tracking**, and **observability**.

The harness is **built on top of the Pi agent ecosystem** (`@mariozechner/pi-coding-agent`, `pi-agent-core`, `pi-ai`). All gh-aw-specific capabilities — safe outputs, MCP gateway bridging, cost tracking, steering, observability — are implemented as **Pi extensions** using Pi's native extensibility mechanism (`ExtensionAPI`). This means the extensions are reusable with standalone Pi and the OpenClaw ecosystem.

### What this IS

- A new `engine: aw` option (registered via `harness: aw_harness.cjs` in frontmatter)
- A Pi SDK application with gh-aw-specific Pi extensions
- Optimized for execution inside the gh-aw Action container (firewall, api-proxy, MCP gateway)
- TypeScript compiled for Node.js 24, bundled as a single `.cjs` in `actions/setup/js/`

### What this is NOT

- NOT a reimplementation of gh-aw (compilation, triggers, safe-outputs post-processing, threat detection stay as-is)
- NOT a CLI spawner — no `copilot` / `claude` / `codex` CLI processes are spawned
- NOT interactive (no TUI — pure headless CI execution via Pi SDK mode)
- NOT a replacement for existing engines — `engine: copilot`, `engine: claude`, `engine: codex` continue to work via their current harnesses

## Model Resolution: api-proxy Integration

The harness does **not** do provider inference or model routing. That is handled by the **api-proxy** (`gh-aw-firewall/containers/api-proxy/model-resolver.js`), which already runs in the container.

### How model resolution works

The api-proxy accepts model names (aliases or explicit `provider/model` refs) and resolves them against available models per provider:

```
Harness (Pi SDK) → api-proxy → LLM provider
  model: "sonnet"           resolves to copilot/claude-sonnet-4.6
  model: "gpt-5-codex"      resolves to copilot/gpt-5.3-codex
  model: "copilot/gpt-4.1"  passes through as-is
```

**Alias config** (via `AWF_MODEL_ALIASES` env var):

```json
{
  "models": {
    "sonnet": ["copilot/*sonnet*", "anthropic/*sonnet*"],
    "gpt-5-codex": ["copilot/gpt-5*-codex", "openai/gpt-5*-codex"],
    "": ["sonnet", "gpt-5*-codex"]
  }
}
```

**Key properties**:

- `providerid/modelid` syntax with `*` wildcards
- Recursive alias resolution with loop detection
- Case-insensitive matching
- Semver sorting (highest version first among candidates)
- Default policy via `""` key (used when no model is specified)

### What this means for the harness

- Pi SDK's `pi-ai` is configured with the api-proxy as an **OpenAI-compatible custom provider** (using `pi.registerProvider()` or `models.json` configuration)
- The harness just passes model names through — aliases like `"sonnet"` work out of the box
- No `provider:` field needed in frontmatter — the proxy handles routing
- Per-step/per-agent model selection is just setting a different model name string
- The harness inherits the api-proxy's full model catalog without maintaining its own

## Architecture: Pi SDK + Extensions

### Core Principle: Everything is a Pi Extension

The harness is a thin orchestration layer that creates Pi `AgentSession` instances (via `createAgentSession()` from the Pi SDK). All gh-aw-specific capabilities are implemented as **Pi extensions** using the `ExtensionAPI` interface:

```typescript
// Each gh-aw capability is a Pi extension
import type { ExtensionAPI } from "@mariozechner/pi-coding-agent";

export default function safeOutputsExtension(pi: ExtensionAPI) {
  // Register safe-output tools the LLM can call
  pi.registerTool({ name: "create_issue", ... });
  pi.registerTool({ name: "create_pull_request", ... });
  pi.registerTool({ name: "add_comment", ... });

  // Hook into agent events for observability
  pi.on("tool_call", async (event, ctx) => { ... });
  pi.on("agent_end", async (event, ctx) => { ... });
}
```

**Why Pi extensions instead of custom code?**

1. **Reusable** — Extensions work with standalone Pi CLI, OpenClaw, and any Pi SDK app
2. **Composable** — Users can add their own extensions alongside gh-aw ones
3. **Standard lifecycle** — Pi handles extension loading, event dispatch, tool registration
4. **Testable** — Extensions can be tested independently with Pi's SDK test harness
5. **Ecosystem-compatible** — Can be published as Pi packages (`pi install npm:@github/aw-extensions`)

### How It Fits in the gh-aw Stack

```
┌─────────────────────────────────────────────────────────────┐
│  GitHub Actions Job (compiled from .lock.yml by gh-aw)       │
│                                                              │
│  ┌──────────────────────────────────────────────────────┐   │
│  │  Container (firewall, api-proxy, MCP gateway)         │   │
│  │                                                       │   │
│  │  ┌─────────────────────────────────────────────────┐ │   │
│  │  │  aw_harness.cjs (entry point)                   │ │   │
│  │  │                                                  │ │   │
│  │  │  1. Reads workflow.md (frontmatter + body)       │ │   │
│  │  │  2. Parses steps from markdown headings          │ │   │
│  │  │  3. Builds execution DAG                         │ │   │
│  │  │  4. For each step: creates Pi AgentSession       │ │   │
│  │  │     with gh-aw extensions loaded                 │ │   │
│  │  │  5. session.prompt() → Pi drives the agent       │ │   │
│  │  │                                                  │ │   │
│  │  │  ┌──────────────────────────────────────────┐   │ │   │
│  │  │  │  Pi SDK (createAgentSession)             │   │ │   │
│  │  │  │  ├─ pi-agent-core (agent loop, events)   │   │ │   │
│  │  │  │  ├─ pi-ai → api-proxy → LLM providers   │   │ │   │
│  │  │  │  └─ compaction, steering, auto-retry      │   │ │   │
│  │  │  └──────────────────────────────────────────┘   │ │   │
│  │  │  ┌──────────────────────────────────────────┐   │ │   │
│  │  │  │  gh-aw Pi Extensions (loaded into each   │   │ │   │
│  │  │  │  AgentSession via ExtensionAPI):          │   │ │   │
│  │  │  │  ├─ safe-outputs (tools + artifact write) │   │ │   │
│  │  │  │  ├─ mcp-bridge (gateway tools → Pi tools) │   │ │   │
│  │  │  │  ├─ cost-tracker (budget gates + events)  │   │ │   │
│  │  │  │  ├─ steering (time/budget pressure)       │   │ │   │
│  │  │  │  ├─ repair (broken session recovery)      │   │ │   │
│  │  │  │  ├─ observability (JSONL + OTel)          │   │ │   │
│  │  │  │  └─ checkpoint (persist/resume state)     │   │ │   │
│  │  │  └──────────────────────────────────────────┘   │ │   │
│  │  │  ┌──────────────────────────────────────────┐   │ │   │
│  │  │  │  api-proxy (model alias resolution)      │   │ │   │
│  │  │  │  └─ Recursive alias → provider routing    │   │ │   │
│  │  │  └──────────────────────────────────────────┘   │ │   │
│  │  │  ┌──────────────────────────────────────────┐   │ │   │
│  │  │  │  MCP Gateway (gh-aw, already running)    │   │ │   │
│  │  │  │  └─ GitHub tools, custom MCP servers      │   │ │   │
│  │  │  └──────────────────────────────────────────┘   │ │   │
│  │  └─────────────────────────────────────────────────┘ │   │
│  └──────────────────────────────────────────────────────┘   │
│                                                              │
│  ┌──────────────────────────────────────────────────────┐   │
│  │  Post-agent jobs (safe-outputs, threat detection)     │   │
│  │  — unchanged, reads same artifact format              │   │
│  └──────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────┘
```

### Design Principles

1. **Built on Pi ecosystem** — Uses Pi SDK (`createAgentSession()`, `Agent`, `AgentTool`) as the runtime. All gh-aw capabilities are Pi extensions, not custom plumbing.
2. **Extensions-first** — Every gh-aw feature is a proper Pi extension using `ExtensionAPI`. Extensions use `pi.registerTool()` for tools, `pi.on()` for events, `pi.registerProvider()` for model routing. This makes them reusable outside gh-aw.
3. **api-proxy for model resolution** — Pi's `pi-ai` talks to the api-proxy as an OpenAI-compatible provider. Model names (aliases or explicit) pass through — the proxy resolves.
4. **Optimized for gh-aw container** — Assumes firewall, api-proxy, MCP gateway are running. No redundant auth, no direct LLM API calls, no network configuration.
5. **TypeScript → Node 24** — Source is TypeScript, compiled to ES2024, bundled via esbuild to a single `.cjs`. Leverages Node 24 features (native fetch, structuredClone, AbortSignal.any).
6. **Output in `actions/setup/js/`** — The bundled `aw_harness.cjs` lives alongside `copilot_harness.cjs` and `claude_harness.cjs`. Same deployment mechanism, same runtime contract.
7. **New opt-in engine** — `engine: aw` is a new choice. Existing engines are untouched.
8. **Markdown-native steps** — `## Heading` = step boundary. HTML comments carry metadata.
9. **Observable** — JSONL event stream + OTel spans, all via the observability extension.

## Pi Provider Configuration for api-proxy

The harness registers the api-proxy as a custom Pi provider at startup, using Pi's `registerProvider()` mechanism:

```typescript
export default async function apiProxyProvider(pi: ExtensionAPI) {
  const proxyUrl = process.env.AWF_API_PROXY_URL || "http://localhost:8080/v1";

  pi.registerProvider("aw-proxy", {
    baseUrl: proxyUrl,
    apiKey: "AW_PROXY_API_KEY",   // env var name — Pi resolves it
    api: "openai-completions",     // api-proxy speaks OpenAI protocol
    models: await fetchAvailableModels(proxyUrl),
  });
}
```

This means:

- Pi treats the api-proxy as a standard OpenAI-compatible provider
- Model aliases (`sonnet`, `gpt-5-codex`) are sent as-is to the proxy
- The proxy resolves aliases, routes to the actual provider, returns the response
- Pi's token counting, cost tracking, and streaming all work transparently

## Workflow Definition Format

Uses the **existing gh-aw frontmatter** with `engine: aw` and a new optional `harness:` block.

```markdown
---
on:
  schedule:
    - cron: "0 9 * * 1-5"

engine:
  id: aw
  model: sonnet                  # Model alias — api-proxy resolves
  harness: aw_harness.cjs

permissions:
  contents: read
  issues: read
  pull-requests: read

tools:
  github:
    toolsets: [issues, pull_requests, code_search]
  bash: [grep, find, wc, git, jq]

safe-outputs:
  create-issue:
    title-prefix: "[review] "
    labels: [automated-review]
    max-count: 3

timeout-minutes: 30

observability:
  otlp:
    endpoint: ${{ secrets.OTLP_ENDPOINT }}
    headers:
      Authorization: ${{ secrets.OTLP_TOKEN }}

# ── Harness config (optional) ───────────────────────────────
harness:
  budget:
    max-cost-usd: 5.00
    warn-at-percent: 80

  context:
    compaction: summarize
    compaction-threshold: 0.75
    transcript-mode: summary

  agents:
    reviewer:
      model: sonnet
      system: |
        You are a senior code reviewer. Focus on correctness, security,
        and maintainability.
    scanner:
      model: gpt-5-codex
      system: |
        You are a security specialist. Focus on vulnerabilities,
        injection risks, and credential exposure.
    synthesizer:
      model: copilot/gpt-4.1
      system: |
        You synthesize multiple review perspectives into a single,
        prioritized action list.

  steps:
    parallel-review:
      agents: [reviewer, scanner]
      parallel: true
    synthesize:
      agent: synthesizer
      depends: [parallel-review]

  steering:
    time-warning-minutes: 5
    time-critical-minutes: 2
    budget-warn-percent: 75
    budget-critical-percent: 90

  checkpoint: true
---

## Daily Code Review

Review all changes pushed to the default branch in the last 24 hours.

### Gather Changes
<!-- harness-step: gather -->
Use `git log --since="24 hours ago"` and `git diff` to collect all
recent changes. Summarize the scope.

### Parallel Review
<!-- harness-step: parallel-review -->
Each reviewer examines the changes independently.

### Synthesize
<!-- harness-step: synthesize -->
Read outputs from both reviewers. Produce a prioritized findings list.

### Report
Create a GitHub issue with the synthesized review.
```

### How Steps Work

**Step extraction**: Each `## Heading` or `### Heading` is a potential step. Linked to `harness.steps` via `<!-- harness-step: name -->` HTML comments.

**Implicit behavior**:

- No `<!-- harness-step -->` → sequential by default (document order)
- With annotation → follows `harness.steps` config (parallel, depends, agent)
- No `harness:` block → entire body = single Pi session prompt

**Step execution**:

```
For each step (respecting DAG order):
  1. Build prompt = step markdown + upstream transcripts + system prompt
  2. Create Pi AgentSession with gh-aw extensions:
     - api-proxy provider registered
     - MCP bridge tools registered
     - Safe-output tools registered
     - Steering, repair, cost, observability extensions active
  3. session.prompt() → Pi agent loop runs
  4. Extensions handle events (cost tracking, steering, observability)
  5. Capture transcript for downstream steps
  6. Budget gate check, checkpoint state
```

## gh-aw Pi Extensions

All gh-aw-specific behavior is packaged as Pi extensions. Each extension is a standalone TypeScript module that exports a default function receiving `ExtensionAPI`.

### Extension 1: api-proxy Provider

Registers the gh-aw api-proxy as a custom Pi provider.

```typescript
export default async function(pi: ExtensionAPI) {
  const proxyUrl = process.env.AWF_API_PROXY_URL || "http://localhost:8080/v1";
  const models = await fetchModelsFromProxy(proxyUrl);

  pi.registerProvider("aw-proxy", {
    baseUrl: proxyUrl,
    apiKey: "AWF_API_PROXY_TOKEN",
    api: "openai-completions",
    models,
  });
}
```

**Why an extension**: Pi's `registerProvider()` is designed exactly for this. Async factory support means we can fetch the model list at startup before any session begins.

### Extension 2: MCP Gateway Bridge

Bridges gh-aw's MCP gateway tools into Pi's tool system as `AgentTool` instances.

```typescript
export default function(pi: ExtensionAPI) {
  const gatewayConfig = loadMCPGatewayConfig();

  for (const [serverName, tools] of Object.entries(gatewayConfig)) {
    for (const tool of tools) {
      pi.registerTool({
        name: `${serverName}_${tool.name}`,
        description: tool.description,
        parameters: tool.inputSchema,
        async execute(toolCallId, params, signal) {
          return await callMCPGateway(serverName, tool.name, params);
        },
      });
    }
  }
}
```

**Why an extension**: Pi's `pi.registerTool()` is the standard way to add tools. MCP gateway tools become first-class Pi tools — visible in sessions, tracked in events, subject to `beforeToolCall`/`afterToolCall` hooks.

### Extension 3: Safe Outputs

Registers safe-output tools (create-issue, create-pull-request, add-comment, etc.) and writes artifact files in gh-aw's expected format.

```typescript
export default function(pi: ExtensionAPI) {
  const config = loadSafeOutputsConfig();

  pi.registerTool({
    name: "create_issue",
    description: "Create a GitHub issue (safe output)",
    parameters: Type.Object({
      title: Type.String(),
      body: Type.String(),
      labels: Type.Optional(Type.Array(Type.String())),
    }),
    async execute(toolCallId, params, signal) {
      const artifact = buildSafeOutputArtifact("create-issue", params, config);
      await writeSafeOutputArtifact(artifact);
      return { content: [{ type: "text", text: `Issue queued: ${params.title}` }] };
    },
  });

  // ... register other safe-output tools

  pi.on("agent_end", async (event, ctx) => {
    await finalizeSafeOutputManifest();
  });
}
```

**Why an extension**: Tool registration + lifecycle hooks. The `agent_end` event is the right place to finalize artifacts.

### Extension 4: Cost Tracker

Monitors token usage and cost via Pi's event stream. Enforces budget gates.

```typescript
export default function(pi: ExtensionAPI) {
  const budget = loadBudgetConfig();
  let totalCost = 0;

  pi.on("turn_end", async (event, ctx) => {
    totalCost += extractCostFromTurn(event);

    const percent = (totalCost / budget.maxCostUsd) * 100;
    if (percent >= budget.budgetCriticalPercent) {
      ctx.agent.abort();
    } else if (percent >= budget.budgetWarnPercent) {
      ctx.agent.steer({
        role: "user",
        content: `⚠️ Budget: ${percent.toFixed(0)}% used. Be concise.`,
        timestamp: Date.now(),
      });
    }
  });

  pi.on("agent_end", async () => {
    emitCostSummary(totalCost);
  });
}
```

**Why an extension**: Pi's event stream (`turn_end`) provides token/cost data per turn. The `steer()` API injects budget warnings naturally.

### Extension 5: Steering (Resource Pressure)

Monitors time remaining and budget, injects steering messages via Pi's native `session.steer()`.

```typescript
export default function(pi: ExtensionAPI) {
  const config = loadSteeringConfig();
  let startTime: number;

  pi.on("agent_start", async () => {
    startTime = Date.now();
  });

  pi.on("turn_end", async (event, ctx) => {
    const elapsed = (Date.now() - startTime) / 60000;
    const remaining = config.timeoutMinutes - elapsed;

    if (remaining <= config.timeCriticalMinutes) {
      ctx.agent.steer({
        role: "user",
        content: `⚠️ CRITICAL: ${remaining.toFixed(0)}min left. Write final output NOW.`,
        timestamp: Date.now(),
      });
    } else if (remaining <= config.timeWarningMinutes) {
      ctx.agent.steer({
        role: "user",
        content: `⚠️ ${remaining.toFixed(0)}min remaining. Wrap up.`,
        timestamp: Date.now(),
      });
    }
  });
}
```

**Why an extension**: Pi's `turn_end` event fires after each tool execution cycle. Perfect timing for resource checks. `steer()` delivers the message before the next LLM call.

### Extension 6: Session Repair

Detects broken tool calls and repairs the session via Pi's message history manipulation.

```typescript
export default function(pi: ExtensionAPI) {
  pi.on("tool_result", async (event, ctx) => {
    if (isCorruptedToolResult(event)) {
      const messages = ctx.agent.state.messages;
      const repaired = truncateBrokenMessages(messages);
      ctx.agent.state.messages = repaired;
      emitRepairEvent("truncate_and_resume", event.toolName);
    }
  });

  pi.on("agent_end", async (event, ctx) => {
    if (event.error && isRecoverableError(event.error)) {
      const summary = await summarizeTranscript(ctx.agent.state.messages);
      ctx.agent.followUp({
        role: "user",
        content: `Previous progress: ${summary}\nContinue from here.`,
        timestamp: Date.now(),
      });
    }
  });
}
```

**Why an extension**: Pi exposes `agent.state.messages` for manipulation and `tool_result` events for interception.

### Extension 7: Observability

Emits JSONL events to stderr and generates OTel spans.

```typescript
export default function(pi: ExtensionAPI) {
  pi.on("agent_start", async (event) => {
    emitJsonl({ event: "step_start", step: currentStep, model: currentModel });
    startOtelSpan(currentStep);
  });

  pi.on("turn_end", async (event) => {
    recordOtelAttributes(event);
  });

  pi.on("tool_execution_end", async (event) => {
    emitJsonl({ event: "tool_end", tool: event.toolName, duration: event.duration });
  });

  pi.on("agent_end", async (event) => {
    emitJsonl({ event: "step_end", step: currentStep, tokens: event.tokens, cost: event.cost });
    endOtelSpan(currentStep);
  });
}
```

**Why an extension**: Pi's full event stream provides exactly the data needed for JSONL and OTel.

### Extension 8: Checkpoint

Persists run state for long workflows.

```typescript
export default function(pi: ExtensionAPI) {
  pi.on("agent_end", async (event, ctx) => {
    await saveCheckpoint({
      step: currentStep,
      status: event.error ? "failed" : "done",
      transcript: ctx.agent.state.messages,
      cost: totalCost,
    });
  });
}
```

## Orchestration Layer (Not an Extension)

The multi-step DAG orchestration is the **harness entry point** — it sits above the Pi SDK and creates/manages multiple `AgentSession` instances. This is NOT a Pi extension because it manages session lifecycles rather than extending a single session.

```typescript
// index.ts — entry point
import { createAgentSession, SessionManager } from "@mariozechner/pi-coding-agent";

async function main() {
  const workflow = parseWorkflow(process.argv[2]);
  const dag = buildDAG(workflow);
  const extensions = [
    apiProxyProvider,
    mcpBridgeExtension,
    safeOutputsExtension,
    costTrackerExtension,
    steeringExtension,
    repairExtension,
    observabilityExtension,
    checkpointExtension,
  ];

  for (const stepGroup of dag.executionOrder()) {
    await Promise.all(stepGroup.map(async (step) => {
      const { session } = await createAgentSession({
        sessionManager: SessionManager.inMemory(),
        extensions,
        model: resolveModel(step.agent?.model || workflow.defaultModel),
        systemPrompt: buildSystemPrompt(step),
      });

      const prompt = buildStepPrompt(step, transcripts);
      await session.prompt(prompt);

      transcripts[step.name] = captureTranscript(session);
      session.dispose();
    }));
  }
}
```

## Backwards Compatibility

| Scenario | Behavior |
|----------|----------|
| `engine: copilot` (existing) | Uses current `copilot_harness.cjs` — unchanged |
| `engine: claude` (existing) | Uses current Claude Code flow — unchanged |
| `engine: codex` (existing) | Uses current Codex flow — unchanged |
| `engine: aw` without `harness:` block | Single-step: entire body = one Pi session prompt |
| `engine: aw` with `harness:` block | Multi-step orchestration mode |
| `engine: aw` with `harness.steps` | Explicit DAG (parallel, depends, agent assignment) |
| `engine: aw` without `harness.agents` | All steps use `engine.model` |

## Build & Deployment

### TypeScript Configuration

```jsonc
// tsconfig.json
{
  "compilerOptions": {
    "target": "es2024",           // Node 24 supports ES2024
    "module": "es2022",
    "lib": ["es2024"],
    "moduleResolution": "bundler",
    "strict": true,
    "skipLibCheck": true,
    "outDir": "dist",
    "declaration": false
  }
}
```

### Bundle Configuration

```typescript
// build.ts — esbuild config
import { build } from "esbuild";

await build({
  entryPoints: ["src/index.ts"],
  bundle: true,
  platform: "node",
  target: "node24",
  format: "cjs",                  // .cjs required by gh-aw harness validation
  outfile: "dist/aw_harness.cjs",
  external: [],                   // Bundle everything (no runtime npm install)
  minify: false,                  // Keep readable for debugging in Actions logs
  sourcemap: "inline",            // Debugging in CI
});
```

### Output Location

The bundled `aw_harness.cjs` is copied to `actions/setup/js/aw_harness.cjs` — alongside existing harnesses:

```
actions/setup/js/
├── copilot_harness.cjs       # Existing
├── claude_harness.cjs        # Existing
├── aw_harness.cjs            # NEW — bundled from aw-harness/
├── *.cjs                     # Other existing action scripts
└── *.test.cjs                # Tests
```

### Testing

Tests use the same Vitest setup as the existing `actions/setup/js/` scripts:

- Unit tests for parser, planner, each extension
- Integration tests with mock Pi sessions (in-memory session manager)
- Tests co-located: `aw_harness.test.cjs` or in a `test/` subdirectory

## Project Structure

```
aw-harness/
├── package.json                  # deps: pi-coding-agent, pi-agent-core, pi-ai
├── tsconfig.json                 # target: es2024, module: es2022
├── build.ts                      # esbuild → dist/aw_harness.cjs
├── src/
│   ├── index.ts                  # Entry point: parseWorkflow → buildDAG → run sessions
│   ├── parser.ts                 # Workflow markdown → steps + config
│   ├── planner.ts                # DAG construction, topological sort
│   ├── dag-runner.ts             # Orchestrate sessions (sequential + parallel)
│   ├── transcript.ts             # Inter-step data flow (save/load/summarize)
│   ├── context.ts                # Prompt assembly, compaction
│   └── extensions/               # gh-aw Pi extensions
│       ├── api-proxy-provider.ts # Register api-proxy as Pi provider
│       ├── mcp-bridge.ts         # MCP gateway tools → Pi AgentTool
│       ├── safe-outputs.ts       # Safe-output tools + artifact writing
│       ├── cost-tracker.ts       # Budget gates via turn_end events
│       ├── steering.ts           # Time/budget pressure via session.steer()
│       ├── repair.ts             # Broken session recovery
│       ├── observability.ts      # JSONL events + OTel spans
│       └── checkpoint.ts         # Persist/resume run state
├── test/
│   ├── parser.test.ts
│   ├── planner.test.ts
│   ├── extensions/
│   │   ├── cost-tracker.test.ts
│   │   ├── steering.test.ts
│   │   ├── repair.test.ts
│   │   └── ...
│   └── integration/
│       └── dag-runner.test.ts
└── dist/
    └── aw_harness.cjs            # → copied to actions/setup/js/
```

## Todos

1. **Scaffold project** — Initialize TypeScript project in `aw-harness/`. Configure package.json with Pi SDK deps (`@mariozechner/pi-coding-agent`, `pi-agent-core`, `pi-ai`). Set up tsconfig for ES2024/Node 24. Configure esbuild bundle → `dist/aw_harness.cjs`.

2. **Implement api-proxy provider extension** — Pi extension that registers the api-proxy as a custom provider via `pi.registerProvider()`. Async factory fetches available models at startup. All model requests route through the proxy.

3. **Implement parser** — Read workflow markdown, extract frontmatter, parse `harness:` block, extract steps from heading boundaries and `<!-- harness-step -->` annotations. Fall back to single-step mode.

4. **Implement DAG planner** — Topological sort, parallel group detection, sequential fallback. Validate no cycles, all agent/step references resolve.

5. **Implement MCP bridge extension** — Pi extension that reads MCP gateway config and registers each MCP tool as a Pi `AgentTool` via `pi.registerTool()`.

6. **Implement safe-outputs extension** — Pi extension that registers safe-output tools (create-issue, create-pull-request, add-comment, etc.). Uses `pi.on("agent_end")` to finalize artifact manifest.

7. **Implement DAG runner** — Orchestrates multiple `createAgentSession()` calls. Sequential steps + `Promise.all()` for parallel groups. Passes gh-aw extensions to each session. Manages transcript flow between steps.

8. **Implement transcript manager** — Save step output to disk. Load for downstream steps. Support `summary` mode (use a Pi session to summarize) and `full` mode.

9. **Implement context engine** — Prompt assembly with priority ordering. Compaction via `none`, `sliding-window`, or `summarize`.

10. **Implement cost tracker extension** — Pi extension that monitors `turn_end` events for token/cost data. Enforces soft (steer warning) and hard (abort) budget gates.

11. **Implement steering extension** — Pi extension that monitors time/budget and injects user messages via `session.steer()` on `turn_end`.

12. **Implement repair extension** — Pi extension that detects broken tool calls via `tool_result` events. Repairs via message truncation or summarize-and-restart.

13. **Implement checkpoint extension** — Pi extension that persists step completion state on `agent_end`. Resume from checkpoint on `--continue`.

14. **Implement observability extension** — Pi extension that emits JSONL to stderr on agent/tool events. Generates OTel spans using `observability.otlp` config.

15. **Write tests** — Unit tests for parser, planner, each extension (mock `ExtensionAPI`). Integration tests with `createAgentSession()` + `SessionManager.inMemory()`.

16. **Write example workflows** — 3 examples: single-step, multi-step sequential, multi-agent parallel with different models.

17. **Add build to Makefile** — Add `make aw-harness` target that runs esbuild and copies `aw_harness.cjs` to `actions/setup/js/`.
