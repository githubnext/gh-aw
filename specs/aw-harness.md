# AW Harness Specification

---

**Title:** AW Harness — Multi-Task Agentic Workflow Execution Engine

**Status:** Unofficial Draft

**Date:** 2025-07-14

**Editor:** GitHub gh-aw Team

---

## Abstract

This document specifies the **AW Harness** (`aw_harness.cjs`), a Node.js execution engine for the `engine: aw` mode of GitHub Agentic Workflows (gh-aw). The harness provides multi-task orchestration, multi-agent coordination, context engineering, cost tracking, and observability, built on top of the Pi agent SDK ecosystem. All gh-aw-specific capabilities are implemented as Pi extensions using Pi's native `ExtensionAPI` extensibility mechanism.

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

The existing gh-aw harnesses (`copilot_harness.cjs`, `claude_harness.cjs`) are thin retry loops around a single CLI invocation. As workflow complexity grows, authors need multi-task orchestration, parallel agent execution, per-task model selection, budget management, and structured observability — none of which the current harnesses provide.

The AW Harness introduces `engine: aw` as a new opt-in execution engine. It does not replace existing engines; `engine: copilot`, `engine: claude`, and `engine: codex` continue to operate unchanged via their current harnesses. The AW Harness is a Pi SDK application: it creates one `AgentSession` per workflow task, loads a fixed set of gh-aw Pi extensions into each session, and orchestrates sessions according to a DAG derived from the workflow's `harness:` frontmatter block and Markdown heading structure.

The harness is designed exclusively for the gh-aw Actions container environment. It assumes the firewall and MCP gateway are already running. AWF injects provider credentials into the container environment; the harness reads these credentials and passes them to Pi SDK directly.

### 1.1 Scope

This specification covers:

- The entry-point invocation contract for `aw_harness.cjs`.
- The frontmatter schema for `engine: aw` workflows.
- The task-extraction and DAG-construction algorithms.
- The normative requirements for each of the seven gh-aw Pi extensions.
- The model connection contract via provider environment variables.
- The build and deployment configuration.

This specification does not cover:

- The compilation of workflow Markdown to GitHub Actions YAML (handled by `gh-aw` proper).
- Safe-outputs post-processing and threat detection (handled by post-agent jobs, unchanged).
- The Pi SDK internals (`pi-agent-core`, `pi-ai`).
- LLM provider internals or credential rotation.

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
: The execution engine implemented in `aw_harness.cjs`, invoked when a workflow declares `engine: aw`. It is responsible for parsing the workflow, constructing the DAG, and orchestrating one `AgentSession` per task.

**AgentSession**
: A Pi SDK session object, obtained via `createAgentSession()`, that manages a single agent's message loop, tool calls, and event stream.

**api-proxy**
: A sidecar process in the gh-aw container used by other engines for model routing. The AW Harness does **not** use the api-proxy; it connects to LLM providers directly via environment variables.

**cli-proxy**
: A feature that mounts MCP servers as CLI tools on `PATH`, making them callable as ordinary shell commands within agent sessions.

**DAG (Directed Acyclic Graph)**
: The execution graph derived from the workflow's `harness.tasks` declarations and implicit document-order dependencies. Nodes are tasks; directed edges encode `depends` relationships. Cycles are prohibited.

**ExtensionAPI**
: The Pi SDK interface (`ExtensionAPI` from `@mariozechner/pi-coding-agent`) that a Pi extension receives as its sole argument. Provides `pi.registerTool()`, `pi.registerProvider()`, and `pi.on()`.

**gh-proxy**
: A feature that provides a pre-authenticated `gh` CLI binary in the agent's bash environment, enabling direct GitHub API access without separate token management.

**harness task**
: A unit of work within an `engine: aw` workflow. Each task is assigned one `AgentSession`, a prompt derived from the corresponding Markdown section, and optionally a named agent definition.

**MCP Gateway**
: The gh-aw MCP gateway process that exposes GitHub tools and custom MCP server tools as CLI commands (via `cli-proxy`) in the agent's bash environment. It runs independently of the harness in the same container.

**model alias**
: A short name (e.g., `"sonnet"`, `"gpt-5-codex"`) that Pi SDK resolves to a fully-qualified `provider/model` string using the provider registrations configured by Extension 1.

**Pi extension**
: A TypeScript module that exports a default function with signature `(pi: ExtensionAPI) => void | Promise<void>`. Loaded into an `AgentSession` to register tools, subscribe to events, or register providers.

**safe output**
: A deferred GitHub action (create issue, create pull request, add comment, etc.) expressed as an artifact file written during agent execution and processed by the post-agent job.

**task annotation**
: An HTML comment of the form `<!-- harness-task: <name> -->` embedded in a Markdown section to associate that section with a named entry in `harness.tasks`.

**transcript**
: The complete message history of a completed `AgentSession`, optionally summarized, passed as context to downstream tasks.

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
│  │  Container (firewall, MCP gateway)                   │   │
│  │                                                       │   │
│  │  ┌─────────────────────────────────────────────────┐ │   │
│  │  │  aw_harness.cjs (entry point)                   │ │   │
│  │  │                                                  │ │   │
│  │  │  1. Reads config.json + prompt.txt (pre-parsed by compiler) │ │   │
│  │  │  2. Parses tasks from task config + prompt text            │ │   │
│  │  │  3. Builds execution DAG                                   │ │   │
│  │  │  4. For each task: creates Pi AgentSession                 │ │   │
│  │  │     with gh-aw extensions loaded                           │ │   │
│  │  │  5. session.prompt() → Pi drives the agent                 │ │   │
│  │  │                                                  │ │   │
│  │  │  ┌──────────────────────────────────────────┐   │ │   │
│  │  │  │  Pi SDK (createAgentSession)             │   │ │   │
│  │  │  │  ├─ pi-agent-core (agent loop, events)   │   │ │   │
│  │  │  │  ├─ pi-ai → provider env vars → LLM providers │   │ │   │
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

3. **Direct provider connections.** Pi SDK **MUST** be configured to use the LLM provider credentials that AWF injects into the container environment. The harness **MUST NOT** interpose an additional proxy layer; provider routing and model selection **MUST** be handled by the Pi SDK and the provider-specific environment variables.

4. **Optimized for gh-aw container.** The harness **MUST** assume that the firewall and MCP gateway are already running. It **MUST NOT** perform redundant network configuration. MCP tools are available to agent sessions as bash CLI tools via `cli-proxy` — no additional bridging is required.

5. **`gh-proxy` and `cli-proxy` always on.** GitHub and other MCP server tools are available to the agent as CLI commands on `PATH` (via `cli-proxy`) and via the pre-authenticated `gh` binary (via `gh-proxy`). A conforming implementation **MUST** enable both `gh-proxy` and `cli-proxy` when `engine: aw` is selected. A conforming implementation **MUST NOT** honor attempts to disable these features for `engine: aw`, regardless of the values specified in the workflow frontmatter (see [Section 6.2](#62-overrides-and-fixed-settings)).

6. **TypeScript → Node 24.** Source **MUST** be TypeScript, compiled to ES2024, bundled via esbuild to a single `.cjs`. Leverages Node 24 features (native fetch, `structuredClone`, `AbortSignal.any`).

7. **Output in `actions/setup/js/`.** The bundled `aw_harness.cjs` **MUST** be placed in `actions/setup/js/aw_harness.cjs`, alongside `copilot_harness.cjs` and `claude_harness.cjs`. The same deployment mechanism and runtime contract apply.

8. **New opt-in engine.** `engine: aw` is an independent opt-in. Existing engines **MUST** be untouched.

9. **Markdown-native tasks.** `## Heading` or `### Heading` elements **MUST** be recognized as task boundaries. HTML comments carry task metadata.

10. **Observable.** All implementations **MUST** emit a JSONL event stream to stderr and **SHOULD** generate OTel spans when an OTLP endpoint is configured.

---

## 5. Harness Invocation Contract

### 5.1 Entry Point

The AW Harness **MUST** be invocable as a Node.js CommonJS module from the command line. The gh-aw compiler pre-processes the workflow markdown (parsing frontmatter, extracting the prompt body, resolving imports) and provides the harness with pre-built input files. A conforming invocation has the form:

```
node aw_harness.cjs --config <config-path> --prompt <prompt-path>
```

where:
- `<config-path>` is the path to the compiler-generated `config.json` file containing the parsed harness configuration (including resolved agent file paths).
- `<prompt-path>` is the path to the compiler-generated `prompt.txt` file containing the extracted prompt body.

A conforming implementation **MUST NOT** read or parse workflow Markdown files directly; all configuration and prompt content **MUST** be consumed from the pre-processed input files provided by the compiler.

### 5.2 Environment Variables

A conforming implementation **MUST** read LLM provider credentials from the container environment. AWF sets up the appropriate provider-specific environment variables (e.g., `ANTHROPIC_API_KEY`, `OPENAI_API_KEY`, `GITHUB_TOKEN`) for whichever providers are enabled in the workflow configuration. The harness **MUST NOT** hard-code any provider URL or token; it **MUST** rely exclusively on the environment injected by AWF.

A conforming implementation **SHOULD** read standard GitHub Actions environment variables (`GITHUB_REPOSITORY`, `GITHUB_RUN_ID`, etc.) for use in observability spans and checkpoint keys.

### 5.3 Exit Codes

| Code | Meaning |
|------|---------|
| `0` | All tasks completed successfully |
| `1` | One or more tasks failed (non-recoverable error) |
| `2` | Invocation error (missing config path, unreadable config file) |

A conforming implementation **MUST** exit with code `0` if and only if all DAG tasks complete without error. It **MUST** exit with a non-zero code on any unrecovered failure.

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
>   model: sonnet                  # Model alias — provider resolves
>
> permissions:
>   contents: read
>   issues: read
>   pull-requests: read
>
> # All files and skills the agent may reference MUST be declared here.
> # The compiler resolves each path and passes the contents to the harness.
> # Skills are files under skills/ and must be listed explicitly.
> imports:
>   - skills/reporting/SKILL.md    # Skill: formatting guidelines for reports
>   - shared/review-criteria.md    # Shared context: review checklist
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
>   tasks:
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
> <!-- harness-task: gather -->
> Use `git log --since="24 hours ago"` and `git diff` to collect all
> recent changes. Summarize the scope.
>
> ### Parallel Review
> <!-- harness-task: parallel-review -->
> Each reviewer examines the changes independently.
>
> ### Synthesize
> <!-- harness-task: synthesize -->
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
- `transcript-mode` (string): One of `full` or `summary`. Controls how upstream task transcripts are included in downstream prompts. Default: `full`.

#### 6.1.3 `harness.agents`

The `harness.agents` key is **OPTIONAL**. Each entry defines a named agent with:

- `model` (string, **REQUIRED**): Model alias or fully-qualified `provider/model` string.
- `system` (string, **OPTIONAL**): System prompt override for this agent.

#### 6.1.4 `harness.tasks`

The `harness.tasks` key is **OPTIONAL**. Each entry defines a named task with:

- `agent` (string, **OPTIONAL**): Name of an agent defined in `harness.agents`. Mutually exclusive with `agents`.
- `agents` (array of strings, **OPTIONAL**): Names of agents to run in parallel for this task. Mutually exclusive with `agent`.
- `parallel` (boolean, **OPTIONAL**): When `true` and `agents` is provided, all listed agents execute in parallel. Default: `false`.
- `depends` (array of strings, **OPTIONAL**): Names of tasks that **MUST** complete before this task begins.

#### 6.1.5 `harness.steering`

The `harness.steering` key is **OPTIONAL**. When present, it **MAY** contain:

- `time-warning-minutes` (number): Minutes before timeout at which a warning **SHOULD** be injected. Default: `5`.
- `time-critical-minutes` (number): Minutes before timeout at which a critical message **MUST** be injected. Default: `2`.
- `budget-warn-percent` (number): Budget percentage at which a warning **SHOULD** be injected. Default: `75`.
- `budget-critical-percent` (number): Budget percentage at which the session **MUST** be aborted. Default: `90`.

#### 6.1.6 `harness.checkpoint`

The `harness.checkpoint` key is **OPTIONAL**. When set to `true`, the checkpoint extension **MUST** persist task state on `agent_end`.

#### 6.1.7 `imports:`

The `imports:` key is **OPTIONAL**. It is a standard gh-aw frontmatter key that lists the paths of files whose contents **MUST** be resolved by the compiler and made available to the harness as part of the compiled inputs.

Each entry is a repository-relative path (string). Entries **MAY** point to:

- **Skill files** — files under `skills/` (e.g., `skills/reporting/SKILL.md`).
- **Shared context files** — markdown or text files shared across workflows (e.g., `shared/review-criteria.md`).
- **Agent files** — custom agent `.yml` files (resolved and embedded by the compiler).

A conforming implementation **MUST NOT** treat any skill, shared file, or agent file as implicitly available unless it appears in `imports:`. Skills directories are **NOT** auto-discovered or auto-loaded.

> [!NOTE] Non-normative example.
>
> ```yaml
> imports:
>   - skills/reporting/SKILL.md        # Skill: formatting guidelines
>   - skills/github-issue-query/SKILL.md  # Skill: querying GitHub issues
>   - shared/review-criteria.md        # Shared review checklist
> ```

### 6.2 Overrides and Fixed Settings

A conforming implementation **MUST** apply the following overrides regardless of values specified in the workflow frontmatter:

| Setting | Enforced value | Reason |
|---------|----------------|--------|
| `cli-proxy` | `true` | Required: MCP tools are exposed as CLI tools on `PATH` |
| `tools.github.mode` | `gh-proxy` | Pi SDK requires `gh-proxy`; `remote` mode is not supported |

A conforming implementation **MUST NOT** honor attempts to disable `cli-proxy` or set `tools.github.mode: remote` when `engine: aw` is active. These settings **MUST** be overridden. A conforming implementation **MUST** emit a warning to stderr when either override is applied, so that workflow authors can diagnose unexpected configuration behaviour.

### 6.3 Task Extraction Algorithm

A conforming implementation **MUST** extract tasks from the compiler-provided inputs as follows:

1. Read `harness.tasks` from `config.json`. If present, each entry defines a named task with its agent assignment, parallel flag, and dependency list.
2. Read the prompt body from `prompt.txt`. The compiler splits the body on ATX heading boundaries (`##` or `###` level) and associates each section with a named task via `<!-- harness-task: <name> -->` annotations before writing `prompt.txt`; the harness consumes the resulting sections.
3. If `harness.tasks` is absent or empty, the entire prompt body **MUST** be treated as a single task with no explicit agent or dependency.
4. Tasks without a named entry in `harness.tasks` are treated as sequential tasks in document order.

### 6.4 Initial Prompt Context

The AW Harness **MUST NOT** inject any predefined or ambient context into agent sessions. There are no implicit files, skills, or instruction documents automatically added to a session's initial prompt.

A conforming implementation **MUST** source every item included in a session's initial prompt from one of the following explicitly declared origins:

- The task's own Markdown body (extracted per [Section 6.3](#63-task-extraction-algorithm)).
- Transcripts from upstream tasks (passed via the DAG execution model in [Section 7](#7-dag-execution-model)).
- The agent's `system` prompt as declared under `harness.agents` in the frontmatter.
- Files, skills, and sub-workflows declared via the `imports:` frontmatter key (see [Section 6.1.7](#617-imports)) and resolved by the compiler into inputs passed at invocation time.

A conforming implementation **MUST NOT** automatically load AGENTS.md files, `.github/agents/` entries, skills directories, or any other ambient repository files unless they are explicitly listed in `imports:`. This behavior is a deliberate divergence from engines such as `engine: copilot` that inject ambient context automatically.

Skills **MUST** be treated as ordinary imported files: they carry no special runtime status and **MUST** be listed individually under `imports:` just like any other resource. There is no automatic discovery of skills based on directory presence or workflow content.

> [!IMPORTANT]
> Workflow authors **MUST** explicitly declare every file and skill they wish the agent to reference using the `imports:` frontmatter key. Relying on ambient context that is auto-injected by other engines will produce a missing-context failure when running with `engine: aw`.

> [!NOTE] Non-normative example.
>
> ```yaml
> # All skills and files must be declared explicitly.
> imports:
>   - skills/reporting/SKILL.md          # Skill: formatting guidelines
>   - skills/github-issue-query/SKILL.md # Skill: querying issues
>   - shared/pr-review-criteria.md       # Shared context: review checklist
> ```

---

## 7. DAG Execution Model

### 7.1 DAG Construction

A conforming implementation **MUST** construct a DAG from the extracted tasks as follows:

1. Create one node per extracted task.
2. For each task with a `depends` list, add a directed edge from each named dependency to that task.
3. Perform a cycle check. If a cycle is detected, the implementation **MUST** abort with exit code `2` and emit a diagnostic to stderr identifying the cycle.
4. Compute a topological order. Tasks at the same depth **MAY** be executed in parallel.

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
>     providerSetupExtension,
>     safeOutputsExtension,
>     costTrackerExtension,
>     steeringExtension,
>     repairExtension,
>     observabilityExtension,
>     checkpointExtension,
>   ];
>
>   for (const taskGroup of dag.executionOrder()) {
>     await Promise.all(taskGroup.map(async (task) => {
>       const { session } = await createAgentSession({
>         sessionManager: SessionManager.inMemory(),
>         extensions,
>         model: resolveModel(task.agent?.model || workflow.defaultModel),
>         systemPrompt: buildSystemPrompt(task),
>       });
>
>       const prompt = buildTaskPrompt(task, transcripts);
>       await session.prompt(prompt);
>
>       transcripts[task.name] = captureTranscript(session);
>       session.dispose();
>     }));
>   }
> }
> ```

For each execution group in topological order:

1. The implementation **MUST** invoke `createAgentSession()` once per task (or once per agent for tasks with `agents: [...]`).
2. The prompt passed to `session.prompt()` **MUST** be assembled from: (a) the task's prompt text (from `prompt.txt`), (b) transcripts from all upstream tasks, and (c) the agent's system prompt, if defined.
3. The implementation **MUST** load all seven gh-aw Pi extensions (see [Section 8](#8-extensions)) into each session.
4. Tasks within the same parallel group **MUST** be executed concurrently using `Promise.all()`.
5. After each session completes, the implementation **MUST** capture the session transcript for use by downstream tasks.
6. After capturing the transcript, the implementation **MUST** call `session.dispose()`.
7. If the budget gate has been triggered (via the cost-tracker extension), the implementation **MUST NOT** launch further sessions and **MUST** exit with code `1`.

### 7.3 Task Execution Summary

The per-task execution sequence is:

```
For each task (respecting DAG order):
  1. Build prompt = task text (from prompt.txt) + upstream transcripts + system prompt
  2. Create Pi AgentSession with gh-aw extensions:
     - Provider setup registered
     - Safe-output tools registered
     - Steering, repair, cost, observability extensions active
     (MCP tools available as bash CLI commands via cli-proxy — no bridging needed)
  3. session.prompt() → Pi agent loop runs
  4. Extensions handle events (cost tracking, steering, observability)
  5. Capture transcript for downstream tasks
  6. Budget gate check, checkpoint state
```

---

## 8. Extensions

All gh-aw-specific behavior **MUST** be packaged as Pi extensions. Each extension **MUST** be a standalone TypeScript module that exports a default function with signature `(pi: ExtensionAPI) => void | Promise<void>`.

The following seven extensions **MUST** be loaded into every `AgentSession` created by the harness.

### 8.1 Extension 1: Provider Setup

**Purpose:** Registers LLM providers with Pi SDK using credentials from the container environment.

**Requirements:**

- The extension **MUST** call `pi.registerProvider()` for each LLM provider whose credentials are present in the environment (e.g., `ANTHROPIC_API_KEY`, `OPENAI_API_KEY`, `GITHUB_TOKEN`).
- The extension **MUST NOT** hard-code provider URLs or API keys; all credentials **MUST** come from environment variables injected by AWF.
- The extension **MUST** register at least one provider before any session begins; if no provider credentials are found, the extension **MUST** fail with a descriptive error.

> [!NOTE] Non-normative example.
>
> ```typescript
> export default async function(pi: ExtensionAPI) {
>   if (process.env.ANTHROPIC_API_KEY) {
>     pi.registerProvider("anthropic", {
>       apiKey: process.env.ANTHROPIC_API_KEY,
>       api: "anthropic",
>     });
>   }
>   if (process.env.OPENAI_API_KEY) {
>     pi.registerProvider("openai", {
>       apiKey: process.env.OPENAI_API_KEY,
>       api: "openai-completions",
>     });
>   }
>   // Additional providers registered as their env vars are present
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
- If `observability.otlp.endpoint` is configured in the workflow frontmatter, the extension **MUST** create and close OTel spans for each task.
- OTel span attributes **MUST** include at minimum: task name, model, token counts, and cost.

> [!NOTE] Non-normative example.
>
> ```typescript
> export default function(pi: ExtensionAPI) {
>   pi.on("agent_start", async (event) => {
>     emitJsonl({ event: "task_start", task: currentTask, model: currentModel });
>     startOtelSpan(currentTask);
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
>     emitJsonl({ event: "task_end", task: currentTask, tokens: event.tokens, cost: event.cost });
>     endOtelSpan(currentTask);
>   });
> }
> ```

### 8.7 Extension 7: Checkpoint

**Purpose:** Persists run state for long workflows, enabling resume from a prior checkpoint.

**Requirements:**

- When `harness.checkpoint: true` is set, the extension **MUST** subscribe to `agent_end` and persist the task name, completion status, session transcript, and accumulated cost.
- Checkpoint data **MUST** be stored in a location accessible across job retries (e.g., a Actions cache or artifact).
- An implementation **SHOULD** support a `--continue` invocation flag that resumes from the last successful checkpoint, skipping already-completed tasks.

> [!NOTE] Non-normative example.
>
> ```typescript
> export default function(pi: ExtensionAPI) {
>   pi.on("agent_end", async (event, ctx) => {
>     await saveCheckpoint({
>       task: currentTask,
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

The harness does not perform provider inference or model routing directly. Pi SDK resolves model names using the providers registered by Extension 1. AWF injects provider-specific credentials into the container environment; the harness passes them through to Pi SDK, which handles the routing.

### 9.1 Model Selection Flow

```
Harness (Pi SDK) → Registered provider → LLM provider API
  model: "claude-sonnet-4.6"   → Anthropic (via ANTHROPIC_API_KEY)
  model: "gpt-5-codex"         → OpenAI (via OPENAI_API_KEY)
  model: "copilot/gpt-4.1"     → Copilot (via GITHUB_TOKEN)
```

### 9.2 Per-Task Model Selection

Per-task and per-agent model selection is accomplished by passing a different model name string to `createAgentSession()`. The harness reads the model name from `config.json` (compiled from `harness.agents[*].model` or the top-level `engine.model` field in the workflow frontmatter).

### 9.3 Implications for the Harness

- The harness passes model name strings through as-is to `createAgentSession()`.
- No `provider:` field is needed in frontmatter — Pi SDK selects the provider based on the model name and registered providers.
- The harness inherits the provider catalog determined by the env vars AWF injects.

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
│   ├── parser.ts                 # config.json + prompt.txt → tasks + config
│   ├── planner.ts                # DAG construction, topological sort
│   ├── dag-runner.ts             # Orchestrate sessions (sequential + parallel)
│   ├── transcript.ts             # Inter-task data flow (save/load/summarize)
│   ├── context.ts                # Prompt assembly, compaction
│   └── extensions/               # gh-aw Pi extensions
│       ├── provider-setup.ts     # Register LLM providers from env vars
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
| `engine: aw` without `harness:` block | Single-task: entire body = one Pi session prompt |
| `engine: aw` with `harness:` block | Multi-task orchestration mode |
| `engine: aw` with `harness.tasks` | Explicit DAG (parallel, depends, agent assignment) |
| `engine: aw` without `harness.agents` | All tasks use `engine.model` |
| `engine: aw` + `cli-proxy: false` | **Ignored** — `cli-proxy` is always on for `engine: aw` |
| `engine: aw` + `tools.github.mode: remote` | **Overridden to `gh-proxy`** — Pi SDK requires `gh-proxy`; `remote` mode is not supported |

### 10.8 Implementation Plan

The following ordered work items describe the implementation sequence:

1. **Scaffold project** — Initialize TypeScript project in `aw-harness/`. Configure package.json with Pi SDK deps (`@mariozechner/pi-coding-agent`, `pi-agent-core`, `pi-ai`). Set up tsconfig for ES2024/Node 24. Configure esbuild bundle → `dist/aw_harness.cjs`.

2. **Implement provider setup extension** — Pi extension that registers LLM providers via `pi.registerProvider()` using provider credentials injected by AWF into the container environment.

3. **Implement parser** — Read `config.json` (compiler-generated harness config) and `prompt.txt` (compiler-generated prompt body). Parse task sections from prompt text; resolve task configuration from `harness.tasks`. Fall back to single-task mode when no task config is present.

4. **Implement DAG planner** — Topological sort, parallel group detection, sequential fallback. Validate no cycles, all agent/task references resolve.

5. **Implement safe-outputs extension** — Pi extension that registers safe-output tools (create-issue, create-pull-request, add-comment, etc.). Uses `pi.on("agent_end")` to finalize artifact manifest.

6. **Implement DAG runner** — Orchestrates multiple `createAgentSession()` calls. Sequential tasks + `Promise.all()` for parallel groups. Passes gh-aw extensions to each session. Manages transcript flow between tasks.

7. **Implement transcript manager** — Save task output to disk. Load for downstream tasks. Support `summary` mode (use a Pi session to summarize) and `full` mode.

8. **Implement context engine** — Prompt assembly with priority ordering. Compaction via `none`, `sliding-window`, or `summarize`.

9. **Implement cost tracker extension** — Pi extension that monitors `turn_end` events for token/cost data. Enforces soft (steer warning) and hard (abort) budget gates.

10. **Implement steering extension** — Pi extension that monitors time/budget and injects user messages via `session.steer()` on `turn_end`.

11. **Implement repair extension** — Pi extension that detects broken tool calls via `tool_result` events. Repairs via message truncation or summarize-and-restart.

12. **Implement checkpoint extension** — Pi extension that persists task completion state on `agent_end`. Resume from checkpoint on `--continue`.

13. **Implement observability extension** — Pi extension that emits JSONL to stderr on agent/tool events. Generates OTel spans using `observability.otlp` config.

14. **Write tests** — Unit tests for parser, planner, each extension (mock `ExtensionAPI`). Integration tests with `createAgentSession()` + `SessionManager.inMemory()`.

15. **Write example workflows** — Three examples: single-task, multi-task sequential, multi-agent parallel with different models.

17. **Add build to Makefile** — Add `make aw-harness` target that runs esbuild and copies `aw_harness.cjs` to `actions/setup/js/`.

---

## 11. Security Considerations

**Mandatory proxy features.** The `gh-proxy` and `cli-proxy` features **MUST** always be active for `engine: aw`. MCP tools are available to agent sessions as CLI commands via `cli-proxy`; disabling it would make those tools inaccessible. Any attempt by a workflow author to disable either feature **MUST** be silently overridden (see [Section 6.2](#62-overrides-and-fixed-settings)).

**No direct LLM routing by harness.** The harness delegates all LLM routing to Pi SDK and the provider credentials injected by AWF. It **MUST NOT** perform additional proxy interception or credential manipulation.

**Safe outputs isolation.** The safe-outputs extension **MUST NOT** perform live GitHub API calls during agent execution. All GitHub mutations **MUST** be expressed as artifact files processed by the post-agent job, which applies threat detection and validation before acting.

**Budget enforcement.** The cost-tracker extension provides a hard budget gate. A conforming implementation **MUST** abort the session if the cost exceeds the configured maximum, preventing runaway spending from misbehaving agents.

**Transcript confidentiality.** Transcripts captured for inter-task context **SHOULD** be stored only in memory or in ephemeral container storage. Implementations **SHOULD NOT** persist transcripts to external storage unless checkpointing is explicitly enabled.

**Token and secret handling.** Provider credentials (e.g., `ANTHROPIC_API_KEY`, `OPENAI_API_KEY`, `GITHUB_TOKEN`) **MUST NOT** be logged to stderr or embedded in JSONL events. Implementations **MUST** treat all credential env vars as opaque secrets.

---

## 12. Privacy Considerations

*(This section is non-normative.)*

**Data residency.** All agent execution occurs within the gh-aw Actions container. No workflow content, prompts, or transcripts leave the container except via the Pi SDK to the configured LLM provider endpoint, or via OTLP to the configured telemetry endpoint.

**Transcript retention.** Task transcripts held in memory for inter-task context are discarded when the harness process exits. If checkpointing is enabled, transcript data may be persisted to GitHub Actions artifacts; workflow authors **SHOULD** evaluate the sensitivity of transcript content before enabling checkpointing.

**Telemetry scope.** When `observability.otlp` is configured, OTel spans contain task names, model names, token counts, and cost data. They **SHOULD NOT** contain raw prompt or response text. Implementations **SHOULD** redact sensitive content from span attributes.

**Model provider data handling.** Prompt content is transmitted to the LLM provider using the credentials AWF injects into the container. Workflow authors are responsible for ensuring that content transmitted to LLM providers complies with applicable data handling policies.

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
