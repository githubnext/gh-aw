# AW Harness Specification

---

**Title:** AW Harness — Multi-Step Agentic Workflow Execution Engine

**Status:** Unofficial Draft

**Date:** 2025-07-14

**Editor:** GitHub gh-aw Team

---

## Abstract

This document specifies the **AW Harness** (`aw_harness.cjs`), a Node.js execution engine for the `engine: aw` mode of GitHub Agentic Workflows (gh-aw). The harness provides multi-step orchestration, multi-agent coordination, context engineering, cost tracking, and observability, built on top of the Pi agent SDK ecosystem. All gh-aw-specific capabilities are implemented as Pi extensions using Pi's native `ExtensionAPI` extensibility mechanism.

## Status of This Document

This is an internal design specification for the GitHub gh-aw project. It is not a W3C standard, nor is it on the W3C standards track. The document describes the intended architecture, contracts, and implementation plan for `aw_harness.cjs`. Feedback and corrections **SHOULD** be submitted via the project's standard pull request process.

---

## Table of Contents

1. [Introduction](#1-introduction)
2. [Conformance](#2-conformance)
3. [Terminology and Definitions](#3-terminology-and-definitions)
4. [Architecture](#4-architecture)
5. [Harness Invocation Contract](#5-harness-invocation-contract)
6. [Workflow Definition](#6-workflow-definition)
7. [DAG Execution Model](#7-dag-execution-model)
8. [Extensions](#8-extensions)
9. [Model Resolution](#9-model-resolution)
10. [Build and Deployment](#10-build-and-deployment)
11. [Security Considerations](#11-security-considerations)
12. [Privacy Considerations](#12-privacy-considerations)
13. [References](#13-references)

---

## 1. Introduction

*(This section is non-normative.)*

The existing gh-aw harnesses (`copilot_harness.cjs`, `claude_harness.cjs`) are thin retry loops around a single CLI invocation. As workflow complexity grows, authors need multi-step orchestration, parallel agent execution, per-step model selection, budget management, and structured observability — none of which the current harnesses provide.

The AW Harness introduces `engine: aw` as a new opt-in execution engine. It does not replace existing engines; `engine: copilot`, `engine: claude`, and `engine: codex` continue to operate unchanged via their current harnesses. The AW Harness is a Pi SDK application: it creates one `AgentSession` per workflow step, loads a fixed set of gh-aw Pi extensions into each session, and orchestrates sessions according to a DAG derived from the workflow's `harness:` frontmatter block and Markdown heading structure.

The harness is designed exclusively for the gh-aw Actions container environment. It assumes the firewall, api-proxy, and MCP gateway are already running. It performs no direct LLM API calls and requires no additional authentication setup beyond what the container already provides.

### 1.1 Scope

This specification covers:

- The entry-point invocation contract for `aw_harness.cjs`.
- The frontmatter schema for `engine: aw` workflows.
- The step-extraction and DAG-construction algorithms.
- The normative requirements for each of the eight gh-aw Pi extensions.
- The model resolution contract via the api-proxy.
- The build and deployment configuration.

This specification does not cover:

- The compilation of workflow Markdown to GitHub Actions YAML (handled by `gh-aw` proper).
- Safe-outputs post-processing and threat detection (handled by post-agent jobs, unchanged).
- The Pi SDK internals (`pi-agent-core`, `pi-ai`).
- The api-proxy internals (`model-resolver.js`).

### 1.2 Background and Motivation

The Pi agent ecosystem (`@mariozechner/pi-coding-agent`, `pi-agent-core`, `pi-ai`) provides a composable, extension-based SDK for building agentic applications. By implementing all gh-aw-specific capabilities as Pi extensions, those extensions become:

- **Reusable** — They work with standalone Pi CLI and any Pi SDK application.
- **Composable** — Users can add their own extensions alongside the provided set.
- **Ecosystem-compatible** — They can be published as Pi packages.

---

## 2. Conformance

The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this document are to be interpreted as described in [RFC 2119].

| Keyword | Meaning |
|---------|---------|
| **MUST** / **REQUIRED** / **SHALL** | Absolute requirement |
| **MUST NOT** / **SHALL NOT** | Absolute prohibition |
| **SHOULD** / **RECOMMENDED** | Strong recommendation; deviation requires documented justification |
| **SHOULD NOT** | Strong recommendation against; deviation requires documented justification |
| **MAY** / **OPTIONAL** | Permitted but not required |

A **conforming implementation** is one that satisfies all **MUST** and **MUST NOT** requirements in this specification.

---

## 3. Terminology and Definitions

**AW Harness**
: The execution engine implemented in `aw_harness.cjs`, invoked when a workflow declares `engine: aw`. It is responsible for parsing the workflow, constructing the DAG, and orchestrating one `AgentSession` per step.

**AgentSession**
: A Pi SDK session object, obtained via `createAgentSession()`, that manages a single agent's message loop, tool calls, and event stream.

**api-proxy**
: A sidecar process (`gh-aw-firewall/containers/api-proxy/model-resolver.js`) running in the gh-aw container. It accepts OpenAI-compatible API requests, resolves model aliases, and routes calls to the appropriate LLM provider. The harness communicates with it at `http://localhost:8080/v1` by default.

**cli-proxy**
: A feature that mounts MCP servers as CLI tools on `PATH`, making them callable as ordinary shell commands within agent sessions.

**DAG (Directed Acyclic Graph)**
: The execution graph derived from the workflow's `harness.steps` declarations and implicit document-order dependencies. Nodes are steps; directed edges encode `depends` relationships. Cycles are prohibited.

**ExtensionAPI**
: The Pi SDK interface (`ExtensionAPI` from `@mariozechner/pi-coding-agent`) that a Pi extension receives as its sole argument. Provides `pi.registerTool()`, `pi.registerProvider()`, and `pi.on()`.

**gh-proxy**
: A feature that provides a pre-authenticated `gh` CLI binary in the agent's bash environment, enabling direct GitHub API access without separate token management.

**harness step**
: A unit of work within an `engine: aw` workflow. Each step is assigned one `AgentSession`, a prompt derived from the corresponding Markdown section, and optionally a named agent definition.

**MCP Gateway**
: The gh-aw MCP gateway process that exposes GitHub tools and custom MCP server tools as CLI commands (via `cli-proxy`) in the agent's bash environment. It runs independently of the harness in the same container.

**model alias**
: A short name (e.g., `"sonnet"`, `"gpt-5-codex"`) resolved by the api-proxy to a fully-qualified `provider/model` string. The harness passes aliases through without resolution.

**Pi extension**
: A TypeScript module that exports a default function with signature `(pi: ExtensionAPI) => void | Promise<void>`. Loaded into an `AgentSession` to register tools, subscribe to events, or register providers.

**safe output**
: A deferred GitHub action (create issue, create pull request, add comment, etc.) expressed as an artifact file written during agent execution and processed by the post-agent job.

**step annotation**
: An HTML comment of the form `<!-- harness-step: <name> -->` embedded in a Markdown section to associate that section with a named entry in `harness.steps`.

**transcript**
: The complete message history of a completed `AgentSession`, optionally summarized, passed as context to downstream steps.

**workflow document**
: A Markdown file with YAML frontmatter that declares an `engine: aw` workflow. The frontmatter is parsed by the gh-aw compiler at compile time; the harness itself never reads the raw Markdown file. Instead, the compiler provides the harness with pre-processed inputs: `config.json` (harness configuration), `prompt.txt` (extracted prompt body), and any referenced agent files.

---

## 4. Architecture

### 4.1 Stack Overview

The AW Harness is the topmost layer within the gh-aw container. The following ASCII diagram illustrates the component relationships.

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
│  │  │  1. Reads config.json + prompt.txt (pre-parsed by compiler) │ │   │
│  │  │  2. Parses steps from step config + prompt text            │ │   │
│  │  │  3. Builds execution DAG                                   │ │   │
│  │  │  4. For each step: creates Pi AgentSession                 │ │   │
│  │  │     with gh-aw extensions loaded                           │ │   │
│  │  │  5. session.prompt() → Pi drives the agent                 │ │   │
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

### 4.2 Design Principles

1. **Built on Pi ecosystem.** A conforming implementation **MUST** use the Pi SDK (`createAgentSession()`, `Agent`, `AgentTool`) as the agent runtime. All gh-aw capabilities **MUST** be Pi extensions, not custom plumbing.

2. **Extensions-first.** Every gh-aw feature **MUST** be implemented as a proper Pi extension using `ExtensionAPI`. Extensions **MUST** use `pi.registerTool()` for tools, `pi.on()` for events, and `pi.registerProvider()` for model routing, making them reusable outside gh-aw.

3. **api-proxy for model resolution.** Pi's `pi-ai` **MUST** be configured to communicate with the api-proxy as an OpenAI-compatible provider. Model names (aliases or explicit) **MUST** be passed through to the proxy without local resolution.

4. **Optimized for gh-aw container.** The harness **MUST** assume that the firewall, api-proxy, and MCP gateway are already running. It **MUST NOT** perform direct LLM API calls, redundant authentication, or network configuration. MCP tools are available to agent sessions as bash CLI tools via `cli-proxy` — no additional bridging is required.

5. **`gh-proxy` and `cli-proxy` always on.** GitHub and other MCP server tools are available to the agent as CLI commands on `PATH` (via `cli-proxy`) and via the pre-authenticated `gh` binary (via `gh-proxy`). A conforming implementation **MUST** enable both `gh-proxy` and `cli-proxy` when `engine: aw` is selected. A conforming implementation **MUST NOT** honor attempts to disable these features for `engine: aw`, regardless of the values specified in the workflow frontmatter (see [Section 6.2](#62-overrides-and-fixed-settings)).

6. **TypeScript → Node 24.** Source **MUST** be TypeScript, compiled to ES2024, bundled via esbuild to a single `.cjs`. Leverages Node 24 features (native fetch, `structuredClone`, `AbortSignal.any`).

7. **Output in `actions/setup/js/`.** The bundled `aw_harness.cjs` **MUST** be placed in `actions/setup/js/aw_harness.cjs`, alongside `copilot_harness.cjs` and `claude_harness.cjs`. The same deployment mechanism and runtime contract apply.

8. **New opt-in engine.** `engine: aw` is an independent opt-in. Existing engines **MUST** be untouched.

9. **Markdown-native steps.** `## Heading` or `### Heading` elements **MUST** be recognized as step boundaries. HTML comments carry step metadata.

10. **Observable.** All implementations **MUST** emit a JSONL event stream to stderr and **SHOULD** generate OTel spans when an OTLP endpoint is configured.

---

## 5. Harness Invocation Contract

### 5.1 Entry Point

The AW Harness **MUST** be invocable as a Node.js CommonJS module from the command line. The gh-aw compiler pre-processes the workflow markdown (parsing frontmatter, extracting the prompt body, resolving imports) and provides the harness with pre-built input files. A conforming invocation has the form:

```
node aw_harness.cjs --config <config-path> --prompt <prompt-path> [--agent-file <agent-path>]
```

where:
- `<config-path>` is the path to the compiler-generated `config.json` file containing the parsed harness configuration.
- `<prompt-path>` is the path to the compiler-generated `prompt.txt` file containing the extracted prompt body.
- `<agent-path>` is the optional path to a compiled agent file (`.md` with frontmatter stripped), provided when the workflow references an agent via `imports:`.

A conforming implementation **MUST NOT** read or parse workflow Markdown files directly; all configuration and prompt content **MUST** be consumed from the pre-processed input files provided by the compiler.

### 5.2 Environment Variables

A conforming implementation **MUST** read the following environment variables:

| Variable | Description | Default |
|----------|-------------|---------|
| `AWF_API_PROXY_URL` | Base URL of the api-proxy OpenAI-compatible endpoint | `http://localhost:8080/v1` |
| `AWF_API_PROXY_TOKEN` | Bearer token for api-proxy authentication | *(required; no default)* |
| `AWF_MODEL_ALIASES` | JSON string containing model alias configuration | *(empty; aliases resolved by proxy)* |

A conforming implementation **SHOULD** read additional standard GitHub Actions environment variables (`GITHUB_REPOSITORY`, `GITHUB_RUN_ID`, etc.) for use in observability spans and checkpoint keys.

### 5.3 Exit Codes

| Code | Meaning |
|------|---------|
| `0` | All steps completed successfully |
| `1` | One or more steps failed (non-recoverable error) |
| `2` | Invocation error (missing config path, unreadable config file) |

A conforming implementation **MUST** exit with code `0` if and only if all DAG steps complete without error. It **MUST** exit with a non-zero code on any unrecovered failure.

### 5.4 Standard Streams

- **stdout**: Reserved for structured output (e.g., JSON summaries). A conforming implementation **SHOULD NOT** write diagnostic messages to stdout.
- **stderr**: All diagnostic messages, JSONL event stream, and debug output **MUST** be written to stderr.

---

## 6. Workflow Definition

### 6.1 Frontmatter Schema

An `engine: aw` workflow document **MUST** include a YAML frontmatter block conforming to the existing gh-aw frontmatter schema, extended with the optional `harness:` key described below. The gh-aw compiler parses this frontmatter at compile time and emits a `config.json` file consumed by the harness at runtime; the harness itself **MUST NOT** re-parse the raw Markdown frontmatter.

> [!NOTE] Non-normative example.
>
> The following is a complete example of an `engine: aw` workflow document illustrating all supported frontmatter keys.
>
> ```markdown
> ---
> on:
>   schedule:
>     - cron: "0 9 * * 1-5"
>
> engine:
>   id: aw
>   model: sonnet                  # Model alias — api-proxy resolves
>   harness: aw_harness.cjs
>
> permissions:
>   contents: read
>   issues: read
>   pull-requests: read
>
> # gh-proxy and cli-proxy are ALWAYS enabled for engine: aw.
> # MCP tools are available as CLI commands on PATH (via cli-proxy) and
> # via the pre-authenticated `gh` binary (via gh-proxy).
> cli-proxy: true
>
> tools:
>   github:
>     mode: gh-proxy               # Always gh-proxy for engine: aw
>     toolsets: [issues, pull_requests, code_search]
>   bash: [grep, find, wc, git, jq]
>
> safe-outputs:
>   create-issue:
>     title-prefix: "[review] "
>     labels: [automated-review]
>     max-count: 3
>
> timeout-minutes: 30
>
> observability:
>   otlp:
>     endpoint: ${{ secrets.OTLP_ENDPOINT }}
>     headers:
>       Authorization: ${{ secrets.OTLP_TOKEN }}
>
> # ── Harness config (optional) ───────────────────────────────
> harness:
>   budget:
>     max-cost-usd: 5.00
>     warn-at-percent: 80
>
>   context:
>     compaction: summarize
>     compaction-threshold: 0.75
>     transcript-mode: summary
>
>   agents:
>     reviewer:
>       model: sonnet
>       system: |
>         You are a senior code reviewer. Focus on correctness, security,
>         and maintainability.
>     scanner:
>       model: gpt-5-codex
>       system: |
>         You are a security specialist. Focus on vulnerabilities,
>         injection risks, and credential exposure.
>     synthesizer:
>       model: copilot/gpt-4.1
>       system: |
>         You synthesize multiple review perspectives into a single,
>         prioritized action list.
>
>   steps:
>     parallel-review:
>       agents: [reviewer, scanner]
>       parallel: true
>     synthesize:
>       agent: synthesizer
>       depends: [parallel-review]
>
>   steering:
>     time-warning-minutes: 5
>     time-critical-minutes: 2
>     budget-warn-percent: 75
>     budget-critical-percent: 90
>
>   checkpoint: true
> ---
>
> ## Daily Code Review
>
> Review all changes pushed to the default branch in the last 24 hours.
>
> ### Gather Changes
> <!-- harness-step: gather -->
> Use `git log --since="24 hours ago"` and `git diff` to collect all
> recent changes. Summarize the scope.
>
> ### Parallel Review
> <!-- harness-step: parallel-review -->
> Each reviewer examines the changes independently.
>
> ### Synthesize
> <!-- harness-step: synthesize -->
> Read outputs from both reviewers. Produce a prioritized findings list.
>
> ### Report
> Create a GitHub issue with the synthesized review.
> ```

#### 6.1.1 `harness.budget`

The `harness.budget` key is **OPTIONAL**. When present, it **MUST** contain:

- `max-cost-usd` (number): Maximum total cost in USD for the run. The cost-tracker extension **MUST** abort the current session if this limit is exceeded.
- `warn-at-percent` (number, 0–100): Percentage of `max-cost-usd` at which a steering warning **MUST** be injected.

#### 6.1.2 `harness.context`

The `harness.context` key is **OPTIONAL**. When present, it **MAY** contain:

- `compaction` (string): One of `none`, `sliding-window`, or `summarize`. Default: `none`.
- `compaction-threshold` (number, 0–1): Context fill fraction at which compaction triggers. Default: `0.75`.
- `transcript-mode` (string): One of `full` or `summary`. Controls how upstream step transcripts are included in downstream prompts. Default: `full`.

#### 6.1.3 `harness.agents`

The `harness.agents` key is **OPTIONAL**. Each entry defines a named agent with:

- `model` (string, **REQUIRED**): Model alias or fully-qualified `provider/model` string.
- `system` (string, **OPTIONAL**): System prompt override for this agent.

#### 6.1.4 `harness.steps`

The `harness.steps` key is **OPTIONAL**. Each entry defines a named step with:

- `agent` (string, **OPTIONAL**): Name of an agent defined in `harness.agents`. Mutually exclusive with `agents`.
- `agents` (array of strings, **OPTIONAL**): Names of agents to run in parallel for this step. Mutually exclusive with `agent`.
- `parallel` (boolean, **OPTIONAL**): When `true` and `agents` is provided, all listed agents execute in parallel. Default: `false`.
- `depends` (array of strings, **OPTIONAL**): Names of steps that **MUST** complete before this step begins.

#### 6.1.5 `harness.steering`

The `harness.steering` key is **OPTIONAL**. When present, it **MAY** contain:

- `time-warning-minutes` (number): Minutes before timeout at which a warning **SHOULD** be injected. Default: `5`.
- `time-critical-minutes` (number): Minutes before timeout at which a critical message **MUST** be injected. Default: `2`.
- `budget-warn-percent` (number): Budget percentage at which a warning **SHOULD** be injected. Default: `75`.
- `budget-critical-percent` (number): Budget percentage at which the session **MUST** be aborted. Default: `90`.

#### 6.1.6 `harness.checkpoint`

The `harness.checkpoint` key is **OPTIONAL**. When set to `true`, the checkpoint extension **MUST** persist step state on `agent_end`.

### 6.2 Overrides and Fixed Settings

A conforming implementation **MUST** apply the following overrides regardless of values specified in the workflow frontmatter:

| Setting | Enforced value | Reason |
|---------|----------------|--------|
| `cli-proxy` | `true` | Required: MCP tools are exposed as CLI tools on `PATH` |
| `tools.github.mode` | `gh-proxy` | Pi SDK requires `gh-proxy`; `remote` mode is not supported |

A conforming implementation **MUST NOT** honor attempts to disable `cli-proxy` or set `tools.github.mode: remote` when `engine: aw` is active. These settings **MUST** be overridden. A conforming implementation **MUST** emit a warning to stderr when either override is applied, so that workflow authors can diagnose unexpected configuration behaviour.

### 6.3 Step Extraction Algorithm

A conforming implementation **MUST** extract steps from the compiler-provided inputs as follows:

1. Read `harness.steps` from `config.json`. If present, each entry defines a named step with its agent assignment, parallel flag, and dependency list.
2. Read the prompt body from `prompt.txt`. The compiler splits the body on ATX heading boundaries (`##` or `###` level) and associates each section with a named step via `<!-- harness-step: <name> -->` annotations before writing `prompt.txt`; the harness consumes the resulting sections.
3. If `harness.steps` is absent or empty, the entire prompt body **MUST** be treated as a single step with no explicit agent or dependency.
4. Steps without a named entry in `harness.steps` are treated as sequential steps in document order.

### 6.4 Initial Prompt Context

The AW Harness **MUST NOT** inject any predefined or ambient context into agent sessions. There are no implicit files, skills, or instruction documents automatically added to a session's initial prompt.

A conforming implementation **MUST** source every item included in a session's initial prompt from one of the following explicitly declared origins:

- The step's own Markdown body (extracted per [Section 6.3](#63-step-extraction-algorithm)).
- Transcripts from upstream steps (passed via the DAG execution model in [Section 7](#7-dag-execution-model)).
- The agent's `system` prompt as declared under `harness.agents` in the frontmatter.
- Files or sub-workflows declared via the standard `imports:` frontmatter key and resolved by the compiler into agent files passed at invocation time.

A conforming implementation **MUST NOT** automatically load AGENTS.md files, `.github/agents/` entries, skills directories, or any other ambient repository files unless they are explicitly listed in `imports:`. This behavior is a deliberate divergence from engines such as `engine: copilot` that inject ambient context automatically.

> [!IMPORTANT]
> Workflow authors **MUST** explicitly declare every file or skill they wish the agent to reference using the `imports:` frontmatter key. Relying on ambient context that is auto-injected by other engines will produce a missing-context failure when running with `engine: aw`.

---

## 7. DAG Execution Model

### 7.1 DAG Construction

A conforming implementation **MUST** construct a DAG from the extracted steps as follows:

1. Create one node per extracted step.
2. For each step with a `depends` list, add a directed edge from each named dependency to that step.
3. Perform a cycle check. If a cycle is detected, the implementation **MUST** abort with exit code `2` and emit a diagnostic to stderr identifying the cycle.
4. Compute a topological order. Steps at the same depth **MAY** be executed in parallel.

### 7.2 Execution Algorithm

A conforming implementation **MUST** execute the DAG as follows:

> [!NOTE] Non-normative example illustrating the orchestration entry point.
>
> ```typescript
> // index.ts — entry point
> import { createAgentSession, SessionManager } from "@mariozechner/pi-coding-agent";
>
> async function main() {
>   const { configPath, promptPath } = parseArgs(process.argv);
>   const workflow = loadWorkflow(configPath, promptPath);
>   const dag = buildDAG(workflow);
>   const extensions = [
>     apiProxyProvider,
>     safeOutputsExtension,
>     costTrackerExtension,
>     steeringExtension,
>     repairExtension,
>     observabilityExtension,
>     checkpointExtension,
>   ];
>
>   for (const stepGroup of dag.executionOrder()) {
>     await Promise.all(stepGroup.map(async (step) => {
>       const { session } = await createAgentSession({
>         sessionManager: SessionManager.inMemory(),
>         extensions,
>         model: resolveModel(step.agent?.model || workflow.defaultModel),
>         systemPrompt: buildSystemPrompt(step),
>       });
>
>       const prompt = buildStepPrompt(step, transcripts);
>       await session.prompt(prompt);
>
>       transcripts[step.name] = captureTranscript(session);
>       session.dispose();
>     }));
>   }
> }
> ```

For each execution group in topological order:

1. The implementation **MUST** invoke `createAgentSession()` once per step (or once per agent for steps with `agents: [...]`).
2. The prompt passed to `session.prompt()` **MUST** be assembled from: (a) the step's prompt text (from `prompt.txt`), (b) transcripts from all upstream steps, and (c) the agent's system prompt, if defined.
3. The implementation **MUST** load all seven gh-aw Pi extensions (see [Section 8](#8-extensions)) into each session.
4. Steps within the same parallel group **MUST** be executed concurrently using `Promise.all()`.
5. After each session completes, the implementation **MUST** capture the session transcript for use by downstream steps.
6. After capturing the transcript, the implementation **MUST** call `session.dispose()`.
7. If the budget gate has been triggered (via the cost-tracker extension), the implementation **MUST NOT** launch further sessions and **MUST** exit with code `1`.

### 7.3 Step Execution Summary

The per-step execution sequence is:

```
For each step (respecting DAG order):
  1. Build prompt = step text (from prompt.txt) + upstream transcripts + system prompt
  2. Create Pi AgentSession with gh-aw extensions:
     - api-proxy provider registered
     - Safe-output tools registered
     - Steering, repair, cost, observability extensions active
     (MCP tools available as bash CLI commands via cli-proxy — no bridging needed)
  3. session.prompt() → Pi agent loop runs
  4. Extensions handle events (cost tracking, steering, observability)
  5. Capture transcript for downstream steps
  6. Budget gate check, checkpoint state
```

---

## 8. Extensions

All gh-aw-specific behavior **MUST** be packaged as Pi extensions. Each extension **MUST** be a standalone TypeScript module that exports a default function with signature `(pi: ExtensionAPI) => void | Promise<void>`.

The following seven extensions **MUST** be loaded into every `AgentSession` created by the harness.

### 8.1 Extension 1: api-proxy Provider

**Purpose:** Registers the gh-aw api-proxy as a custom Pi provider.

**Requirements:**

- The extension **MUST** call `pi.registerProvider()` with the api-proxy base URL and the `AWF_API_PROXY_TOKEN` environment variable as the API key reference.
- The extension **MUST** fetch the available model list from the api-proxy at startup, before any session begins.
- The extension **MUST** use `"openai-completions"` as the API type, as the api-proxy speaks the OpenAI completions protocol.
- The `AWF_API_PROXY_URL` environment variable **MUST** be used as the base URL, defaulting to `http://localhost:8080/v1`.

> [!NOTE] Non-normative example.
>
> ```typescript
> export default async function(pi: ExtensionAPI) {
>   const proxyUrl = process.env.AWF_API_PROXY_URL || "http://localhost:8080/v1";
>   const models = await fetchModelsFromProxy(proxyUrl);
>
>   pi.registerProvider("aw-proxy", {
>     baseUrl: proxyUrl,
>     apiKey: "AWF_API_PROXY_TOKEN",
>     api: "openai-completions",
>     models,
>   });
> }
> ```

### 8.2 Extension 2: Safe Outputs

**Purpose:** Registers safe-output tools (create-issue, create-pull-request, add-comment, etc.) and writes artifact files in the gh-aw safe-outputs format.

**Requirements:**

- The extension **MUST** register at minimum the `create_issue`, `create_pull_request`, and `add_comment` tools via `pi.registerTool()`.
- Each `execute` handler **MUST** write a safe-output artifact file in the format expected by the post-agent job, without performing any live GitHub API calls.
- The extension **MUST** subscribe to the `agent_end` Pi event to finalize the safe-output manifest.
- The extension **MUST** enforce any `max-count` limit declared in the workflow's `safe-outputs` frontmatter.

> [!NOTE] Non-normative example.
>
> ```typescript
> export default function(pi: ExtensionAPI) {
>   const config = loadSafeOutputsConfig();
>
>   pi.registerTool({
>     name: "create_issue",
>     description: "Create a GitHub issue (safe output)",
>     parameters: Type.Object({
>       title: Type.String(),
>       body: Type.String(),
>       labels: Type.Optional(Type.Array(Type.String())),
>     }),
>     async execute(toolCallId, params, signal) {
>       const artifact = buildSafeOutputArtifact("create-issue", params, config);
>       await writeSafeOutputArtifact(artifact);
>       return { content: [{ type: "text", text: `Issue queued: ${params.title}` }] };
>     },
>   });
>
>   // ... register other safe-output tools
>
>   pi.on("agent_end", async (event, ctx) => {
>     await finalizeSafeOutputManifest();
>   });
> }
> ```

### 8.3 Extension 3: Cost Tracker

**Purpose:** Monitors token usage and cost via Pi's event stream and enforces budget gates.

**Requirements:**

- The extension **MUST** subscribe to `turn_end` events and accumulate total cost from each turn.
- When accumulated cost reaches or exceeds `harness.budget.warn-at-percent`, the extension **MUST** inject a steering message via `ctx.agent.steer()` warning the agent to be concise.
- When accumulated cost reaches or exceeds the value corresponding to `harness.steering.budget-critical-percent`, the extension **MUST** call `ctx.agent.abort()`.
- The extension **MUST** subscribe to `agent_end` and emit a cost summary to the JSONL event stream.

> [!NOTE] Non-normative example.
>
> ```typescript
> export default function(pi: ExtensionAPI) {
>   const budget = loadBudgetConfig();
>   let totalCost = 0;
>
>   pi.on("turn_end", async (event, ctx) => {
>     totalCost += extractCostFromTurn(event);
>
>     const percent = (totalCost / budget.maxCostUsd) * 100;
>     if (percent >= budget.budgetCriticalPercent) {
>       ctx.agent.abort();
>     } else if (percent >= budget.budgetWarnPercent) {
>       ctx.agent.steer({
>         role: "user",
>         content: `⚠️ Budget: ${percent.toFixed(0)}% used. Be concise.`,
>         timestamp: Date.now(),
>       });
>     }
>   });
>
>   pi.on("agent_end", async () => {
>     emitCostSummary(totalCost);
>   });
> }
> ```

### 8.4 Extension 4: Steering (Resource Pressure)

**Purpose:** Monitors time remaining and budget, and injects steering messages via Pi's native `session.steer()`.

**Requirements:**

- The extension **MUST** subscribe to `agent_start` to record the session start time.
- The extension **MUST** subscribe to `turn_end` and compute elapsed time after each turn.
- When time remaining falls below `harness.steering.time-warning-minutes`, the extension **MUST** inject a warning steering message.
- When time remaining falls below `harness.steering.time-critical-minutes`, the extension **MUST** inject a critical steering message directing the agent to produce final output immediately.

> [!NOTE] Non-normative example.
>
> ```typescript
> export default function(pi: ExtensionAPI) {
>   const config = loadSteeringConfig();
>   let startTime: number;
>
>   pi.on("agent_start", async () => {
>     startTime = Date.now();
>   });
>
>   pi.on("turn_end", async (event, ctx) => {
>     const elapsed = (Date.now() - startTime) / 60000;
>     const remaining = config.timeoutMinutes - elapsed;
>
>     if (remaining <= config.timeCriticalMinutes) {
>       ctx.agent.steer({
>         role: "user",
>         content: `⚠️ CRITICAL: ${remaining.toFixed(0)}min left. Write final output NOW.`,
>         timestamp: Date.now(),
>       });
>     } else if (remaining <= config.timeWarningMinutes) {
>       ctx.agent.steer({
>         role: "user",
>         content: `⚠️ ${remaining.toFixed(0)}min remaining. Wrap up.`,
>         timestamp: Date.now(),
>       });
>     }
>   });
> }
> ```

### 8.5 Extension 5: Session Repair

**Purpose:** Detects broken tool calls and repairs the session via Pi's message history manipulation.

**Requirements:**

- The extension **MUST** subscribe to `tool_result` events and inspect results for corruption indicators.
- On detection of a corrupted tool result, the extension **MUST** truncate the broken messages from `ctx.agent.state.messages` and emit a repair event to the JSONL stream.
- The extension **MUST** subscribe to `agent_end` and attempt recovery if the error is classified as recoverable, by injecting a follow-up message containing a summary of prior progress.

> [!NOTE] Non-normative example.
>
> ```typescript
> export default function(pi: ExtensionAPI) {
>   pi.on("tool_result", async (event, ctx) => {
>     if (isCorruptedToolResult(event)) {
>       const messages = ctx.agent.state.messages;
>       const repaired = truncateBrokenMessages(messages);
>       ctx.agent.state.messages = repaired;
>       emitRepairEvent("truncate_and_resume", event.toolName);
>     }
>   });
>
>   pi.on("agent_end", async (event, ctx) => {
>     if (event.error && isRecoverableError(event.error)) {
>       const summary = await summarizeTranscript(ctx.agent.state.messages);
>       ctx.agent.followUp({
>         role: "user",
>         content: `Previous progress: ${summary}\nContinue from here.`,
>         timestamp: Date.now(),
>       });
>     }
>   });
> }
> ```

### 8.6 Extension 6: Observability

**Purpose:** Emits JSONL events to stderr and generates OTel spans.

**Requirements:**

- The extension **MUST** subscribe to `agent_start`, `turn_end`, `tool_execution_end`, and `agent_end` events.
- On each event, the extension **MUST** emit a corresponding JSONL record to stderr.
- If `observability.otlp.endpoint` is configured in the workflow frontmatter, the extension **MUST** create and close OTel spans for each step.
- OTel span attributes **MUST** include at minimum: step name, model, token counts, and cost.

> [!NOTE] Non-normative example.
>
> ```typescript
> export default function(pi: ExtensionAPI) {
>   pi.on("agent_start", async (event) => {
>     emitJsonl({ event: "step_start", step: currentStep, model: currentModel });
>     startOtelSpan(currentStep);
>   });
>
>   pi.on("turn_end", async (event) => {
>     recordOtelAttributes(event);
>   });
>
>   pi.on("tool_execution_end", async (event) => {
>     emitJsonl({ event: "tool_end", tool: event.toolName, duration: event.duration });
>   });
>
>   pi.on("agent_end", async (event) => {
>     emitJsonl({ event: "step_end", step: currentStep, tokens: event.tokens, cost: event.cost });
>     endOtelSpan(currentStep);
>   });
> }
> ```

### 8.7 Extension 7: Checkpoint

**Purpose:** Persists run state for long workflows, enabling resume from a prior checkpoint.

**Requirements:**

- When `harness.checkpoint: true` is set, the extension **MUST** subscribe to `agent_end` and persist the step name, completion status, session transcript, and accumulated cost.
- Checkpoint data **MUST** be stored in a location accessible across job retries (e.g., a Actions cache or artifact).
- An implementation **SHOULD** support a `--continue` invocation flag that resumes from the last successful checkpoint, skipping already-completed steps.

> [!NOTE] Non-normative example.
>
> ```typescript
> export default function(pi: ExtensionAPI) {
>   pi.on("agent_end", async (event, ctx) => {
>     await saveCheckpoint({
>       step: currentStep,
>       status: event.error ? "failed" : "done",
>       transcript: ctx.agent.state.messages,
>       cost: totalCost,
>     });
>   });
> }
> ```

---

## 9. Model Resolution

*(This section is non-normative.)*

The harness does not perform provider inference or model routing. That responsibility belongs to the api-proxy (`gh-aw-firewall/containers/api-proxy/model-resolver.js`), which runs as a sidecar in the gh-aw container.

### 9.1 Alias Resolution Flow

```
Harness (Pi SDK) → api-proxy → LLM provider
  model: "sonnet"           resolves to copilot/claude-sonnet-4.6
  model: "gpt-5-codex"      resolves to copilot/gpt-5.3-codex
  model: "copilot/gpt-4.1"  passes through as-is
```

### 9.2 Alias Configuration

Model aliases are configured in the api-proxy via the `AWF_MODEL_ALIASES` environment variable.

> [!NOTE] Non-normative example of alias configuration JSON.
>
> ```json
> {
>   "models": {
>     "sonnet": ["copilot/*sonnet*", "anthropic/*sonnet*"],
>     "gpt-5-codex": ["copilot/gpt-5*-codex", "openai/gpt-5*-codex"],
>     "": ["sonnet", "gpt-5*-codex"]
>   }
> }
> ```

**Key properties of the alias resolver:**

- `providerid/modelid` syntax with `*` wildcards.
- Recursive alias resolution with loop detection.
- Case-insensitive matching.
- Semver sorting (highest version first among candidates).
- Default policy via `""` key (applied when no model is specified in the workflow).

### 9.3 Implications for the Harness

- Pi SDK's `pi-ai` is configured with the api-proxy as an OpenAI-compatible custom provider (via `pi.registerProvider()` in Extension 1).
- The harness passes model name strings through as-is — aliases such as `"sonnet"` work without any harness-side resolution.
- No `provider:` field is needed in frontmatter — the proxy handles all routing.
- Per-step and per-agent model selection is accomplished by passing a different model name string to `createAgentSession()`.
- The harness inherits the api-proxy's full model catalog without maintaining its own.

> [!NOTE] Non-normative example of provider registration.
>
> ```typescript
> export default async function apiProxyProvider(pi: ExtensionAPI) {
>   const proxyUrl = process.env.AWF_API_PROXY_URL || "http://localhost:8080/v1";
>
>   pi.registerProvider("aw-proxy", {
>     baseUrl: proxyUrl,
>     apiKey: "AWF_API_PROXY_TOKEN",
>     api: "openai-completions",
>     models: await fetchAvailableModels(proxyUrl),
>   });
> }
> ```

---

## 10. Build and Deployment

*(This section is non-normative.)*

### 10.1 TypeScript Configuration

> [!NOTE] Non-normative example.
>
> ```jsonc
> // tsconfig.json
> {
>   "compilerOptions": {
>     "target": "es2024",           // Node 24 supports ES2024
>     "module": "es2022",
>     "lib": ["es2024"],
>     "moduleResolution": "bundler",
>     "strict": true,
>     "skipLibCheck": true,
>     "outDir": "dist",
>     "declaration": false
>   }
> }
> ```

### 10.2 Bundle Configuration

> [!NOTE] Non-normative example.
>
> ```typescript
> // build.ts — esbuild config
> import { build } from "esbuild";
>
> await build({
>   entryPoints: ["src/index.ts"],
>   bundle: true,
>   platform: "node",
>   target: "node24",
>   format: "cjs",                  // .cjs required by gh-aw harness validation
>   outfile: "dist/aw_harness.cjs",
>   external: [],                   // Bundle everything (no runtime npm install)
>   minify: false,                  // Keep readable for debugging in Actions logs
>   sourcemap: "inline",            // Debugging in CI
> });
> ```

### 10.3 Output Location

The bundled `aw_harness.cjs` is placed in `actions/setup/js/aw_harness.cjs`, alongside existing harnesses:

```
actions/setup/js/
├── copilot_harness.cjs       # Existing
├── claude_harness.cjs        # Existing
├── aw_harness.cjs            # NEW — bundled from aw-harness/
├── *.cjs                     # Other existing action scripts
└── *.test.cjs                # Tests
```

### 10.4 Project Structure

```
aw-harness/
├── package.json                  # deps: pi-coding-agent, pi-agent-core, pi-ai
├── tsconfig.json                 # target: es2024, module: es2022
├── build.ts                      # esbuild → dist/aw_harness.cjs
├── src/
│   ├── index.ts                  # Entry point: read config.json + prompt.txt → buildDAG → run sessions
│   ├── parser.ts                 # config.json + prompt.txt → steps + config
│   ├── planner.ts                # DAG construction, topological sort
│   ├── dag-runner.ts             # Orchestrate sessions (sequential + parallel)
│   ├── transcript.ts             # Inter-step data flow (save/load/summarize)
│   ├── context.ts                # Prompt assembly, compaction
│   └── extensions/               # gh-aw Pi extensions
│       ├── api-proxy-provider.ts # Register api-proxy as Pi provider
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

### 10.5 Testing

Tests use the same Vitest setup as the existing `actions/setup/js/` scripts:

- Unit tests for parser, planner, and each extension.
- Integration tests with mock Pi sessions (`SessionManager.inMemory()`).
- Tests co-located: `aw_harness.test.cjs` or in a `test/` subdirectory.

### 10.6 Build Integration

A `make aw-harness` Makefile target **SHOULD** be added that runs esbuild and copies the output to `actions/setup/js/aw_harness.cjs`.

### 10.7 Backwards Compatibility

| Scenario | Behavior |
|----------|----------|
| `engine: copilot` (existing) | Uses current `copilot_harness.cjs` — unchanged |
| `engine: claude` (existing) | Uses current Claude Code flow — unchanged |
| `engine: codex` (existing) | Uses current Codex flow — unchanged |
| `engine: aw` without `harness:` block | Single-step: entire body = one Pi session prompt |
| `engine: aw` with `harness:` block | Multi-step orchestration mode |
| `engine: aw` with `harness.steps` | Explicit DAG (parallel, depends, agent assignment) |
| `engine: aw` without `harness.agents` | All steps use `engine.model` |
| `engine: aw` + `cli-proxy: false` | **Ignored** — `cli-proxy` is always on for `engine: aw` |
| `engine: aw` + `tools.github.mode: remote` | **Overridden to `gh-proxy`** — Pi SDK requires `gh-proxy`; `remote` mode is not supported |

### 10.8 Implementation Plan

The following ordered work items describe the implementation sequence:

1. **Scaffold project** — Initialize TypeScript project in `aw-harness/`. Configure package.json with Pi SDK deps (`@mariozechner/pi-coding-agent`, `pi-agent-core`, `pi-ai`). Set up tsconfig for ES2024/Node 24. Configure esbuild bundle → `dist/aw_harness.cjs`.

2. **Implement api-proxy provider extension** — Pi extension that registers the api-proxy as a custom provider via `pi.registerProvider()`. Async factory fetches available models at startup. All model requests route through the proxy.

3. **Implement parser** — Read `config.json` (compiler-generated harness config) and `prompt.txt` (compiler-generated prompt body). Parse step sections from prompt text; resolve step configuration from `harness.steps`. Fall back to single-step mode when no step config is present.

4. **Implement DAG planner** — Topological sort, parallel group detection, sequential fallback. Validate no cycles, all agent/step references resolve.

5. **Implement safe-outputs extension** — Pi extension that registers safe-output tools (create-issue, create-pull-request, add-comment, etc.). Uses `pi.on("agent_end")` to finalize artifact manifest.

6. **Implement DAG runner** — Orchestrates multiple `createAgentSession()` calls. Sequential steps + `Promise.all()` for parallel groups. Passes gh-aw extensions to each session. Manages transcript flow between steps.

7. **Implement transcript manager** — Save step output to disk. Load for downstream steps. Support `summary` mode (use a Pi session to summarize) and `full` mode.

8. **Implement context engine** — Prompt assembly with priority ordering. Compaction via `none`, `sliding-window`, or `summarize`.

9. **Implement cost tracker extension** — Pi extension that monitors `turn_end` events for token/cost data. Enforces soft (steer warning) and hard (abort) budget gates.

10. **Implement steering extension** — Pi extension that monitors time/budget and injects user messages via `session.steer()` on `turn_end`.

11. **Implement repair extension** — Pi extension that detects broken tool calls via `tool_result` events. Repairs via message truncation or summarize-and-restart.

12. **Implement checkpoint extension** — Pi extension that persists step completion state on `agent_end`. Resume from checkpoint on `--continue`.

13. **Implement observability extension** — Pi extension that emits JSONL to stderr on agent/tool events. Generates OTel spans using `observability.otlp` config.

14. **Write tests** — Unit tests for parser, planner, each extension (mock `ExtensionAPI`). Integration tests with `createAgentSession()` + `SessionManager.inMemory()`.

15. **Write example workflows** — Three examples: single-step, multi-step sequential, multi-agent parallel with different models.

17. **Add build to Makefile** — Add `make aw-harness` target that runs esbuild and copies `aw_harness.cjs` to `actions/setup/js/`.

---

## 11. Security Considerations

**Mandatory proxy features.** The `gh-proxy` and `cli-proxy` features **MUST** always be active for `engine: aw`. MCP tools are available to agent sessions as CLI commands via `cli-proxy`; disabling it would make those tools inaccessible. Any attempt by a workflow author to disable either feature **MUST** be silently overridden (see [Section 6.2](#62-overrides-and-fixed-settings)).

**No direct LLM access.** The harness **MUST NOT** make direct calls to LLM provider APIs. All model requests **MUST** pass through the api-proxy, which enforces rate limits, budget caps, and access controls at the container boundary.

**Safe outputs isolation.** The safe-outputs extension **MUST NOT** perform live GitHub API calls during agent execution. All GitHub mutations **MUST** be expressed as artifact files processed by the post-agent job, which applies threat detection and validation before acting.

**Budget enforcement.** The cost-tracker extension provides a hard budget gate. A conforming implementation **MUST** abort the session if the cost exceeds the configured maximum, preventing runaway spending from misbehaving agents.

**Transcript confidentiality.** Transcripts captured for inter-step context **SHOULD** be stored only in memory or in ephemeral container storage. Implementations **SHOULD NOT** persist transcripts to external storage unless checkpointing is explicitly enabled.

**Token and secret handling.** The `AWF_API_PROXY_TOKEN` value **MUST NOT** be logged to stderr or embedded in JSONL events. Implementations **MUST** treat it as an opaque secret.

---

## 12. Privacy Considerations

*(This section is non-normative.)*

**Data residency.** All agent execution occurs within the gh-aw Actions container. No workflow content, prompts, or transcripts leave the container except via the api-proxy to the configured LLM provider endpoint, or via OTLP to the configured telemetry endpoint.

**Transcript retention.** Step transcripts held in memory for inter-step context are discarded when the harness process exits. If checkpointing is enabled, transcript data may be persisted to GitHub Actions artifacts; workflow authors **SHOULD** evaluate the sensitivity of transcript content before enabling checkpointing.

**Telemetry scope.** When `observability.otlp` is configured, OTel spans contain step names, model names, token counts, and cost data. They **SHOULD NOT** contain raw prompt or response text. Implementations **SHOULD** redact sensitive content from span attributes.

**Model provider data handling.** Prompt content is transmitted to the LLM provider as configured in the api-proxy. Workflow authors are responsible for ensuring that content transmitted to LLM providers complies with applicable data handling policies.

---

## 13. References

### 13.1 Normative References

**[RFC 2119]**
Bradner, S., "Key words for use in RFCs to Indicate Requirement Levels", BCP 14, RFC 2119, March 1997. <https://www.rfc-editor.org/rfc/rfc2119>

### 13.2 Informative References

**[Pi SDK]**
`@mariozechner/pi-coding-agent` — Pi agent SDK providing `createAgentSession()`, `Agent`, `AgentTool`, and `ExtensionAPI`.

**[pi-agent-core]**
Core agent loop, event dispatch, and message history management for Pi SDK.

**[pi-ai]**
Pi AI provider abstraction layer, supporting OpenAI-compatible backends.

**[esbuild]**
JavaScript/TypeScript bundler. <https://esbuild.github.io>

**[OpenTelemetry]**
OpenTelemetry specification for distributed tracing. <https://opentelemetry.io/docs/>

**[gh-aw]**
GitHub Agentic Workflows — the gh-aw CLI extension that compiles Markdown workflow files to GitHub Actions YAML. <https://github.com/github/gh-aw>
