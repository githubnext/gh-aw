---
title: Glossary
description: Definitions of technical terms and concepts used throughout GitHub Agentic Workflows documentation.
sidebar:
  order: 1000
---

This glossary provides definitions for key technical terms and concepts used in GitHub Agentic Workflows.

> [!TIP]
> New to GitHub Agentic Workflows?
> Technical terms throughout the documentation link to their definitions here. Click any glossary link to understand unfamiliar concepts. Bookmark this page for quick reference!

## Core Concepts

### Agentic

Having agency - the ability to act independently, make context-aware decisions, and adapt behavior based on circumstances. Agentic workflows use AI to understand context and choose appropriate actions, contrasting with deterministic workflows that execute fixed sequences. From "agent" + "-ic" (having the characteristics of).

### Agentic Workflow

An AI-powered workflow that reasons, makes decisions, and takes autonomous actions using natural language instructions. Written in markdown instead of complex YAML, agentic workflows interpret context and adapt behavior flexibly. For example, instead of "if issue has label X, do Y", you write "analyze this issue and provide helpful context", and the AI decides what's helpful based on the specific issue content.

### Orchestration

Workflows that coordinate one or more worker workflows toward a shared goal. An orchestrator decides what work to do next and dispatches workers, while workers execute concrete tasks with scoped tools and limits. See the [Orchestration guide](/gh-aw/patterns/orchestration/).

### Orchestrator Workflow

A workflow that fans out work by dispatching other workflows (workers), aggregates results, and optionally posts summaries.

### Worker Workflow

A workflow dispatched by an orchestrator that performs a focused unit of work (triage, analysis, code changes, validation).

### Agentic Engine or Coding Agent

The AI system (typically GitHub Copilot CLI) that executes natural language instructions in an agentic workflow. The agent interprets tasks, uses available tools (GitHub API, file system, web search), and generates outputs based on context autonomously.

### Frontmatter

Configuration section at the top of a workflow file, enclosed between `---` markers. Contains YAML settings controlling when the workflow runs, permissions, and available tools, separating technical configuration from natural language instructions.

### Compilation

Translating Markdown workflows (`.md` files) into GitHub Actions YAML format (`.lock.yml` files), including validation, import resolution, tool configuration, and security hardening.

### Workflow Lock File (.lock.yml)

The compiled GitHub Actions workflow file from a workflow markdown file (`.md`). Contains complete GitHub Actions YAML with security hardening applied. Both `.md` and `.lock.yml` files should be committed to version control. At runtime, GitHub Actions executes the lock file using a coding agent while referencing the markdown for instructions.

## Tools and Integration

### MCP (Model Context Protocol)

A standardized protocol that allows AI agents to securely connect to external tools, databases, and services. MCP enables workflows to integrate with GitHub APIs, web services, file systems, and custom integrations while maintaining security controls.

### MCP Gateway

A transparent proxy service that enables unified HTTP access to multiple MCP servers using different transport mechanisms (stdio, HTTP). Provides protocol translation, server isolation, authentication, and health monitoring, allowing clients to interact with multiple backends through a single HTTP endpoint.

### MCP Server

A service that implements the Model Context Protocol to provide specific capabilities to AI agents. Examples include the GitHub MCP server (for GitHub API operations), Playwright MCP server (for browser automation), or custom MCP servers for specialized tools. See [Playwright Reference](/gh-aw/reference/playwright/) for browser automation configuration.

### Tools

Capabilities that an AI agent can use during workflow execution. Tools are configured in the frontmatter and include GitHub operations ([`github:`](/gh-aw/reference/github-tools/)), file editing (`edit:`), web access (`web-fetch:`, `web-search:`), shell commands (`bash:`), browser automation ([`playwright:`](/gh-aw/reference/playwright/)), and custom MCP servers.

## Security and Outputs

### Safe Inputs

Custom MCP tools defined inline in workflow frontmatter using JavaScript or shell scripts. Enables lightweight tool creation without external dependencies while maintaining controlled secret access. Tools are generated at runtime and mounted as an MCP server with typed input parameters, default values, and environment variables. Configured via `safe-inputs:` section.

### SARIF

Static Analysis Results Interchange Format - a standardized JSON format for reporting results from static analysis tools. Used by GitHub Code Scanning to display security vulnerabilities and code quality issues. Workflows can generate SARIF files using the `create-code-scanning-alert` safe output.

### SBOM

Software Bill of Materials - a comprehensive inventory of all components, libraries, and dependencies in a software project. Used for security auditing, vulnerability tracking, and compliance. Common formats include SPDX and CycloneDX.

### Safe Outputs

Pre-approved actions the AI can take without elevated permissions. The AI generates structured output describing what to create (issues, comments, pull requests), processed by separate permission-controlled jobs. Configured via `safe-outputs:` section, letting AI agents create GitHub content without direct write access.

### Threat Detection

Automated security analysis that scans agent output and code changes for potential security issues before application. When safe outputs are configured, a threat detection job automatically runs between the agent job and safe output processing to identify prompt injection attempts, secret leaks, and malicious code patches. See [Threat Detection Reference](/gh-aw/reference/threat-detection/).

### Secrecy

An optional field on safe output tool calls indicating the confidentiality level of the message content. Accepted values include `"public"`, `"internal"`, and `"private"`. Used alongside [`integrity`](#integrity) as security metadata displayed in safe output step summaries and available to [Threat Detection](#threat-detection) scanning.

### Staged Mode

A preview mode where workflows simulate actions without making changes. The AI generates output showing what would happen, but no GitHub API write operations are performed. Use for testing before production runs.

### Integrity

An optional field on safe output tool calls indicating the trustworthiness level of the message source. Accepted values include `"low"`, `"medium"`, and `"high"`. Used alongside [`secrecy`](#secrecy) as security metadata displayed in safe output step summaries and available to [Threat Detection](#threat-detection) scanning.

### Lockdown Mode

A security feature of the GitHub MCP server that filters content in public repositories to only surface items (issues, pull requests, comments, discussions) from users with push access. Protects agentic workflows from processing potentially malicious or misleading content submitted by untrusted users. Enabled via `lockdown: true` in the `tools.github` section. See [Lockdown Mode](/gh-aw/reference/lockdown-mode/).

### Status Comment

A comment posted on the triggering issue or pull request that shows workflow run status (started and completed). Configured via `status-comment: true` in `safe-outputs`. Must be explicitly enabled — it is not automatically bundled with `ai-reaction`.

### XPIA (Cross-Prompt Injection Attack)

A security attack where malicious instructions embedded in external data (issue bodies, PR descriptions, file contents) attempt to hijack an AI agent's behavior. GitHub Agentic Workflows mitigates XPIA through system prompt hardening and [Threat Detection](#threat-detection) scanning. See also [Lockdown Mode](#lockdown-mode).

### Permissions

Access controls defining workflow operations. Workflows follow least privilege, starting with read-only access by default. Write operations are typically handled through safe outputs.

### Safe Output Messages

Customizable messages workflows can display during execution. Configured in `safe-outputs.messages` with types `run-started`, `run-success`, `run-failure`, and `footer`. Supports GitHub context variables like `{workflow_name}` and `{run_url}`.

### Upload Assets

A safe output capability for uploading generated files (screenshots, charts, reports) to an orphaned git branch for persistent storage. The AI calls the `upload_asset` tool to register files, which are committed to a dedicated assets branch by a separate permission-controlled job. Assets are accessible via GitHub raw URLs. Commonly used for visual testing artifacts, data visualizations, and generated documentation.

### Base Branch

Configuration field in the `create-pull-request` safe output specifying which branch the pull request should target. Defaults to `github.base_ref || github.ref_name` if not specified. Useful for cross-repository pull requests targeting non-default branches.

### Minimize Comment

A safe output capability for hiding or minimizing GitHub comments without requiring write permissions. When minimized, comments are classified as SPAM. Requires GraphQL node IDs to identify comments. Useful for content moderation workflows.

### Assign to Agent

A safe output capability (`assign-to-agent:`) that programmatically assigns the GitHub Copilot coding agent to existing issues or pull requests. Automates the standard GitHub workflow for delegating implementation tasks to Copilot. Supports cross-repository PR creation via `pull-request-repo` and agent model selection via `model`. See [Assign to Copilot](/gh-aw/reference/assign-to-copilot/).

### GH_AW_AGENT_TOKEN

A recognized "magic" repository secret name that GitHub Agentic Workflows automatically uses as a fallback Personal Access Token for `assign-to-agent` operations. When set, no explicit `github-token:` reference is needed in workflow frontmatter — the token is injected automatically. Required because GitHub App installation tokens are rejected by the Copilot assignment API. The token fallback chain is: `assign-to-agent.github-token` → `safe-outputs.github-token` → `GH_AW_AGENT_TOKEN` → `GH_AW_GITHUB_TOKEN` → `GITHUB_TOKEN`. See [Assign to Copilot](/gh-aw/reference/assign-to-copilot/).

### Custom Safe Outputs

An extension mechanism for safe outputs that enables integration with third-party services beyond built-in GitHub operations. Defined under `safe-outputs.jobs:`, custom safe outputs separate read and write operations: agents use read-only MCP tools for queries, while custom jobs execute write operations with secret access after agent completion. Supports services like Slack, Notion, Jira, or any external API. See [Custom Safe Outputs](/gh-aw/reference/custom-safe-outputs/).

### Unassign from User

A safe output capability for removing user assignments from issues or pull requests. Supports an `allowed` list to restrict which users can be unassigned, and a `blocked` list using glob patterns to prevent unassignment of specific users regardless of the allow list. Configured via `unassign-from-user:` in `safe-outputs`.

### Temporary ID

A workflow-scoped identifier (format: `aw_` followed by 3–8 alphanumeric characters, e.g. `aw_abc1`) that lets an AI agent reference a resource before it is created. Safe output tools that support temporary IDs — including `create_issue`, `create_discussion`, and `add_comment` — accept a `temporary_id` field. References like `#aw_abc1` in subsequent operations are automatically resolved to actual resource numbers during execution. Useful for creating interlinked resources in a single workflow run. See [Safe Outputs Reference](/gh-aw/reference/safe-outputs/).

### Update Issue

A safe output capability (`update-issue:`) for modifying existing issues without creating new ones. Each updatable field (`status`, `title`, `body`) must be explicitly enabled. Body updates accept an `operation` field: `append` (default), `prepend`, `replace`, or `replace-island` (updates a specific section delimited by HTML comments). Supports cross-repository issue updates. See [Safe Outputs Reference](/gh-aw/reference/safe-outputs/#issue-updates-update-issue).

## Workflow Components

### Activation Token (`on.github-token:`, `on.github-app:`)

Custom GitHub token or GitHub App used by the activation job to post reactions and status comments on the triggering item. Configured via `github-token:` (for a PAT or token expression) or `github-app:` (to mint a short-lived installation token) inside the `on:` section. Affects only the activation job — agent job tokens are configured separately via `tools.github.github-token` or `safe-outputs.github-app`. See [Authentication Reference](/gh-aw/reference/auth/).

### Cron Schedule

A time-based trigger format. Use short syntax like `daily` or `weekly on monday` (recommended with automatic time scattering) or standard cron expressions for fixed times. See also Fuzzy Scheduling and Time Scattering.

### Engine

The AI system that powers the agentic workflow - essentially "which AI to use" to execute workflow instructions. GitHub Agentic Workflows supports multiple engines, with GitHub Copilot as the default.

### Fuzzy Scheduling

Natural language schedule syntax that automatically distributes workflow execution times to avoid load spikes. Instead of specifying exact times with cron expressions, fuzzy schedules like `daily`, `weekly`, or `daily on weekdays` are converted by the compiler into deterministic but scattered cron expressions. The compiler automatically adds `workflow_dispatch:` trigger for manual runs. Example: `schedule: daily on weekdays` compiles to something like `43 5 * * 1-5` with varied execution times across different workflows.

### Imports

Reusable workflow components shared across multiple workflows. Specified in the `imports:` field, can include tool configurations, common instructions, or security guidelines.

### Label Trigger Shorthand

A compact syntax for label-based triggers: `on: issue labeled bug` or `on: pull_request labeled needs-review`. The compiler expands the shorthand to standard GitHub Actions trigger syntax and automatically includes a `workflow_dispatch` trigger with an `inputs.item_number` parameter, enabling manual dispatch for a specific issue or pull request. Supported for `issue`, `pull_request`, and `discussion` events. See [LabelOps patterns](/gh-aw/patterns/label-ops/).

### Labels

Optional workflow metadata for categorization and organization. Enables filtering workflows in the CLI using the `--label` flag.

### Network Permissions

Controls over external domains and services a workflow can access. Configured via `network:` section with options: `defaults` (common infrastructure), custom allow-lists, or `{}` (no access).

### Stop After

A workflow configuration field (`stop-after:`) that automatically prevents new runs after a specified time limit. Accepts absolute dates (`YYYY-MM-DD`, ISO 8601) or relative time deltas (`+48h`, `+7d`). Minimum granularity is hours. Useful for trial periods, experimental features, and cost-controlled schedules. Recompile with `gh aw compile --refresh-stop-time` to reset the deadline. See [Ephemerals](/gh-aw/guides/ephemerals/).

### Triggers

Events that cause a workflow to run, defined in the `on:` section of frontmatter. Includes issue events, pull requests, schedules, manual runs, and slash commands.

### Trigger File

A plain GitHub Actions workflow (`.yml`) that separates trigger definitions from agentic workflow logic. Calls a compiled orchestrator's `workflow_call` entry point in response to any GitHub event (issues, pushes, labels, manual dispatch). Decouples trigger changes from the compilation cycle — updating when an orchestrator runs requires editing only the trigger file, not recompiling the agentic workflow. See [CentralRepoOps](/gh-aw/patterns/central-repo-ops/).

### Weekday Schedules

Scheduled workflows configured to run only Monday through Friday using `daily on weekdays` syntax. Recommended for daily workflows to avoid the "Monday wall of work" where tasks accumulate over weekends and create a backlog on Monday morning. The compiler converts this to cron expressions with `1-5` in the day-of-week field. Example: `schedule: daily on weekdays` generates a cron like `43 5 * * 1-5`.

### workflow_call

A trigger enabling a compiled workflow to be invoked by another workflow in the same organization. Adding `workflow_call` to the `on:` section exposes the lock file as a callable workflow, with optional inputs callers can pass for context. Commonly used with a [Trigger File](#trigger-file) to decouple trigger definitions from agentic workflow compilation. See [CentralRepoOps](/gh-aw/patterns/central-repo-ops/).

### workflow_dispatch

A manual trigger that runs a workflow on demand from the GitHub Actions UI or via the GitHub API. Requires explicit user initiation.

## GitHub and Infrastructure Terms

### GitHub Actions

GitHub's built-in automation platform that runs workflows in response to repository events. Agentic workflows compile to GitHub Actions YAML format, leveraging existing infrastructure for execution, permissions, and secrets.

### GitHub Projects (Projects v2)

GitHub's project management and tracking system organizing issues and pull requests using customizable boards, tables, and roadmaps. Provides flexible custom fields, automation, and GraphQL API access. Agentic workflows can manage project boards using the `update-project` safe output. Requires organization-level Projects permissions.

### GitHub Actions Secret

A secure, encrypted variable stored in repository or organization settings holding sensitive values like API keys or tokens. Access via `${{ secrets.SECRET_NAME }}` syntax.

### GitHub App (`github-app:`)

A GitHub App installation used for authentication and token minting in workflows. The `github-app:` field (which replaces the deprecated `app:` key) accepts `app-id` and `private-key` to mint short-lived installation access tokens with fine-grained, automatically-revoked permissions. Can be configured in `safe-outputs:` to override the default `GITHUB_TOKEN` for all safe output operations, or in `checkout:` for accessing private repositories. See [Authentication Reference](/gh-aw/reference/auth/#using-a-github-app-for-authentication).

### YAML

A human-friendly data format for configuration files using indentation and simple syntax to represent structured data. In agentic workflows, YAML appears in frontmatter and compiled `.lock.yml` files.

### Personal Access Token (PAT)

A token authenticating you to GitHub's APIs with specific permissions. Required for GitHub Copilot CLI to access Copilot services. Created at github.com/settings/personal-access-tokens.

### Agent Files

Markdown files with YAML frontmatter stored in `.github/agents/` defining interactive Copilot Chat agents. Created by `gh aw init`, these files can be invoked with the `/agent` command in Copilot Chat to guide workflow creation, debugging, and updates. The `agentic-workflows` agent is a unified dispatcher routing requests to specialized prompts.

### Fine-grained Personal Access Token

A GitHub Personal Access Token with granular permission control, specifying exactly which repositories the token can access and what permissions it has. Created at github.com/settings/personal-access-tokens.

## Development and Compilation

### CLI (Command Line Interface)

The `gh aw` extension for GitHub CLI providing commands for managing agentic workflows: compile, run, status, logs, add, and project management.

### Playground

An interactive web-based editor for authoring, compiling, and previewing agentic workflows without local installation. The Playground runs the gh-aw compiler in the browser using [WebAssembly](#webassembly-wasm) and auto-saves editor content to `localStorage` so work is preserved across sessions. Available at `/gh-aw/editor/`.

### Validation

Checking workflow files for errors, security issues, and best practices. Occurs during compilation and can be enhanced with strict mode and security scanners.

### WebAssembly (Wasm)

A compilation target allowing the gh-aw compiler to run in browser environments without server-side Go installation. The compiler is built as a `.wasm` module that packages markdown parsing, frontmatter extraction, import resolution, and YAML generation into a single file loaded with Go's `wasm_exec.js` runtime. Enables interactive playgrounds, editor integrations, and offline workflow compilation tools. See [WebAssembly Compilation](/gh-aw/reference/wasm-compilation/).

## Advanced Features

### AWF (Agent Workflow Firewall)

The default coding agent sandbox that isolates AI agent execution in a container with network egress control through domain-based access lists. AWF makes the host filesystem and environment variables available inside the container while restricting outbound network access to configured domains. Enabled with `sandbox.agent: awf` (the default when `sandbox` is not specified). See [Sandbox Configuration](/gh-aw/reference/sandbox/).

### Cache Memory

Persistent storage for workflows preserving data between runs. Configured via `cache-memory:` in tools section with 7-day retention in GitHub Actions cache. See [Cache Memory](/gh-aw/reference/cache-memory/).

### Command Triggers

Special triggers responding to slash commands in issue and PR comments. Configured using the `slash_command:` section with a command name.

### Conclusion Job

An automatically generated job in compiled workflows that handles post-agent reporting and cleanup. Receives a workflow-specific concurrency group (`gh-aw-conclusion-{workflow-name}`) to prevent collision when multiple agent instances run the same workflow concurrently. Requires no manual configuration — the compiler sets the group automatically. See [Concurrency Control](/gh-aw/reference/concurrency/).

### Concurrency Control

Settings limiting how many workflow instances can run simultaneously. Configured via `concurrency:` field to prevent resource conflicts or rate limiting.

### Custom Agents

Specialized instructions customizing AI agent behavior for specific tasks or repositories. Stored as agent files (`.github/agents/*.agent.md`) for Copilot Chat or instruction files (`.github/copilot/instructions/`) for path-specific Copilot instructions.

### Ephemerals

A category of features for automatically expiring workflow resources to reduce repository noise and control costs. Includes workflow stop-after scheduling, safe output expiration (auto-closing issues, discussions, and pull requests), and hidden older status comments. See [Ephemerals](/gh-aw/guides/ephemerals/).

### Environment Variables (env)

Configuration section in frontmatter defining environment variables for the workflow. Variables can reference GitHub context values, workflow inputs, or static values. Accessible via `${{ env.VARIABLE_NAME }}` syntax.

### Repo Memory

Persistent file storage via Git branches with unlimited retention. Unlike cache-memory (7-day retention), repo-memory stores files permanently in dedicated Git branches with automatic branch cloning, file access, commits, pushes, and merge conflict resolution. See [Repo Memory](/gh-aw/reference/repo-memory/).

### Sandbox

Configuration for the AI agent execution environment, providing two isolation layers: the **Coding Agent Sandbox** ([AWF](#awf-agent-workflow-firewall) by default) for network egress control, and the **MCP Gateway** for routing MCP server calls through a unified HTTP endpoint. Configured via the `sandbox:` field in frontmatter. To disable only the agent firewall while keeping the MCP Gateway active, use `sandbox.agent: false`. See [Sandbox Configuration](/gh-aw/reference/sandbox/).

### Strict Mode

Enhanced validation mode enforcing additional security checks and best practices. Enabled via `strict: true` in frontmatter or `--strict` flag when compiling.

### Time Scattering

Automatic distribution of workflow execution times across the day to reduce load spikes on GitHub Actions infrastructure. When using fuzzy scheduling, the compiler deterministically assigns different start times to each workflow based on repository and workflow name. Prevents all scheduled workflows from running simultaneously at common times like midnight or the top of the hour.

### Timeout

Maximum duration a workflow can run before automatic cancellation. Configured via `timeout-minutes:` in frontmatter. Default is 360 minutes (6 hours); workflows can specify shorter timeouts to fail faster.

### Toolsets

Predefined collections of related MCP tools enabled together. Used with the GitHub MCP server to group capabilities like `repos`, `issues`, and `pull_requests`. Configured in the `toolsets:` field.

### Tracker ID

A unique identifier enabling external monitoring and coordination without bidirectional coupling. Orchestrator workflows use tracker IDs to correlate worker runs and discover outputs while workers operate independently.

### Workflow Inputs

Parameters provided when manually triggering a workflow with `workflow_dispatch`. Defined in the `on.workflow_dispatch.inputs` section with type, description, default value, and required status.

## Operational Patterns

Operational patterns (suffixed with "-Ops") are established workflow architectures for common automation scenarios. Each pattern addresses specific use cases with recommended triggers, tools, and safe outputs.

### CentralRepoOps

A [MultiRepoOps](#multirepoops) deployment variant where a single private repository acts as a control plane for coordinating large-scale operations across many repositories. Enables consistent rollouts, policy updates, and centralized tracking using cross-repository safe outputs and secure authentication. See [CentralRepoOps](/gh-aw/patterns/central-repo-ops/).

### ChatOps

Interactive automation triggered by slash commands (`/review`, `/deploy`) in issues and pull requests, enabling human-in-the-loop automation where developers invoke AI assistance on demand. See [ChatOps](/gh-aw/patterns/chat-ops/).

### DailyOps

Scheduled workflows for incremental daily improvements, automating progress toward large goals through small, manageable changes on weekday schedules. See [DailyOps](/gh-aw/patterns/daily-ops/).

### DataOps

Hybrid pattern combining deterministic data extraction in `steps:` with agentic analysis in the workflow body. Shell commands fetch and structure data, then the AI agent interprets results and produces insights. See [DataOps](/gh-aw/patterns/data-ops/).

### DispatchOps

Manual workflow execution via GitHub Actions UI or CLI using `workflow_dispatch` trigger. Enables on-demand tasks, testing, and workflows requiring human judgment about timing. Workflows can accept custom input parameters. See [DispatchOps](/gh-aw/patterns/dispatch-ops/).

### IssueOps

Automated issue management that analyzes, categorizes, and responds to issues when created. Uses issue event triggers with safe outputs for secure automated triage without requiring write permissions for the AI job. See [IssueOps Examples](/gh-aw/patterns/issue-ops/).

### LabelOps

Workflows triggered by label changes on issues and pull requests. Uses labels as triggers, metadata, and state markers with filtering for specific label additions or removals. See [LabelOps Examples](/gh-aw/patterns/label-ops/).

### MemoryOps

Stateful workflows that persist data between runs using `cache-memory` and `repo-memory`, enabling progress tracking, resumption after interruptions, and incremental processing to avoid API throttling. See [MemoryOps](/gh-aw/guides/memoryops/).

### MultiRepoOps

Cross-repository coordination extending automation patterns across multiple repositories. Uses secure authentication and cross-repository safe outputs to synchronize features, centralize tracking, and enforce organization-wide policies. See [MultiRepoOps](/gh-aw/patterns/multi-repo-ops/).

### ProjectOps

AI-powered GitHub Projects board management automating issue triage, routing, and field updates. Analyzes issue/PR content to make intelligent decisions about project assignment, status, priority, and custom fields using the `update-project` safe output. See [ProjectOps](/gh-aw/patterns/project-ops/).

### SideRepoOps

Development pattern where workflows run from a separate "side" repository targeting your main codebase. Keeps AI-generated issues, comments, and workflow runs isolated from the main repository for cleaner separation between automation infrastructure and production code. See [SideRepoOps](/gh-aw/patterns/side-repo-ops/).

### SpecOps

Maintaining and propagating W3C-style specifications using the `w3c-specification-writer` agent. Creates formal specifications with RFC 2119 keywords and automatically synchronizes changes to consuming implementations. See [SpecOps](/gh-aw/patterns/spec-ops/).

### TaskOps

Scaffolded AI-powered code improvement strategy with three phases: research agent investigates, developer reviews and invokes planner agent to create actionable issues, then assigns approved issues to Copilot for automated implementation. Keeps developers in control with clear decision points. See [TaskOps](/gh-aw/patterns/task-ops/).

### TrialOps

Testing and validation pattern executing workflows in isolated trial repositories before production deployment. Creates temporary private repositories where workflows run safely, capturing safe outputs without modifying your actual codebase. See [TrialOps](/gh-aw/patterns/trial-ops/).

## Related Resources

For detailed documentation on specific topics, see:

- [Frontmatter Reference](/gh-aw/reference/frontmatter/)
- [Tools Reference](/gh-aw/reference/tools/)
- [Safe Inputs Reference](/gh-aw/reference/safe-inputs/)
- [Safe Outputs Reference](/gh-aw/reference/safe-outputs/)
- [Using MCPs Guide](/gh-aw/guides/mcps/)
- [Security Guide](/gh-aw/introduction/architecture/)
- [AI Engines Reference](/gh-aw/reference/engines/)
