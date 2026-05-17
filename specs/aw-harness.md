# AW Harness Specification

---

**Title:** AW Harness — Single-Session Agentic Workflow Execution Engine

**Status:** Working Draft

**Date:** 2025-07-14

**Last Updated:** 2026-05-10

**Editor:** GitHub gh-aw Team

---

## Abstract

This document specifies the **AW Harness** (`aw_harness.cjs`), a Node.js execution engine for the `engine: aw` mode of GitHub Agentic Workflows (gh-aw). The harness runs a single Pi `AgentSession` with a compiled prompt, budget management, steering, and observability. All gh-aw-specific capabilities are implemented as Pi extensions using Pi's native `ExtensionAPI` extensibility mechanism.

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
7. [Single-Session Execution Model](#7-single-session-execution-model)
8. [Extensions](#8-extensions)
9. [Model Resolution](#9-model-resolution)
10. [Build and Deployment](#10-build-and-deployment)
11. [Security Considerations](#11-security-considerations)
12. [Compliance Tests](#12-compliance-tests)
13. [Privacy Considerations](#13-privacy-considerations)
14. [References](#14-references)

---

## 1. Introduction

*(This section is non-normative.)*

The existing gh-aw harnesses (`copilot_harness.cjs`, `claude_harness.cjs`) are thin retry loops around a single CLI invocation. As workflow complexity grows, authors need structured budget management, cost tracking, time steering, and structured observability — none of which the current harnesses provide.

The AW Harness introduces `engine: aw` as a new opt-in execution engine. It does not replace existing engines; `engine: copilot`, `engine: claude`, and `engine: codex` continue to operate unchanged via their current harnesses. The AW Harness is a Pi SDK application: it creates a single `AgentSession` and runs the compiled prompt through it, with gh-aw Pi extensions providing budget gating, steering, and observability. Safe-output tools are CLI commands provided by `cli-proxy` and require no special Pi-level extension.

The harness is designed exclusively for the gh-aw Actions container environment. It assumes the firewall and MCP gateway are already running. AWF injects provider credentials into the container environment; the harness reads these credentials and passes them to Pi SDK directly.

### 1.1 Scope

This specification covers:

- The entry-point invocation contract for `aw_harness.cjs`.
- The frontmatter schema for `engine: aw` workflows.
- The prompt loading and session execution algorithm.
- The normative requirements for each of the six gh-aw Pi extensions.
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
: The execution engine implemented in `aw_harness.cjs`, invoked when a workflow declares `engine: aw`. It is responsible for loading the compiled prompt, creating a single Pi `AgentSession`, and running that session to completion.

**AgentSession**
: A Pi SDK session object, obtained via `createAgentSession()`, that manages a single agent's message loop, tool calls, and event stream.

**api-proxy**
: A sidecar process in the gh-aw container used by other engines for model routing. The AW Harness does **not** use the api-proxy; it connects to LLM providers directly via environment variables.

**cli-proxy**
: A feature that mounts MCP servers as CLI tools on `PATH`, making them callable as ordinary shell commands within the agent session.

**ExtensionAPI**
: The Pi SDK interface (`ExtensionAPI` from `@mariozechner/pi-coding-agent`) that a Pi extension receives as its sole argument. Provides `pi.registerTool()`, `pi.registerProvider()`, and `pi.on()`.

**gh-proxy**
: A feature that provides a pre-authenticated `gh` CLI binary in the agent's bash environment, enabling direct GitHub API access without separate token management.

**MCP Gateway**
: The gh-aw MCP gateway process that exposes GitHub tools and custom MCP server tools as CLI commands (via `cli-proxy`) in the agent's bash environment. It runs independently of the harness in the same container.

**model alias**
: A short name (e.g., `"sonnet"`, `"gpt-5-codex"`) that Pi SDK resolves to a fully-qualified `provider/model` string using the provider registrations configured by Extension 1.

**Pi extension**
: A TypeScript module that exports a default function with signature `(pi: ExtensionAPI) => void | Promise<void>`. Loaded into an `AgentSession` to register tools, subscribe to events, or register providers.

**safe output**
: A deferred GitHub action (create issue, create pull request, add comment, etc.) expressed as an artifact file written during agent execution and processed by the post-agent job.

**workflow document**
: A Markdown file with YAML frontmatter that declares an `engine: aw` workflow. The frontmatter is parsed by the gh-aw compiler at compile time; the harness itself never reads the raw Markdown file. Instead, the compiler provides the harness with pre-processed inputs: `config.json` (harness configuration) and `prompt.txt` (extracted prompt body).

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
│  │  │  2. Creates a single Pi AgentSession                       │ │   │
│  │  │     with gh-aw extensions loaded                           │ │   │
│  │  │  3. session.prompt() → Pi drives the agent                 │ │   │
│  │  │                                                  │ │   │
│  │  │  ┌──────────────────────────────────────────┐   │ │   │
│  │  │  │  Pi SDK (createAgentSession)             │   │ │   │
│  │  │  │  ├─ pi-agent-core (agent loop, events)   │   │ │   │
│  │  │  │  ├─ pi-ai → provider env vars → LLM providers │   │ │   │
│  │  │  │  └─ compaction, steering, auto-retry      │   │ │   │
│  │  │  └──────────────────────────────────────────┘   │ │   │
│  │  │  ┌──────────────────────────────────────────┐   │ │   │
│  │  │  │  gh-aw Pi Extensions (loaded into the    │   │ │   │
│  │  │  │  AgentSession via ExtensionAPI):          │   │ │   │
│  │  │  │  ├─ cost-tracker (budget gates + events)  │   │ │   │
│  │  │  │  ├─ steering (time/budget pressure)       │   │ │   │
│  │  │  │  ├─ repair (broken session recovery)      │   │ │   │
│  │  │  │  └─ observability (JSONL + OTel)          │   │ │   │
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

9. **Observable.** All implementations **MUST** emit a JSONL event stream to stderr and **SHOULD** generate OTel spans when an OTLP endpoint is configured.

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

A conforming implementation **SHOULD** read standard GitHub Actions environment variables (`GITHUB_REPOSITORY`, `GITHUB_RUN_ID`, etc.) for use in observability spans.

### 5.3 Exit Codes

| Code | Meaning |
|------|---------|
| `0` | Prompt completed successfully |
| `1` | Session failed (non-recoverable error) |
| `2` | Invocation error (missing config path, unreadable config file) |

A conforming implementation **MUST** exit with code `0` if and only if the agent session completes without error. It **MUST** exit with a non-zero code on any unrecovered failure.

### 5.4 Standard Streams

- **stdout**: Reserved for structured output (e.g., JSON summaries). A conforming implementation **SHOULD NOT** write diagnostic messages to stdout.
- **stderr**: All diagnostic messages, JSONL event stream, and debug output **MUST** be written to stderr.
- **GitHub Actions step summary** (`$GITHUB_STEP_SUMMARY`): The harness **MUST** write a Markdown-formatted execution summary to the file path indicated by the `GITHUB_STEP_SUMMARY` environment variable when that variable is set. The summary **MUST** be valid GitHub-flavored Markdown so that it renders correctly in the GitHub Actions step summary UI.

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
>     max-effective-tokens: 100000
>
>   context:
>     compaction: summarize
>     compaction-threshold: 0.75
>
>   steering:
>     time-warning-minutes: 5
>     time-critical-minutes: 2
>     budget-warn-percent: 75
>     budget-critical-percent: 90
>
>   # Optional: Pi SDK-compatible extensions to load alongside built-in gh-aw extensions.
>   # Each entry is a repo-relative path to a compiled .cjs file or an npm package name.
>   extensions:
>     - ./extensions/custom-tool.cjs
> ---
>
> Review all changes pushed to the default branch in the last 24 hours.
> Use `git log --since="24 hours ago"` and `git diff` to collect changes,
> review them for correctness, security, and maintainability, then create
> a GitHub issue with a prioritized findings list.
> ```

#### 6.1.1 `harness.budget`

The `harness.budget` key is **OPTIONAL**. When present, it **MUST** contain:

- `max-effective-tokens` (number): Maximum effective token count for the run. The cost-tracker extension **MUST** abort the current session if this limit is exceeded. Using token count rather than cost makes this budget reliable across providers where pricing is unknown.

#### 6.1.2 `harness.context`

The `harness.context` key is **OPTIONAL**. When present, it **MAY** contain:

- `compaction` (string): One of `none`, `sliding-window`, or `summarize`. Default: `none`.
- `compaction-threshold` (number, 0–1): Context fill fraction at which compaction triggers. Default: `0.75`.

#### 6.1.3 `harness.steering`

The `harness.steering` key is **OPTIONAL**. When present, it **MAY** contain:

- `time-warning-minutes` (number): Minutes before timeout at which a warning **SHOULD** be injected. Default: `5`.
- `time-critical-minutes` (number): Minutes before timeout at which a critical message **MUST** be injected. Default: `2`.
- `budget-warn-percent` (number): Budget percentage at which a warning **SHOULD** be injected. Default: `75`.
- `budget-critical-percent` (number): Budget percentage at which the session **MUST** be aborted. Default: `90`.

#### 6.1.4 `harness.extensions`

The `harness.extensions` key is **OPTIONAL**. When present, it **MUST** be a list of Pi SDK-compatible extension references that the harness loads and registers into the `AgentSession` at runtime, in addition to the built-in gh-aw extensions.

Each entry is a string in one of the following forms:

- **Repository-relative path** — a path starting with `./` or `../` pointing to a compiled CommonJS file (`.cjs`) co-located with the workflow (e.g., `./extensions/my-tool.cjs`).
- **npm package name** — a bare or scoped npm package name (e.g., `@my-org/my-pi-extension`) that is available in the container `node_modules` at runtime.

Each referenced module **MUST** export a default function with the Pi SDK `ExtensionAPI` signature:

```typescript
export default function(pi: ExtensionAPI): void | Promise<void>;
```

A conforming implementation **MUST**:

- Dynamically load each extension using `require()` (for local paths) or `require()` / `import()` (for npm packages).
- Register each user extension into the `AgentSession` alongside the built-in gh-aw extensions, after all built-in extensions have been registered.
- Emit a warning to stderr if an extension fails to load, and **MUST NOT** abort the session for a failed user extension unless `harness.extensions-required: true` is set.

A conforming implementation **MUST NOT** allow user extensions to override or replace the built-in gh-aw extensions. User extensions run after all built-in extensions and **MUST NOT** be able to unregister or intercept built-in extension behavior.

> [!NOTE] Non-normative example.
>
> ```yaml
> harness:
>   budget:
>     max-effective-tokens: 100000
>   extensions:
>     - ./extensions/custom-tool.cjs       # Local compiled extension
>     - @my-org/pi-extension-rate-limiter  # npm package extension
> ```
>
> Each extension module exports a Pi SDK-compatible function:
>
> ```typescript
> // extensions/custom-tool.cjs (compiled from TypeScript)
> module.exports = function(pi) {
>   pi.registerTool({
>     name: "my_custom_tool",
>     description: "Does something custom",
>     parameters: { type: "object", properties: { input: { type: "string" } } },
>     execute: async (_id, params) => ({ content: [{ type: "text", text: params.input }] }),
>   });
> };
> ```

#### 6.1.5 `imports:`

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

### 6.3 Prompt Loading

A conforming implementation **MUST** load the prompt from the compiler-provided inputs as follows:

1. Read `config.json` to obtain the harness configuration (budget, context, steering settings).
2. Read the prompt body from `prompt.txt`. The entire contents of `prompt.txt` constitute the initial user message passed to the single `AgentSession`.
3. Prepend the contents of any files resolved from `imports:` (as declared in the workflow frontmatter and compiled into the inputs) to the prompt body or supply them as system context, in the order declared.

A conforming implementation **MUST NOT** split or subdivide `prompt.txt` into sub-tasks. The prompt is treated as a single, atomic instruction to the agent.

### 6.4 Initial Prompt Context

The AW Harness **MUST NOT** inject any predefined or ambient context into the agent session. There are no implicit files, skills, or instruction documents automatically added to the session's initial prompt.

A conforming implementation **MUST** source every item included in the session's initial prompt from one of the following explicitly declared origins:

- The Markdown body from `prompt.txt` (loaded per [Section 6.3](#63-prompt-loading)).
- Files, skills, and sub-workflows declared via the `imports:` frontmatter key (see [Section 6.1.5](#615-imports)) and resolved by the compiler into inputs passed at invocation time.

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

## 7. Single-Session Execution Model

### 7.1 Execution Algorithm

A conforming implementation **MUST** execute the workflow as follows:

> [!NOTE] Non-normative example illustrating the execution entry point.
>
> ```typescript
> // index.ts — entry point
> import { createAgentSession, SessionManager } from "@mariozechner/pi-coding-agent";
>
> async function main() {
>   const { configPath, promptPath } = parseArgs(process.argv);
>   const { config, prompt } = loadInputs(configPath, promptPath);
>
>   // Load user-declared extensions from harness.extensions
>   const userExtensions = await loadUserExtensions(config.harness?.extensions ?? []);
>
>   const extensions = [
>     providerSetupExtension,
>     costTrackerExtension,
>     steeringExtension,
>     repairExtension,
>     observabilityExtension,
>     ...userExtensions,  // User extensions run after built-in extensions
>   ];
>
>   const { session } = await createAgentSession({
>     sessionManager: SessionManager.inMemory(),
>     extensions,
>     model: config.model,
>   });
>
>   await session.prompt(prompt);
>   session.dispose();
> }
> ```

1. The implementation **MUST** invoke `createAgentSession()` exactly once per harness invocation.
2. The prompt passed to `session.prompt()` **MUST** be the full contents of `prompt.txt` as loaded per [Section 6.3](#63-prompt-loading).
3. The implementation **MUST** load all five gh-aw Pi extensions (see [Section 8](#8-extensions)) into the session. If `harness.extensions` is declared in the configuration, the implementation **MUST** also load each user-declared extension after the built-in extensions (see [Section 6.1.4](#614-harness-extensions)).
4. After the session completes (success or failure), the implementation **MUST** call `session.dispose()`.
5. If the budget gate has been triggered (via the cost-tracker extension), the implementation **MUST** exit with code `1`.

### 7.2 Execution Summary

```
1. Load config.json + prompt.txt
2. Create single Pi AgentSession with gh-aw extensions:
   - Provider setup registered
   - Steering, repair, cost, observability extensions active
   (MCP tools and safe-output tools available as bash CLI commands via cli-proxy — no bridging needed)
   - User extensions (from harness.extensions) loaded and registered after built-ins
3. session.prompt(promptText) → Pi agent loop runs
4. Extensions handle events (cost tracking, steering, observability)
5. session.dispose()
```

---

## 8. Extensions

All gh-aw-specific behavior **MUST** be packaged as Pi extensions. Each extension **MUST** be a standalone TypeScript module that exports a default function with signature `(pi: ExtensionAPI) => void | Promise<void>`.

The following five extensions **MUST** be loaded into the `AgentSession` created by the harness.

### 8.1 Extension 1: Provider Setup

**Purpose:** Registers LLM providers with Pi SDK using credentials from the container environment.

**Requirements:**

- The extension **MUST** call `pi.registerProvider()` for each LLM provider whose credentials are present in the environment (e.g., `ANTHROPIC_API_KEY`, `OPENAI_API_KEY`, `GITHUB_TOKEN`).
- The extension **MUST** also check for provider-specific base URL environment variables (e.g., `ANTHROPIC_BASE_URL`, `OPENAI_BASE_URL`) and, when present, use them as the endpoint for the corresponding provider.
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
>       ...(process.env.ANTHROPIC_BASE_URL ? { baseUrl: process.env.ANTHROPIC_BASE_URL } : {}),
>     });
>   }
>   if (process.env.OPENAI_API_KEY) {
>     pi.registerProvider("openai", {
>       apiKey: process.env.OPENAI_API_KEY,
>       api: "openai-completions",
>       ...(process.env.OPENAI_BASE_URL ? { baseUrl: process.env.OPENAI_BASE_URL } : {}),
>     });
>   }
>   // Additional providers registered as their env vars are present
> }
> ```

### 8.2 Extension 2: Cost Tracker

**Purpose:** Monitors token usage via Pi's event stream and enforces budget gates.

**Requirements:**

- The extension **MUST** subscribe to `turn_end` events and accumulate total token usage from each turn.
- When accumulated tokens reach or exceed `harness.steering.budget-warn-percent` of `harness.budget.max-effective-tokens`, the extension **MUST** inject a steering message via `ctx.agent.steer()` warning the agent to be concise.
- When accumulated tokens reach or exceed `harness.steering.budget-critical-percent` of `harness.budget.max-effective-tokens`, the extension **MUST** call `ctx.agent.abort()`.

> [!NOTE] Non-normative example.
>
> ```typescript
> export default function(pi: ExtensionAPI) {
>   const budget = loadBudgetConfig();
>   let totalTokens = 0;
>
>   pi.on("turn_end", async (event, ctx) => {
>     totalTokens += extractTokensFromTurn(event);
>
>     const percent = (totalTokens / budget.maxEffectiveTokens) * 100;
>     if (percent >= budget.budgetCriticalPercent) {
>       ctx.agent.abort();
>     } else if (percent >= budget.budgetWarnPercent) {
>       ctx.agent.steer({
>         role: "user",
>         content: `⚠️ Token budget: ${percent.toFixed(0)}% used (${totalTokens}/${budget.maxEffectiveTokens}). Be concise.`,
>         timestamp: Date.now(),
>       });
>     }
>   });
> }
> ```

### 8.3 Extension 3: Steering (Resource Pressure)

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

### 8.4 Extension 4: Session Repair

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

### 8.5 Extension 5: Observability

**Purpose:** Emits structured event streams to stderr, writes a context provenance file for downstream analysis, renders a Markdown step summary, and reports per-turn token consumption.

**Requirements:**

#### 8.5.1 JSONL Event Stream

- The extension **MUST** subscribe to all Pi SDK agent events and, on each event, emit a corresponding JSONL record to stderr.
- If `observability.otlp.endpoint` is configured in the workflow frontmatter, the extension **MUST** create and close OTel spans for the session.
- OTel span attributes **MUST** include at minimum: model, token counts, and cost.

#### 8.5.2 Context Provenance File

- The extension **MUST** produce a context provenance file at a well-known path (e.g., `/tmp/gh-aw/context-provenance.jsonl`) when the session completes.
- The file **MUST** contain one JSON record per context entry added to the session, in chronological order. Each record **MUST** include:
  - `timestamp` (ISO 8601 string): When the entry was added.
  - `source` (string): The declared origin of the text — one of `"prompt"` (from `prompt.txt`) or `"import"` (from an `imports:` file, with `path` sub-field).
  - `path` (string, **OPTIONAL**): Repository-relative path for `"import"` entries.
  - `tokens` (number): Estimated token count for this entry at the time it was added.
  - `cumulative_tokens` (number): Running total of tokens in the context window at the time of this entry.
  - `role` (string): The message role — `"user"`, `"assistant"`, or `"system"`.
- The purpose of this file is to allow downstream tools (e.g., `gh aw audit`) to perform deep analysis of context growth, identify which imports consumed the most token budget, and diagnose context-window pressure.

#### 8.5.3 GitHub Actions Step Summary

- When the `GITHUB_STEP_SUMMARY` environment variable is set, the extension **MUST** write a Markdown-formatted execution summary to the file at that path.
- The summary **MUST** be valid GitHub-flavored Markdown so that it renders correctly in the GitHub Actions step summary UI.
- The summary **MUST** include at minimum:
  - A header identifying the workflow and model used.
  - A table showing per-turn token consumption (input tokens, output tokens, cumulative total, and estimated cost).
  - A final row with session totals (total tokens, total cost, elapsed time).
  - A context provenance section listing each `imports:` file with its token contribution.

#### 8.5.4 Per-Turn Token Consumption Output

- The extension **MUST** subscribe to `turn_end` events and emit a human-readable token consumption line to stderr after each turn.
- The line **MUST** report: turn number, input tokens, output tokens, cumulative total tokens, and estimated cumulative cost.
- The line **MUST** be formatted as valid GitHub-flavored Markdown (e.g., using a `>` blockquote prefix) so that it renders correctly when appended to the step summary.

> [!NOTE] Non-normative examples.
>
> **JSONL event (turn_end):**
> ```json
> {"event":"turn_end","turn":3,"input_tokens":4200,"output_tokens":850,"cumulative_tokens":15320,"cumulative_cost_usd":0.0412,"model":"claude-sonnet-4.6","ts":"2026-05-02T10:30:00.000Z"}
> ```
>
> **Context provenance record:**
> ```json
> {"timestamp":"2026-05-02T10:29:00.000Z","source":"import","path":"skills/reporting/SKILL.md","tokens":1240,"cumulative_tokens":1240,"role":"user"}
> {"timestamp":"2026-05-02T10:29:00.001Z","source":"prompt","tokens":520,"cumulative_tokens":1760,"role":"user"}
> ```
>
> **Step summary (excerpt):**
> ```markdown
> ## AW Harness Run — `claude-sonnet-4.6`
>
> | Turn | Input Tokens | Output Tokens | Cumulative | Est. Cost |
> |------|-------------|---------------|------------|-----------|
> | 1    | 1,760       | 420           | 2,180      | $0.0058   |
> | 2    | 2,180       | 640           | 2,820      | $0.0076   |
> | **Total** | | | **2,820** | **$0.0076** |
>
> ### Context Provenance
> | Source | Path | Tokens |
> |--------|------|--------|
> | import | skills/reporting/SKILL.md | 1,240 |
> | prompt | _(prompt.txt)_ | 520 |
> ```
>
> **Implementation sketch:**
>
> ```typescript
> export default function(pi: ExtensionAPI) {
>   let turnCount = 0;
>   let cumulativeTokens = 0;
>   let cumulativeCost = 0;
>   const provenanceLog: ProvenanceEntry[] = [];
>
>   pi.on("agent_start", async (event) => {
>     emitJsonl({ event: "session_start", model: currentModel });
>     startOtelSpan("aw_session");
>     recordContextProvenance(provenanceLog); // records imports + prompt entries
>   });
>
>   pi.on("turn_end", async (event, ctx) => {
>     turnCount++;
>     cumulativeTokens += event.inputTokens + event.outputTokens;
>     cumulativeCost += event.costUsd ?? 0;
>     emitJsonl({
>       event: "turn_end",
>       turn: turnCount,
>       input_tokens: event.inputTokens,
>       output_tokens: event.outputTokens,
>       cumulative_tokens: cumulativeTokens,
>       cumulative_cost_usd: cumulativeCost,
>       model: currentModel,
>       ts: new Date().toISOString(),
>     });
>     // Human-readable per-turn line to stderr (markdown blockquote)
>     process.stderr.write(
>       `> **Turn ${turnCount}**: ${event.inputTokens} in / ${event.outputTokens} out ` +
>       `| cumulative ${cumulativeTokens.toLocaleString()} tokens ($${cumulativeCost.toFixed(4)})\n`
>     );
>     recordOtelAttributes(event);
>   });
>
>   pi.on("tool_execution_end", async (event) => {
>     emitJsonl({ event: "tool_end", tool: event.toolName, duration: event.duration });
>   });
>
>   pi.on("agent_end", async (event) => {
>     emitJsonl({ event: "session_end", tokens: cumulativeTokens, cost: cumulativeCost });
>     endOtelSpan("aw_session");
>     await writeContextProvenanceFile(provenanceLog);
>     await writeStepSummary({ turnCount, cumulativeTokens, cumulativeCost, provenanceLog });
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

### 9.2 Model Selection

The model name is read from `config.json` (compiled from the top-level `engine.model` field in the workflow frontmatter) and passed as-is to `createAgentSession()`.

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
│   ├── index.ts                  # Entry point: read config.json + prompt.txt → create session → run
│   ├── loader.ts                 # config.json + prompt.txt → config + prompt string
│   ├── context.ts                # Prompt assembly, compaction
│   └── extensions/               # gh-aw Pi extensions
│       ├── provider-setup.ts     # Register LLM providers from env vars
│       ├── cost-tracker.ts       # Budget gates via turn_end events
│       ├── steering.ts           # Time/budget pressure via session.steer()
│       ├── repair.ts             # Broken session recovery
│       └── observability.ts      # JSONL events + OTel spans
├── test/
│   ├── loader.test.ts
│   ├── extensions/
│   │   ├── cost-tracker.test.ts
│   │   ├── steering.test.ts
│   │   ├── repair.test.ts
│   │   └── ...
│   └── integration/
│       └── harness.test.ts
└── dist/
    └── aw_harness.cjs            # → copied to actions/setup/js/
```

### 10.5 Testing

Tests use the same Vitest setup as the existing `actions/setup/js/` scripts:

- **Unit: session execution orchestration** — Unit tests **MUST** validate that the harness entry point loads `config.json` and `prompt.txt`, creates exactly one session, passes prompt content once, and disposes the session on success and on failure.
- **Integration: exit-code contract** — Integration tests **MUST** execute `node aw_harness.cjs --config ... --prompt ...` with controlled fixtures to assert exit codes `0`, `1`, and `2` for success, session failure, and invocation failure respectively.
- **Mock harness contracts** — Extension-facing tests **SHOULD** use mocked `ExtensionAPI` and mocked `SessionManager.inMemory()` sessions to verify:
  - budget abort behavior (`budget_exceeded` event + exit code `1`)
  - steer-message emission thresholds
  - extension load failure handling (`extensions-required: false` warning vs `true` fatal)
- Tests are RECOMMENDED to remain co-located as `aw_harness.test.cjs` plus `test/extensions/*.test.ts` to align with existing JS harness patterns.

### 10.6 Build Integration

A `make aw-harness` Makefile target **SHOULD** be added that runs esbuild and copies the output to `actions/setup/js/aw_harness.cjs`.

### 10.7 Backwards Compatibility

| Scenario | Behavior |
|----------|----------|
| `engine: copilot` (existing) | Uses current `copilot_harness.cjs` — unchanged |
| `engine: claude` (existing) | Uses current Claude Code flow — unchanged |
| `engine: codex` (existing) | Uses current Codex flow — unchanged |
| `engine: gemini` (existing) | Uses current Gemini flow — unchanged |
| `engine: opencode` (existing) | Uses current OpenCode flow — unchanged |
| `engine: crush` (existing) | Uses current Crush flow — unchanged |
| `engine: aw` | Single-session: entire `prompt.txt` = one Pi session prompt |
| `engine: aw` without `harness:` block | Uses defaults for budget/steering/compaction |
| `engine: aw` + `cli-proxy: false` | **Ignored** — `cli-proxy` is always on for `engine: aw` |
| `engine: aw` + `tools.github.mode: remote` | **Overridden to `gh-proxy`** — Pi SDK requires `gh-proxy`; `remote` mode is not supported |

### 10.8 Implementation Plan

The following ordered work items describe the implementation sequence:

1. **Scaffold project** — Initialize TypeScript project in `aw-harness/`. Configure package.json with Pi SDK deps (`@mariozechner/pi-coding-agent`, `pi-agent-core`, `pi-ai`). Set up tsconfig for ES2024/Node 24. Configure esbuild bundle → `dist/aw_harness.cjs`.

2. **Implement provider setup extension** — Pi extension that registers LLM providers via `pi.registerProvider()` using provider credentials injected by AWF into the container environment. Also detects provider-specific base URL env vars (e.g., `ANTHROPIC_BASE_URL`, `OPENAI_BASE_URL`) and uses them as the provider endpoint when present.

3. **Implement loader** — Read `config.json` (compiler-generated harness config) and `prompt.txt` (compiler-generated prompt body). Return config and prompt string to the entry point.

4. **Implement entry point** — Create a single `createAgentSession()` with gh-aw extensions loaded. Pass `prompt.txt` contents as the prompt. Dispose session on completion.

5. **Implement user extension loader** — Read `harness.extensions` from `config.json`. For each entry: resolve repository-relative paths from the harness working directory; load npm package names via `require()`. Verify each loaded module exports a default function of type `(pi: ExtensionAPI) => void`. Emit a stderr warning for each failed load; abort only if `harness.extensions-required: true` is set. Append loaded extensions to the session extension list after built-ins.

6. **Implement context engine** — Prompt assembly with priority ordering. Compaction via `none`, `sliding-window`, or `summarize`.

7. **Implement cost tracker extension** — Pi extension that monitors `turn_end` events for effective token usage. Enforces soft (steer warning) and hard (abort) budget gates against `harness.budget.max-effective-tokens`.

8. **Implement steering extension** — Pi extension that monitors time/budget and injects user messages via `session.steer()` on `turn_end`.

9. **Implement repair extension** — Pi extension that detects broken tool calls via `tool_result` events. Repairs via message truncation or summarize-and-restart.

10. **Implement observability extension** — Pi extension that:
    - Emits JSONL to stderr on all agent events (§8.5.1).
    - Writes a context provenance file (`/tmp/gh-aw/context-provenance.jsonl`) on `agent_end` recording the source and token cost of every context entry (§8.5.2).
    - Appends a Markdown execution summary table (per-turn tokens + context provenance) to `$GITHUB_STEP_SUMMARY` when that env var is set (§8.5.3).
    - Emits a human-readable per-turn token consumption line to stderr after each `turn_end` (§8.5.4).
    - Generates OTel spans using `observability.otlp` config.

11. **Write tests** — Unit tests for loader, each extension (mock `ExtensionAPI`). Integration tests with `createAgentSession()` + `SessionManager.inMemory()`.

12. **Write example workflows** — Single-task examples demonstrating `engine: aw` with various tools.

13. **Add build to Makefile** — Add `make aw-harness` target that runs esbuild and copies `aw_harness.cjs` to `actions/setup/js/`.

---

## 11. Security Considerations

### 11.1 General Security Requirements

**Mandatory proxy features.** The `gh-proxy` and `cli-proxy` features **MUST** always be active for `engine: aw`. MCP tools are available to agent sessions as CLI commands via `cli-proxy`; disabling it would make those tools inaccessible. Any attempt by a workflow author to disable either feature **MUST** be silently overridden (see [Section 6.2](#62-overrides-and-fixed-settings)).

**No direct LLM routing by harness.** The harness delegates all LLM routing to Pi SDK and the provider credentials injected by AWF. It **MUST NOT** perform additional proxy interception or credential manipulation.

**User extension isolation.** Extensions declared via `harness.extensions` run inside the same Node.js process as the built-in extensions. A conforming implementation **MUST NOT** execute user extensions with elevated privileges. Extension authors are responsible for ensuring that their extensions do not exfiltrate credentials or subvert built-in budget behavior. Workflow authors are responsible for auditing third-party npm extension packages before referencing them in `harness.extensions`.

**Budget enforcement.** The cost-tracker extension provides a hard budget gate. A conforming implementation **MUST** abort the session if the effective token count exceeds `harness.budget.max-effective-tokens`, preventing runaway usage from misbehaving agents.

**Token and secret handling.** Provider credentials (e.g., `ANTHROPIC_API_KEY`, `OPENAI_API_KEY`, `GITHUB_TOKEN`) **MUST NOT** be logged to stderr or embedded in JSONL events. Implementations **MUST** treat all credential env vars as opaque secrets.

### 11.2 Safeguards

This section specifies normative failure-mode responses that a conforming implementation **MUST** provide. Each safeguard defines a failure mode and its required normative response.

#### 11.2.1 Pi SDK Failure to Load

**Failure mode:** The Pi SDK package (`@mariozechner/pi-coding-agent`) or one of its core dependencies (`pi-agent-core`, `pi-ai`) cannot be loaded at harness startup (e.g., missing from bundle, corrupted installation, incompatible Node.js version).

**Normative response:**

- The harness **MUST** catch the load error and emit a structured JSONL error event to stderr indicating the SDK load failure and the originating error message.
- The harness **MUST** write a human-readable error summary to `$GITHUB_STEP_SUMMARY` (if set) that identifies the failed module and suggests reinstalling or rebuilding the bundle.
- The harness **MUST** exit with code `2` (invocation error) rather than code `1` (session failure), to distinguish SDK infrastructure failures from session-level failures.
- The harness **MUST NOT** attempt to proceed with a partial or degraded session; no `AgentSession` **MUST** be created if the SDK cannot be loaded.

#### 11.2.2 Budget Exhaustion

**Failure mode:** The cumulative effective token count across all turns exceeds `harness.budget.max-effective-tokens` during an active session.

**Normative response:**

- When effective tokens reach the **soft limit** (default: 80% of `max-effective-tokens`), the cost-tracker extension **MUST** inject a steering message via `session.steer()` informing the agent that it is approaching the token budget and **SHOULD** conclude its work soon.
- When effective tokens reach the **hard limit** (`max-effective-tokens`), the cost-tracker extension **MUST** abort the session immediately by invoking the session's abort API. The harness **MUST NOT** allow additional turns to proceed after the hard limit is reached.
- Upon hard-limit abort, the harness **MUST** emit a `budget_exceeded` JSONL event to stderr containing the final cumulative token count and the configured limit.
- The harness **MUST** write a step summary entry to `$GITHUB_STEP_SUMMARY` (if set) indicating that the session was terminated due to budget exhaustion, showing the final token count versus the limit.
- On forced budget termination, the harness **MUST** preserve durable artifacts that were finalized before abort (`safe-outputs.ndjson` entries already appended, JSONL events already emitted, and step-summary rows for completed turns).
- On forced budget termination, the harness **MUST** discard in-flight turn state that did not reach a completed turn boundary (partial assistant output, partially collected tool results, and uncommitted per-turn aggregates).
- A "completed turn boundary" means the `turn_end` event has been emitted and all per-turn persistence for that turn (JSONL line, counters, and step-summary row) has succeeded.
- The `budget_exceeded` event **MUST** explicitly signal forced termination (`reason: "hard_limit"` and `forced_termination: true`) so downstream consumers can distinguish budget aborts from other session failures.
- The harness **MUST** exit with code `1` (session failure) after a hard-limit abort, so that the GitHub Actions job is marked as failed.

#### 11.2.3 Extension Crash Isolation

**Failure mode:** A user-supplied Pi extension (declared via `harness.extensions`) throws an uncaught exception or returns a rejected Promise during its initialization function, or throws during event handler execution.

**Normative response:**

- **During initialization:** If an extension's default export function throws or rejects, the harness **MUST** catch the error, emit a warning to stderr identifying the failing extension by name/path and the error message, and continue loading the remaining extensions. The failing extension **MUST** be skipped and **MUST NOT** be registered into the session. If `harness.extensions-required: true` is set, the harness **MUST** instead abort startup with exit code `2` and a descriptive error message.
- **During event handling:** If an extension's event handler (registered via `pi.on()`) throws or rejects, the Pi SDK event dispatch **MUST** catch the error. If the Pi SDK does not isolate handler errors, the harness **MUST** wrap all user extension event handlers in a try/catch that emits a structured JSONL warning and allows the session to continue.
- **Built-in extensions are never skipped:** The five built-in gh-aw extensions (provider setup, cost-tracker, steering, repair, observability) **MUST NOT** be subject to the skip-on-error policy described above. If a built-in extension fails to load, the harness **MUST** treat it as a fatal startup error and exit with code `2`.
- The harness **MUST NOT** allow a crashing user extension to terminate the entire harness process without first completing the cleanup described above (step summary, final JSONL event).

### 11.3 MUST/MUST NOT Traceability (Spec ↔ Harness Source)

The following matrix records where each normative harness requirement is enforced in source. Until `actions/setup/js/aw_harness.cjs` is present in-repo, rows remain pending and serve as implementation obligations.

| Requirement anchor | Normative statement (summary) | Expected source assertion/guard | Status |
|--------------------|-------------------------------|----------------------------------|--------|
| §5.1 | MUST consume compiler-generated `config.json`/`prompt.txt`; MUST NOT parse markdown directly | `actions/setup/js/aw_harness.cjs` argument parser + loader guard | Pending (`aw_harness.cjs` not present) |
| §5.3 | MUST return exit code `0` only on clean completion; non-zero on unrecovered failure | `actions/setup/js/aw_harness.cjs` top-level process exit mapping | Pending (`aw_harness.cjs` not present) |
| §5.4 | MUST write diagnostics to stderr and summary to `$GITHUB_STEP_SUMMARY` | `actions/setup/js/aw_harness.cjs` output routing and summary writer | Pending (`aw_harness.cjs` not present) |
| §6.2 | MUST force-enable `gh-proxy` and `cli-proxy`; MUST NOT allow disabling | `actions/setup/js/aw_harness.cjs` config normalization guard | Pending (`aw_harness.cjs` not present) |
| §11.2.1 | MUST fail fast with exit `2` when Pi SDK cannot load; MUST NOT create partial session | `actions/setup/js/aw_harness.cjs` SDK import try/catch guard | Pending (`aw_harness.cjs` not present) |
| §11.2.2 | MUST hard-abort on budget limit; MUST NOT continue turns after hard limit | `actions/setup/js/aw_harness.cjs` cost-tracker abort gate and post-abort turn guard | Pending (`aw_harness.cjs` not present) |
| §11.2.3 | MUST isolate crashing user extensions; built-in extension failures are fatal | `actions/setup/js/aw_harness.cjs` extension loader policy checks | Pending (`aw_harness.cjs` not present) |

### 11.4 Degraded Mode & Safeguards

This section defines normative safeguard requirements for scenarios where the harness enters a degraded operating mode due to resource exhaustion, infrastructure unavailability, or partial subsystem failure. A conforming implementation **MUST** apply all safeguards numbered below.

1. **Budget-exhaustion shutdown path**: When the effective token budget is exhausted (hard limit reached), the harness **MUST** execute an orderly shutdown sequence: (a) immediately abort the active `AgentSession` turn via the session abort API; (b) flush all in-progress JSONL events and the step-summary buffer to their respective sinks; (c) emit a `budget_exceeded` event with `forced_termination: true` and the final cumulative token count; and (d) exit with code `1`. The harness **MUST NOT** start a new turn or accept additional tool calls after the hard-limit threshold is crossed, even if the session's internal queue contains pending callbacks.

2. **Partial observability failure behavior**: When the OTLP exporter or the context-provenance file writer fails (e.g., network unreachable, disk full, OTLP endpoint returns a non-retryable error), the harness **MUST** continue session execution and **MUST NOT** abort the session or exit with a non-zero code solely due to the observability failure. The harness **SHOULD** emit a structured JSONL warning event to stderr identifying the failed observability sink and the error reason. Observability subsystem failures **MUST** be treated as non-fatal degraded-mode conditions; data loss in telemetry **MUST NOT** propagate as a session-level failure.

3. **Fail-secure exit codes**: The harness **MUST** use the following exit-code contract to ensure downstream consumers can unambiguously detect failure class: exit code `0` — clean session completion with no budget abort and no fatal errors; exit code `1` — session-level failure, including hard-limit budget abort, unrecovered agent error, or failed session finalization; exit code `2` — invocation or infrastructure failure, including Pi SDK load failure, missing required configuration, or fatal built-in extension failure. The harness **MUST NOT** mask an exit code `1` or `2` condition by exiting `0`, even if the step summary was written successfully.

4. **Degraded-mode marking**: When the harness enters any degraded mode (observability failure, extension skip, or partial artifact flush), it **MUST** annotate the step summary (if `$GITHUB_STEP_SUMMARY` is set) with a visible degraded-mode notice that identifies which subsystem is degraded and what data may be incomplete. This notice **SHOULD** include a remediation hint (e.g., "check OTLP endpoint connectivity" or "extension X was skipped due to error Y").

---

## 12. Compliance Tests

This section specifies normative test cases for the AW Harness. A conforming implementation **MUST** provide an automated test suite that passes all tests marked **MUST** below. Tests marked **SHOULD** are strongly recommended.

Each test case specifies:
- **ID**: Stable test identifier.
- **Precondition**: The state required before the test stimulus is applied.
- **Stimulus**: The action taken to trigger the behavior under test.
- **Expected Result**: The observable outcome that a conforming implementation **MUST** produce.

---

### T-AW-001: Harness Invocation Contract

**ID**: `T-AW-001`  
**Precondition**: A valid `config.json` and `prompt.txt` are present at known paths. The Pi SDK is installed and loadable. AWF has injected at least one LLM provider credential into the container environment.  
**Stimulus**: Invoke `node aw_harness.cjs --config <config-path> --prompt <prompt-path>` with the correct paths.  
**Expected Result**: The harness starts without error, creates an `AgentSession`, runs the prompt to completion, and exits with code `0`. At least one JSONL `session_start` event and one `session_end` event are written to stderr. The step summary file (if `$GITHUB_STEP_SUMMARY` is set) contains a valid Markdown execution summary.

---

### T-AW-002: Extension Loading

**ID**: `T-AW-002`  
**Precondition**: `config.json` references one valid user extension at a local path that exports a default function, and one invalid extension path that does not exist. `harness.extensions-required` is `false` (default).  
**Stimulus**: Invoke the harness with this configuration.  
**Expected Result**: The harness loads the valid extension successfully. The missing extension causes a warning message on stderr (including the path and reason) but does **not** cause the harness to abort. The session proceeds with the valid extension registered and the missing extension skipped. Exit code is `0` (assuming the session completes successfully).

---

### T-AW-003: Budget Gate

**ID**: `T-AW-003`  
**Precondition**: `config.json` sets `harness.budget.max-effective-tokens` to a very low value (e.g., `1000` tokens). The Pi SDK is running in a mode where token counts can be simulated or observed. The agent session is active.  
**Stimulus**: Drive the session to consume effective tokens exceeding the configured hard limit.  
**Expected Result**: The cost-tracker extension aborts the session. A `budget_exceeded` JSONL event is emitted to stderr. The harness exits with code `1`. The step summary (if `$GITHUB_STEP_SUMMARY` is set) contains a budget exhaustion notice with the final token count and configured limit.

---

### T-AW-004: Model Resolution

**ID**: `T-AW-004`  
**Precondition**: `config.json` specifies a model alias (e.g., `"claude-sonnet-4.6"`) in the engine model field. The `ANTHROPIC_API_KEY` environment variable is set (or a stub is configured). The provider-setup extension is loaded.  
**Stimulus**: Start the harness and allow the `AgentSession` to be created.  
**Expected Result**: The provider-setup extension registers the Anthropic provider using the `ANTHROPIC_API_KEY` credential. The Pi SDK resolves the model alias to the corresponding Anthropic endpoint. No hard-coded provider URL or token appears in any JSONL event or the step summary. The session starts without a provider resolution error.

---

### T-AW-005: Session Termination

**ID**: `T-AW-005`  
**Precondition**: The harness is running a session. The Pi SDK session completes normally (agent returns a final response without error).  
**Stimulus**: The `AgentSession` reaches its natural end (agent sends a final message with no pending tool calls).  
**Expected Result**: The harness receives the `agent_end` event. The observability extension writes the context provenance file to `/tmp/gh-aw/context-provenance.jsonl`. A `session_end` JSONL event is emitted to stderr. The step summary (if `$GITHUB_STEP_SUMMARY` is set) contains a per-turn token table. The harness disposes the session and exits with code `0`. No dangling async operations remain after exit.

---

### T-AW-006: Pi SDK Failure to Load

**ID**: `T-AW-006`  
**Precondition**: The Pi SDK is unavailable (e.g., the bundle is intentionally corrupted or the require path is incorrect).  
**Stimulus**: Invoke the harness.  
**Expected Result**: The harness catches the load error, emits a structured JSONL error event identifying the missing module, writes a human-readable error to `$GITHUB_STEP_SUMMARY` (if set), and exits with code `2`. No `AgentSession` is created.

---

### T-AW-007: Extension Crash Isolation

**ID**: `T-AW-007`  
**Precondition**: `config.json` references a user extension that throws an exception during its initialization function. `harness.extensions-required` is `false`.  
**Stimulus**: Invoke the harness.  
**Expected Result**: The harness catches the extension initialization error, emits a warning to stderr identifying the failing extension and error message, skips the crashing extension, and continues loading the remaining extensions. The session proceeds without the crashing extension. The harness does **not** exit with a non-zero code due to the extension crash alone.

---

## 13. Privacy Considerations

*(This section is non-normative.)*

**Data residency.** All agent execution occurs within the gh-aw Actions container. No workflow content, prompts, or session data leave the container except via the Pi SDK to the configured LLM provider endpoint, or via OTLP to the configured telemetry endpoint.

**Telemetry scope.** When `observability.otlp` is configured, OTel spans contain model names, token counts, and cost data. They **SHOULD NOT** contain raw prompt or response text. Implementations **SHOULD** redact sensitive content from span attributes.

**Context provenance file.** The context provenance file (`/tmp/gh-aw/context-provenance.jsonl`) records the source path and token count of every context entry added to the session. It **MUST NOT** include raw prompt or response text; only metadata (source type, path, token counts) is recorded. Workflow authors **SHOULD** evaluate the sensitivity of file paths before enabling downstream analysis tools that read this file.

**Model provider data handling.** Prompt content is transmitted to the LLM provider using the credentials AWF injects into the container. Workflow authors are responsible for ensuring that content transmitted to LLM providers complies with applicable data handling policies.

---

## 14. References

### 14.1 Normative References

**[RFC 2119]**
Bradner, S., "Key words for use in RFCs to Indicate Requirement Levels", BCP 14, RFC 2119, March 1997. <https://www.rfc-editor.org/rfc/rfc2119>

### 14.2 Informative References

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

---

## Sync Notes

This section maps normative spec sections to their primary implementation files and directories in the `github/gh-aw` repository. Maintainers **SHOULD** keep this table updated whenever implementation files are added, renamed, or removed.

| Spec section | Implementation file / directory | Notes |
|---|---|---|
| §5 Harness Invocation Contract; §6 Workflow Definition; §7 Single-Session Execution Model | `actions/setup/js/aw_harness.cjs` | Primary harness entry point. All session lifecycle, config loading, and prompt execution logic lives here. Pending creation (see §11.3). |
| §8 Extensions (provider-setup, cost-tracker, steering, repair, observability) | `actions/setup/js/aw_harness.cjs` (inline extension registrations) | Built-in Pi extensions are implemented as inline factory functions exported from or co-located with the harness. When extracted, each extension SHOULD move to a sibling file named `aw_ext_{name}.cjs`. |
| §10 Build and Deployment; §10.1 esbuild configuration | `actions/setup/js/` (directory); `package.json` build scripts in `github/gh-aw` | JavaScript build toolchain. The harness is compiled with esbuild; build configuration and bundle output paths are tracked here. |
| §9 Model Resolution; §11.1 General Security Requirements (token/credential handling) | `pkg/workflow/` (Go compiler — `aw_engine.go` or equivalent) | The `engine: aw` compilation path in Go generates the `config.json` that specifies the model, provider credentials, and feature flags consumed by the harness at runtime. |
| §11.2 Safeguards; §11.4 Degraded Mode & Safeguards | `actions/setup/js/aw_harness.cjs` | Budget-gating, observability-failure recovery, and fail-secure exit-code enforcement are all implemented inside the harness. |
| §12 Compliance Tests (T-AW-001 through T-AW-007) | `pkg/cli/workflows/` (integration test workflows); `actions/setup/js/*.test.cjs` (unit tests) | Harness lifecycle integration tests live in `pkg/cli/workflows/`. Unit-level tests for harness helpers reside alongside the JavaScript source. |
