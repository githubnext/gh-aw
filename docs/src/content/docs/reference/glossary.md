---
title: Glossary
description: Definitions of technical terms and concepts used throughout GitHub Agentic Workflows documentation.
sidebar:
  order: 1000
---

This glossary provides definitions for key technical terms and concepts used in GitHub Agentic Workflows.

## Core Concepts

### Agentic

Having agency - the ability to act independently, make context-aware decisions, and adapt behavior based on circumstances. Agentic workflows use AI to understand context and choose appropriate actions, contrasting with deterministic workflows that execute fixed sequences. From "agent" + "-ic" (having the characteristics of).

### Agentic Workflow

An AI-powered workflow that reasons, makes decisions, and takes autonomous actions using natural language instructions. Written in markdown instead of complex YAML, agentic workflows interpret context and adapt behavior flexibly. For example, instead of "if issue has label X, do Y", you write "analyze this issue and provide helpful context", and the AI decides what's helpful based on the specific issue content.

### Orchestration

Workflows that coordinate one or more worker workflows toward a shared goal. An orchestrator decides what work to do next and dispatches workers, while workers execute concrete tasks with scoped tools and limits. See the [OrchestratorOps pattern](/gh-aw/patterns/orchestrator-ops/).

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

### Trusted Bots (`sandbox.mcp.trusted-bots`)

A frontmatter field that passes additional GitHub bot identity strings to the [MCP Gateway](#mcp-gateway). The gateway merges these with its built-in trusted identity list to determine which bot identities are permitted. This field is additive — it can only extend the gateway's internal list, not remove built-in entries. Configured under `sandbox.mcp:` and compiled into the `trustedBots` array in the generated gateway configuration. Example entries: `github-actions[bot]`, `copilot-swe-agent[bot]`. See [MCP Gateway Reference](/gh-aw/reference/mcp-gateway/).

### MCP Server

A service that implements the Model Context Protocol to provide specific capabilities to AI agents. Examples include the GitHub MCP server (for GitHub API operations), Playwright MCP server (for browser automation), or custom MCP servers for specialized tools. See [Playwright Reference](/gh-aw/reference/playwright/) for browser automation configuration.

### QMD Documentation Search (`qmd:`)

A built-in tool that provides vector similarity search over documentation files. Configured via `tools.qmd:` in frontmatter, the `qmd` tool runs [tobi/qmd](https://github.com/tobi/qmd) as an MCP server so agents can find relevant documentation by natural language query. The search index is built in a dedicated indexing job (which has `contents: read`) and shared with the agent job via `actions/cache`, so the agent job does not need `contents: read`. Supports indexing from repository checkouts, GitHub code search queries, and cache-only read-only mode. See [QMD Documentation Search](/gh-aw/reference/qmd/).

### Tools

Capabilities that an AI agent can use during workflow execution. Tools are configured in the frontmatter and include GitHub operations ([`github:`](/gh-aw/reference/github-tools/)), file editing (`edit:`), web access (`web-fetch:`, `web-search:`), shell commands (`bash:`), browser automation ([`playwright:`](/gh-aw/reference/playwright/)), and custom MCP servers.

### GitHub Access Mode (`tools.github.mode`)

A `tools.github` field that controls how the agent accesses GitHub APIs. Three values are supported: `gh-proxy` (recommended — provides pre-authenticated `gh` CLI prompt guidance without mounting a GitHub MCP server, replacing the deprecated `features.cli-proxy: true`), `local` (Docker-based GitHub MCP server, the legacy default), and `remote` (hosted GitHub MCP server at `api.githubcopilot.com`). Use `gh-proxy` for better performance; use `local` or `remote` when MCP-based GitHub toolsets are required. See [GitHub Tools Reference](/gh-aw/reference/github-tools/).

### Allowed Repos (`tools.github.allowed-repos`)

A GitHub Tools configuration field that restricts which repositories the agent can access through GitHub tools during workflow execution. Accepts `"all"` (default — all repositories accessible by the token), `"public"` (public repositories only), `"current"` (the repository where the workflow is running, normalized to `${{ github.repository }}`), or an array of repository patterns (`"owner/repo"`, `"owner/*"`, `"owner/prefix*"`). Wildcards are only permitted at the end of the repository name component. Use `current` in reusable workflows to express "this repository only" without hard-coding owner/repo values. Patterns must be lowercase. See [GitHub Tools Reference](/gh-aw/reference/github-tools/).

```aw wrap
tools:
  github:
    toolsets: [issues, pull_requests]
    allowed-repos: current
    min-integrity: approved
```

### Tool Call Limits (`max-calls`)

A per-tool call cap declared inline on entries in `tools.github.allowed`. Each entry can be a plain string tool name (no limit) or an object with a `name` field and an optional `max-calls` integer. When `max-calls` is set, the MCP gateway enforces that the tool may be called at most that many times in a single workflow run, preventing runaway re-invocation of read tools. Call limits are emitted into the `allow-only.tool-call-limits` guard policy at compile time; enforcement is delegated to the AWF MCP gateway. See [GitHub Tools Reference](/gh-aw/reference/github-tools/).

```aw wrap
tools:
  github:
    allowed:
      - issue_read          # no limit
      - name: list_labels
        max-calls: 3
```

### Ignore If Missing (`ignore-if-missing`)

A GitHub App authentication field that gracefully skips token minting when `client-id` or `private-key` resolve to empty strings at runtime (e.g., on fork pull requests where App secrets are unavailable). When set to `true` under `github-app.ignore-if-missing`, the workflow falls back to the standard token chain (`secrets.GH_AW_GITHUB_TOKEN || secrets.GITHUB_TOKEN`) instead of failing. Applies consistently to all token mint paths: safe outputs, activation, pre-activation, and checkout. Default behavior (fail when keys are empty) remains unchanged when omitted or set to `false`. See [Authentication Reference](/gh-aw/reference/auth/).

```aw wrap
safe-outputs:
  github-app:
    client-id: ${{ vars.APP_ID }}
    private-key: ${{ secrets.APP_PRIVATE_KEY }}
    ignore-if-missing: true
  create-issue:
```

## Security and Outputs

### MCP Scripts

Custom MCP tools defined inline in workflow frontmatter using JavaScript or shell scripts. Enables lightweight tool creation without external dependencies while maintaining controlled secret access. Tools are generated at runtime and mounted as an MCP server with typed input parameters, default values, and environment variables. Configured via `mcp-scripts:` section.

### SARIF

Static Analysis Results Interchange Format - a standardized JSON format for reporting results from static analysis tools. Used by GitHub Code Scanning to display security vulnerabilities and code quality issues. Workflows can generate SARIF files using the `create-code-scanning-alert` safe output.

### Safe Outputs

Pre-approved actions the AI can take without elevated permissions. The AI generates structured output describing what to create (issues, comments, pull requests), processed by separate permission-controlled jobs. Configured via `safe-outputs:` section, letting AI agents create GitHub content without direct write access.

### Outcome

The observable repository state that follows a [safe output](#safe-outputs). While safe outputs record what a workflow did, outcomes record what happened afterward — whether a pull request was merged, an issue was resolved, or a comment received follow-up activity. Outcome data is based on visible repository state, not on the workflow's self-assessment. See [Outcomes](/gh-aw/reference/outcomes/).

### Outcome State

The classification assigned to each evaluated safe output result. The six outcome states are:

- `accepted` — kept, merged, completed, or otherwise accepted
- `rejected` — explicitly undone, closed, or not accepted
- `pending` — exists but has not reached a terminal state
- `ignored` — received no meaningful follow-up within the evaluation window
- `lifecycle` — closed or removed by the workflow itself as part of normal operation (for example, a stale-issue workflow), not a rejection
- `lifecycle_close` — a `close_issue` or `close_pull_request` output where the closing actor was a lifecycle bot and no non-bot actor has since reopened it

See [Outcomes](/gh-aw/reference/outcomes/).

### Accepted Outcome

The simplest useful unit for measuring workflow effectiveness. An outcome is accepted when its result was kept, merged, completed, or acted on — for example, a merged pull request, a resolved issue, or a comment that received a reaction or reply. Different output types use type-specific rules to determine acceptance. See [Outcomes](/gh-aw/reference/outcomes/).

### Outcome Efficiency

A cost-quality metric computed as AI Credits (AIC) divided by accepted outcomes. Lower values indicate the workflow consumed fewer AI Credits per accepted result. Outcome efficiency makes the difference between cost savings from genuine efficiency gains and cost savings from doing less useful work. See [Measuring Impact](/gh-aw/reference/measuring-impact/).

### Pwn Request

A critical security vulnerability that occurs when a `pull_request_target` workflow checks out and executes code from a fork PR. Because `pull_request_target` runs in the context of the target (base) branch with full write permissions and access to repository secrets, executing untrusted fork code grants an attacker the ability to exfiltrate secrets or make unauthorized changes. The compiler emits a warning (non-strict mode) or a hard error (strict mode) when `pull_request_target` is used without `checkout: false`. Add `checkout: false` to prevent the insecure checkout; use `pull_request` instead when you do not need write-back access. See the [GitHub Security Lab advisory on pwn requests](https://securitylab.github.com/resources/github-actions-preventing-pwn-requests/).

### Threat Detection

Automated security analysis that scans agent output and code changes for potential security issues before application. When safe outputs are configured, a threat detection job automatically runs between the agent job and safe output processing to identify prompt injection attempts, secret leaks, and malicious code patches. See [Threat Detection Reference](/gh-aw/reference/threat-detection/).

### Threat Detection Max AI Credits (`safe-outputs.threat-detection.max-ai-credits`)

A `safe-outputs.threat-detection` field that caps the total AI Credits (AIC) the AWF proxy will spend for a single threat-detection run. Defaults to `400` AIC when omitted. Accepts an integer, a `K`/`M` suffix string (e.g., `750`), or `-1` to disable budget steering for detection runs. The organization-wide default can be overridden at runtime via `vars.GH_AW_DEFAULT_DETECTION_MAX_AI_CREDITS` without recompiling. Precedence: frontmatter literal → `GH_AW_DEFAULT_DETECTION_MAX_AI_CREDITS` variable → built-in default of `400`. See [Compiler Enterprise Environment Controls](/gh-aw/reference/compiler-enterprise-environment-controls/).

```aw wrap
safe-outputs:
  create-pull-request:
  threat-detection:
    max-ai-credits: 750
```

### Staged Mode

A preview mode where workflows simulate actions without making changes. The AI generates output showing what would happen, but no GitHub API write operations are performed. Use for testing before production runs. See [Staged Mode](/gh-aw/reference/staged-mode/) for details.

### Integrity Filtering

A guardrail feature that controls which GitHub content an agent can access, filtering by author trust and merge status. Content below the configured `min-integrity` threshold is silently removed before the AI engine sees it. The four levels are `merged`, `approved`, `unapproved`, and `none` (most to least restrictive). For public repositories, `min-integrity: approved` is applied automatically — restricting content to owners, members, and collaborators — even without additional authentication. Set `min-integrity: none` to allow all content through for workflows designed to process untrusted input (e.g., triage bots).

Three additional fields extend integrity filtering beyond the level threshold: `trusted-users` elevates specific GitHub usernames to `approved` integrity regardless of their author association; `blocked-users` unconditionally denies content from listed usernames regardless of level; and `approval-labels` promotes items bearing any listed label to `approved` integrity, enabling human-review workflows. See [Integrity Filtering](/gh-aw/reference/integrity/).

### DIFC Proxy (`tools.github.integrity-proxy`)

Controls full Data Integrity and Flow Control (DIFC) proxy enforcement. When `tools.github.min-integrity` is configured, the compiler injects proxy steps around the agent job that enforce integrity-level isolation at the network boundary. The proxy is **enabled by default** — set `tools.github.integrity-proxy: false` to disable it and rely solely on MCP gateway-level filtering. Filtered content is recorded as `DIFC_FILTERED` events in `gateway.jsonl` for later inspection. See [Integrity Filtering](/gh-aw/reference/integrity/).

### Integrity Reactions (`features.integrity-reactions`)

A feature flag that enables GitHub reactions (👍, ❤️, 👎, 😕) to promote or demote content past the integrity filter. When `integrity-reactions: true` is set, trusted members can add a reaction to an issue or comment to elevate its integrity to `approved` (endorsement reactions) or demote it to `none` (disapproval reactions) — without modifying labels. Enabling this flag automatically activates `cli-proxy` mode, which is required to identify reaction authors at the network boundary. Available from gh-aw v0.68.2. See [Maintaining Repos](/gh-aw/examples/maintaining-repos/#reactions-as-trust-signals).

### Status Comment

A comment posted on the triggering issue or pull request that shows workflow run status (started and completed). Configured via `status-comment: true` in `safe-outputs`. Defaults to `true` for `slash_command` and `label_command` triggers; must be explicitly enabled for other trigger types. Set `status-comment: false` to disable. Not automatically bundled with `ai-reaction` — each must be configured independently.

### Permissions

Access controls defining workflow operations. Workflows follow least privilege, starting with read-only access by default. Write operations are typically handled through safe outputs.

### Safe Output Messages

Customizable messages workflows can display during execution. Configured in `safe-outputs.messages` with types `run-started`, `run-success`, `run-failure`, and `footer`. Supports GitHub context variables like `{workflow_name}` and `{run_url}`.

### Failure Issue Reporting (`report-failure-as-issue:`)

A `safe-outputs` option controlling whether workflow run failures are automatically reported as GitHub issues. Defaults to `true` when safe outputs are configured. Set to `false` to suppress failure issue creation for workflows where failures are expected or handled externally:

```yaml
safe-outputs:
  report-failure-as-issue: false
```

See [Safe Outputs Reference](/gh-aw/reference/safe-outputs/).

### Failure Issue Repository (`failure-issue-repo:`)

A `safe-outputs` option that redirects failure tracking issues to a different repository. Useful when the workflow's repository has issues disabled:

```yaml
safe-outputs:
  failure-issue-repo: github/docs-engineering
```

See [Safe Outputs Reference](/gh-aw/reference/safe-outputs/).

### Upload Assets

A safe output capability for uploading generated files (screenshots, charts, reports) to an orphaned git branch for persistent storage. The AI calls the `upload_asset` tool to register files, which are committed to a dedicated assets branch by a separate permission-controlled job. Assets are accessible via GitHub raw URLs. Commonly used for visual testing artifacts, data visualizations, and generated documentation.

### Base Branch

Configuration field in the `create-pull-request` safe output specifying which branch the pull request should target. Defaults to `github.base_ref || github.ref_name` if not specified. Useful for cross-repository pull requests targeting non-default branches.

### Minimize Comment

A safe output capability for hiding or minimizing GitHub comments without requiring write permissions. When minimized, comments are classified as SPAM. Requires GraphQL node IDs to identify comments. Useful for content moderation workflows.

### Hide Older Comments (`hide-older-comments`)

A field on `add-comment:` safe outputs that minimizes previous comments before posting a new one. Accepts a boolean (`true`) or an object with `enabled` and `match` keys. The boolean form hides earlier comments from the same workflow only (identified by `GITHUB_WORKFLOW`). The object form adds a `match` list of additional workflow IDs whose older comments are also minimized — using exact full-string matching — so a single run can clean up comments from multiple related workflows.

```aw wrap
safe-outputs:
  add-comment:
    hide-older-comments:
      enabled: true
      match:
        - my-other-workflow
```

See [Safe Outputs Reference](/gh-aw/reference/safe-outputs/#hide-older-comments).

### Add Labels (`add-labels:`)

A safe output capability for adding labels to issues or pull requests. Supports an `allowed` list to restrict which labels can be applied, and a `blocked` list using glob patterns to reject specific labels regardless of the allow list — providing protection against prompt injection via label manipulation. Accepts `target` (`"triggering"`, `"*"`, or a specific number), a `max` limit (default: 3), and cross-repository configuration via `target-repo`. See [Safe Outputs Reference](/gh-aw/reference/safe-outputs/#add-labels-add-labels).

### Remove Labels (`remove-labels:`)

A safe output capability for removing labels from issues or pull requests. Supports `allowed` to restrict which labels can be removed and `blocked` to prevent removal of labels matching glob patterns. Silently skips labels not present on the target. See [Safe Outputs Reference](/gh-aw/reference/safe-outputs/#remove-labels-remove-labels).

### Assign to Agent

A safe output capability (`assign-to-agent:`) that programmatically assigns the GitHub Copilot coding agent to existing issues or pull requests. Automates the standard GitHub workflow for delegating implementation tasks to Copilot. Supports cross-repository PR creation via `pull-request-repo` and agent model selection via `model`. See [Copilot Cloud Agent](/gh-aw/reference/copilot-cloud-agent/).

### GH_AW_AGENT_TOKEN

A recognized "magic" repository secret name that GitHub Agentic Workflows automatically uses as a fallback Personal Access Token for `assign-to-agent` operations. When set, no explicit `github-token:` reference is needed in workflow frontmatter — the token is injected automatically. Required because GitHub App installation tokens are rejected by the Copilot assignment API. The token fallback chain is: `assign-to-agent.github-token` → `safe-outputs.github-token` → `GH_AW_AGENT_TOKEN` → `GH_AW_GITHUB_TOKEN` → `GITHUB_TOKEN`. See [Copilot Cloud Agent](/gh-aw/reference/copilot-cloud-agent/).

### GH_AW_GITHUB_TOKEN

A recognized "magic" repository secret name used as the default fallback token for GitHub operations outside engine inference when no explicit `github-token:` is configured. Commonly used for safe outputs and tool operations that require permissions beyond the default `GITHUB_TOKEN`, and as the non-App fallback when `github-app.ignore-if-missing: true` is enabled. Fallback chain: `custom github-token` → `GH_AW_GITHUB_TOKEN` → `GITHUB_TOKEN`. See [Authentication Reference](/gh-aw/reference/auth/).

### Custom Safe Outputs

An extension mechanism for safe outputs that enables integration with third-party services beyond built-in GitHub operations. Defined under `safe-outputs.jobs:`, custom safe outputs separate read and write operations: agents use read-only MCP tools for queries, while custom jobs execute write operations with secret access after agent completion. Supports services like Slack, Notion, Jira, or any external API. See [Custom Safe Outputs](/gh-aw/reference/custom-safe-outputs/).

### Dispatch Repository (`dispatch_repository`)

An experimental safe output type that triggers `repository_dispatch` events in external repositories for cross-repository orchestration. Each key under `safe-outputs.dispatch_repository:` defines a named tool exposed to the agent. A tool requires a `workflow` identifier (forwarded in `client_payload` for routing), an `event_type`, and either a static `repository` slug or an `allowed_repositories` list. GitHub Actions expressions (`${{ ... }}`) are supported in repository fields and are passed through without format validation. At compile time the compiler emits a warning: `Using experimental feature: dispatch_repository`. See [Safe Outputs Reference](/gh-aw/reference/safe-outputs/#repository-dispatch-dispatch_repository).

### Safe Output Actions

A mechanism for mounting any public GitHub Action as a once-callable MCP tool within the consolidated safe-outputs job. Defined under `safe-outputs.actions:`, each action is specified with a `uses` field (matching GitHub Actions syntax) and an optional `description` override. At compile time, `gh aw compile` fetches the action's `action.yml` to resolve its inputs and pins the reference to a specific SHA. Unlike [Custom Safe Outputs](#custom-safe-outputs) (separate jobs) and [Safe Output Scripts](#safe-output-scripts) (inline JavaScript), actions run as steps inside the safe-outputs job with full secret access via `env:`. Useful for reusing existing marketplace actions as agent tools. See [Custom Safe Outputs](/gh-aw/reference/custom-safe-outputs/#github-action-wrappers-safe-outputsactions).

### Safe Output Scripts

Lightweight inline JavaScript handlers defined under `safe-outputs.scripts:` that execute inside the consolidated safe-outputs job handler loop. Unlike [Custom Safe Outputs](#custom-safe-outputs) (`safe-outputs.jobs`), which create a separate GitHub Actions job per tool call, scripts run in-process with no job scheduling overhead. Scripts do not have direct access to repository secrets, making them suitable for lightweight processing and logging. Each script declares `description`, `inputs`, and a `script` body; the compiler wraps the body and registers the handler as an MCP tool available to the agent. See [Custom Safe Outputs](/gh-aw/reference/custom-safe-outputs/#inline-script-handlers-safe-outputsscripts).

### Safe Outputs Dependencies (`safe-outputs.needs:`)

A `safe-outputs` option that extends the consolidated `safe_outputs` job dependencies with custom workflow jobs. `safe-outputs.needs` is merged with built-in dependencies (`agent`, `activation`, optional `detection`, optional `unlock`) and deduplicated. Useful for injecting credential-fetching or secret-provisioning jobs that the safe-outputs job depends on. Values must reference custom jobs from the top-level `jobs:` section; built-in job names are rejected at compile time with an actionable error. See [Safe Outputs Reference](/gh-aw/reference/safe-outputs/#safe-outputs-dependencies-needs).

### Unassign from User

A safe output capability for removing user assignments from issues or pull requests. Supports an `allowed` list to restrict which users can be unassigned, and a `blocked` list using glob patterns to prevent unassignment of specific users regardless of the allow list. Configured via `unassign-from-user:` in `safe-outputs`.

### Temporary ID

A workflow-scoped identifier (format: `aw_` followed by 3–8 alphanumeric characters, e.g. `aw_abc1`) that lets an AI agent reference a resource before it is created. Safe output tools that support temporary IDs — including `create_issue`, `create_discussion`, and `add_comment` — accept a `temporary_id` field. References like `#aw_abc1` in subsequent operations are automatically resolved to actual resource numbers during execution. Useful for creating interlinked resources in a single workflow run. See [Safe Outputs Reference](/gh-aw/reference/safe-outputs/).

### Merge Pull Request (`merge-pull-request:`)

An experimental safe output capability for merging pull requests after policy-driven gate checks pass. Validates status checks, required approvals, resolved review threads, label and branch constraints, and GitHub mergeability before applying the merge. Supports `merge`, `squash`, and `rebase` methods and cross-repository targets. Compiling a workflow with `merge-pull-request` emits an experimental feature warning. See [Safe Outputs Specification](/gh-aw/specs/safe-outputs-specification/#type-merge_pull_request).

### Close Pull Request (`close-pull-request:`)

A safe output capability for closing pull requests without merging, with an optional comment. Supports filtering via `required-labels` and `required-title-prefix` to prevent unintended closures. Accepts `target` to identify the PR (`"triggering"`, `"*"`, or a specific number), cross-repository configuration via `target-repo`, and a `max` limit on closures. See [Safe Outputs (Pull Requests)](/gh-aw/reference/safe-outputs-pull-requests/#close-pull-request-close-pull-request).

### Update Issue

A safe output capability (`update-issue:`) for modifying existing issues without creating new ones. Each updatable field (`status`, `title`, `body`) must be explicitly enabled. Body updates accept an `operation` field: `append` (default), `prepend`, `replace`, or `replace-island` (updates a specific section delimited by HTML comments). Supports cross-repository issue updates. See [Safe Outputs Reference](/gh-aw/reference/safe-outputs/#issue-updates-update-issue).

### Update Pull Request (`update-pull-request:`)

A safe output capability for modifying a pull request's `title` or `body`. Each field must be explicitly enabled (`true` or `false`). The `operation` field controls how body changes are applied: `append` (default), `prepend`, or `replace`. Accepts `target` (`"triggering"`, `"*"`, or a specific number) and cross-repository updates via `target-repo`. When `target: "*"` is used, the agent must supply `pull_request_number` in the tool output. The optional `update-branch: true` field synchronizes the PR branch with the latest base branch changes before applying other updates. See [Safe Outputs (Pull Requests)](/gh-aw/reference/safe-outputs-pull-requests/#pull-request-updates-update-pull-request).

### Protected Files

A security mechanism on `create-pull-request` and `push-to-pull-request-branch` safe outputs that prevents AI agents from modifying sensitive repository files. By default, protects dependency manifests (e.g., `package.json`, `go.mod`), GitHub Actions workflow files, and lock files. Configured via `protected-files:` with three policies: `blocked` (default — fails with error), `allowed` (no restriction), or `fallback-to-issue` (creates a review issue for human inspection instead of applying changes). Also accepts an object form `{ policy: string, exclude: [...] }` to remove specific files or path prefixes from the default protected set while keeping protection active for the remaining files. See [Safe Outputs (Pull Requests)](/gh-aw/reference/safe-outputs-pull-requests/#protected-files).

### Allow Workflows (`allow-workflows:`)

A field on `create-pull-request` and `push-to-pull-request-branch` safe outputs that adds `workflows: write` to the GitHub App token's permissions. Required when `allowed-files:` targets paths under `.github/workflows/`, because the `workflows` permission is a GitHub App-only permission that cannot be granted via `GITHUB_TOKEN`. Requires a `safe-outputs.github-app` configuration — the compiler rejects `allow-workflows: true` without one. This opt-in design keeps the elevated permission visible and auditable in the workflow source. See [Safe Outputs (Pull Requests)](/gh-aw/reference/safe-outputs-pull-requests/#allowing-workflow-file-changes-with-allow-workflows).

### Allowed Events (`allowed-events:`)

A field on `submit-pull-request-review:` safe outputs that restricts which PR review event types the agent may submit. Accepts an array of `APPROVE`, `COMMENT`, and `REQUEST_CHANGES`. When set, the safe-outputs handler rejects any review event not in the list, providing infrastructure-level enforcement regardless of what the agent attempts to output. If omitted, all three event types are allowed. Preferred default for bot reviews: `allowed-events: [COMMENT]`. Example: `allowed-events: [COMMENT, REQUEST_CHANGES]` prevents the agent from approving PRs. See [Safe Outputs Reference](/gh-aw/reference/safe-outputs/#submit-pr-review-submit-pull-request-review).

### Supersede Older Reviews (`supersede-older-reviews:`)

A field on `submit-pull-request-review:` safe outputs that dismisses older `REQUEST_CHANGES` reviews from the same workflow after posting a replacement review. When `supersede-older-reviews: true` is set, the safe-output handler fetches recent reviews, identifies prior `REQUEST_CHANGES` reviews submitted by the same workflow call, and dismisses them before the new review takes effect. This is best-effort behavior — dismissal failures do not block the new review. Useful when a workflow is configured with `allowed-events: [REQUEST_CHANGES]` and repeated runs would otherwise accumulate blocking reviews. See [Safe Outputs (Pull Requests)](/gh-aw/reference/safe-outputs-pull-requests/#submit-pr-review-submit-pull-request-review).

### Deduplicate by Title (`deduplicate-by-title:`)

A `create-issue` safe-output field that drops duplicate issues before creation by comparing titles. Accepts `true` for exact matching (after normalization) or an integer `0`–`100` for fuzzy matching within the given Levenshtein edit distance (e.g., `1` allows one-character differences). Deduplication runs at MCP tool-call time (within-run) and at apply time (against open and recently-closed repository issues). Dropped items are recorded in the safe-output summary with the matched title, edit distance, and source. See [Safe Outputs Reference](/gh-aw/reference/safe-outputs/#issue-creation-create-issue).

### Allowed Fields (`create-issue:`)

A configuration field on `create-issue:` safe outputs that restricts which GitHub Project custom fields the agent may set when creating issues. Accepts an array of field names (e.g., `[Priority, Iteration]`). When set, the safe-outputs handler rejects any attempt to populate a field not in the list. When omitted, all project fields are permitted. Example: `allowed-fields: [Priority, Iteration]`. See [Safe Outputs Reference](/gh-aw/reference/safe-outputs/#issue-creation-create-issue).

### Allowed Files

An exclusive allowlist for `create-pull-request` and `push-to-pull-request-branch` safe outputs. When `allowed-files:` is set to a list of glob patterns, **only** files matching those patterns may be modified — every other file (including normal source files) is refused. This is a restriction, not an exception: listing `.github/workflows/*` does not additionally allow normal source files; it blocks them. Runs independently from [Protected Files](#protected-files): both checks must pass. To modify a protected file, it must both match `allowed-files` and have `protected-files: allowed`. See [Safe Outputs (Pull Requests)](/gh-aw/reference/safe-outputs-pull-requests/#restricting-changes-to-specific-files-with-allowed-files).

### Branch Prefix (`branch-prefix:`)

An optional field on `create-pull-request` safe outputs that prepends a fixed string to the agent-specified or auto-generated branch name. Useful when repository policies require branches to follow naming conventions (e.g., `signed/` for signed-commit workflows). The default prefix is `signed/`. See [Safe Outputs (Pull Requests)](/gh-aw/reference/safe-outputs-pull-requests/).

### Preserve Branch Name (`preserve-branch-name:`)

An option on `create-pull-request` safe outputs that omits the random hex salt suffix normally appended to the agent-specified branch name. Useful when the target repository enforces naming conventions such as Jira keys in uppercase (for example, `bugfix/BR-329-red` instead of `bugfix/br-329-red-cde2a954`). Invalid characters are always replaced for safety, and casing is always preserved regardless of this setting. Defaults to `false`. See [Safe Outputs (Pull Requests)](/gh-aw/reference/safe-outputs-pull-requests/).

### Max Patch Files (`max-patch-files:`)

A `create-pull-request` safe-output field that sets the maximum number of unique files allowed in a single PR's patch. Defaults to `100`. Workflows that regenerate large sets of files (e.g., per-package API schemas) can raise this limit. If the limit is exceeded, PR creation fails with an actionable error showing the exact file count and the field to configure. See [Safe Outputs (Pull Requests)](/gh-aw/reference/safe-outputs-pull-requests/).

### Max Patch Size (`max-patch-size:`)

A `create-pull-request` and `push-to-pull-request-branch` safe output field that limits the total size of the git patch in kilobytes. Accepts an integer in the range 1–10,240 KB. Defaults to `4096` KB (4 MB). If the patch exceeds the limit, PR creation fails with an actionable error. Useful when workflows generate large diffs and the default limit is too restrictive or too permissive. See [Safe Outputs (Pull Requests)](/gh-aw/reference/safe-outputs-pull-requests/).

### Recreate Ref (`recreate-ref:`)

An option on `create-pull-request` safe outputs that force-deletes and recreates the remote branch when the agent-supplied branch name already exists on the remote. Requires `preserve-branch-name: true`. The handler force-pushes the agent's local HEAD to the stale remote ref, enabling reuse of long-lived reusable branches whose previous PR was merged. Without `recreate-ref: true`, the default behavior is to fall back (for example, open an issue when `fallback-as-issue: true`) rather than overwrite the remote. Defaults to `false`. See [Safe Outputs (Pull Requests)](/gh-aw/reference/safe-outputs-pull-requests/).

### Create Pull Request Review Comment (`create-pull-request-review-comment:`)

A safe output capability for posting inline review comments on specific lines in a pull request diff. Supports single-line and multi-line comments with configurable `side` (`LEFT` or `RIGHT`). When `target: "*"` is set, the agent must supply `pull_request_number` in the tool call. For cross-repository scenarios, the agent may also supply `repo` (in `owner/repo` format) matching `target-repo` or `allowed-repos`. See [Safe Outputs (Pull Requests)](/gh-aw/reference/safe-outputs-pull-requests/#pr-review-comments-create-pull-request-review-comment).

### Reply to PR Review Comment (`reply-to-pull-request-review-comment:`)

A safe output capability for replying to existing review comments on pull requests. Allows the AI agent to respond to reviewer feedback, answer questions, or acknowledge inline review comments by their numeric comment ID. Supports an optional `footer` field (`always`, `none`, or `if-body`) to control AI attribution. Configured via `reply-to-pull-request-review-comment:` in `safe-outputs`. See [Safe Outputs (Pull Requests)](/gh-aw/reference/safe-outputs-pull-requests/#reply-to-pr-review-comment-reply-to-pull-request-review-comment).

### Resolve PR Review Thread (`resolve-pull-request-review-thread:`)

A safe output capability for marking GitHub PR review threads as resolved. Uses the GitHub GraphQL `resolveReviewThread` mutation, requiring the thread's node ID. Allows AI agents to clean up addressed review comments after implementing feedback. Accepts the same `target`, `target-repo`, and `allowed-repos` options as other pull-request safe outputs. See [Safe Outputs (Pull Requests)](/gh-aw/reference/safe-outputs-pull-requests/#resolve-pr-review-thread-resolve-pull-request-review-thread).

### Report Incomplete (`report_incomplete`)

A mandatory safe output signal that agents emit when a task cannot be completed due to an infrastructure or tool failure — for example, an MCP server crash, missing authentication, or an inaccessible repository. Unlike `noop` (which signals no action was needed), `report_incomplete` indicates an active failure that prevented the task from running. The safe-outputs handler activates failure handling regardless of agent exit code. Accepts a required `reason` field (max 1024 characters) and an optional `details` field for extended diagnostic context.

### Set Issue Type (`set-issue-type:`)

A safe output capability for setting or clearing the GitHub issue type on existing issues. The agent calls `set_issue_type` to assign a named type (e.g., `Bug`, `Feature`) to an issue. An `allowed` list restricts which types the agent may set; omitting it permits any type. Passing an empty string clears the current type. Supports cross-repository targeting via `target-repo` and `allowed-repos`. Configured via `set-issue-type:` in `safe-outputs`.

### Set Issue Field (`set-issue-field:`)

A safe output capability for setting one issue field value on existing issues. The agent calls `set_issue_field` with `value` and either `field_name` (for discovery by field label) or `field_node_id` (to skip discovery). Unknown field names return actionable errors listing available fields and suggesting explicit IDs. Supports optional `allowed-fields` restrictions (including `["*"]` wildcard) and cross-repository targeting via `target-repo` and `allowed-repos`. Configured via `set-issue-field:` in `safe-outputs`.

### Parameterized Safe-Output Fields

A pattern for `workflow_call` reuse where safe-output policy and list fields accept GitHub Actions expression strings (e.g., `${{ inputs.protected-files-policy }}`) in addition to literal values. At compile time the compiler detects the `${{...}}` form and passes it through unchanged; GitHub Actions evaluates the expression at runtime before the handler executes. Enum-valued policy fields such as `protected-files` and `patch-format` validate literal values at compile time but defer expression-based values to runtime (failing closed on unrecognized input). List-valued fields such as `labels`, `allowed-repos`, and `allowed-base-branches` accept either a YAML array or a single expression string. This enables a single reusable workflow to serve callers with different constraint configurations without duplicating files. See [Safe Outputs (Pull Requests)](/gh-aw/reference/safe-outputs-pull-requests/#parameterizing-policy-fields-in-reusable-workflows).

## Workflow Components

### Activation Token (`on.github-token:`, `on.github-app:`)

Custom GitHub token or GitHub App used by the activation job to post reactions and status comments on the triggering item. Configured via `github-token:` (for a PAT or token expression) or `github-app:` (to mint a short-lived installation token) inside the `on:` section. Affects only the activation job — agent job tokens are configured separately via `tools.github.github-token` or `safe-outputs.github-app`. See [Authentication Reference](/gh-aw/reference/auth/).

### BYOK (Bring Your Own Key)

A Copilot engine mode that routes AI requests to an external LLM provider (such as OpenAI, Anthropic, or a self-hosted Ollama/vLLM instance) instead of the default GitHub Copilot backend. Activated by setting `COPILOT_PROVIDER_BASE_URL` in `engine.env`. The three BYOK credential variables (`COPILOT_PROVIDER_BASE_URL`, `COPILOT_PROVIDER_API_KEY`, `COPILOT_PROVIDER_BEARER_TOKEN`) accept `${{ secrets.* }}` references under strict mode and are never exposed to the agent container. Use `COPILOT_MODEL` to specify the target model. See [AI Engines Reference](/gh-aw/reference/engines/#copilot-bring-your-own-key-byok-mode).

### Cron Schedule

A time-based trigger format. Use short syntax like `daily` or `weekly on monday` (recommended with automatic time scattering) or standard cron expressions for fixed times. Cron-based schedule items accept an optional `timezone` field with any [IANA timezone identifier](https://en.wikipedia.org/wiki/List_of_tz_database_time_zones) (e.g., `America/New_York`) to interpret the expression in a specific timezone instead of UTC. See also [Fuzzy Scheduling](#fuzzy-scheduling) and [Time Scattering](#time-scattering).

### Ecosystem Identifiers

Named shorthand references to predefined domain sets used in `network.allowed` and `safe-outputs.allowed-domains`. Instead of listing individual domain names, ecosystem identifiers expand to curated sets for a language runtime or service category. Common identifiers: `python` (PyPI/pip), `node` (npm), `go` (proxy.golang.org), `github` (GitHub domains), `dev-tools` (CI/CD services such as Codecov, Snyk, Shields.io), `local` (loopback addresses), and `default-safe-outputs` (a compound set combining `defaults` + `dev-tools` + `github` + `local`, recommended as a baseline for `safe-outputs.allowed-domains`). See [Network Permissions Reference](/gh-aw/reference/network/#ecosystem-identifiers).

### Engine

The AI system that powers the agentic workflow - essentially "which AI to use" to execute workflow instructions. GitHub Agentic Workflows supports seven engines: **Copilot** (default), **Claude**, **Codex**, **Gemini**, **Crush** (experimental), **OpenCode** (experimental), and **Pi** (experimental). Set `engine:` in frontmatter to choose; omit it to use Copilot. See [AI Engines Reference](/gh-aw/reference/engines/).

### Anthropic Workload Identity Federation (WIF)

A keyless authentication method for the Claude engine that uses short-lived GitHub OIDC tokens instead of a long-lived `ANTHROPIC_API_KEY` secret. Configured via [`engine.auth`](#engine-auth-engineauth) with `type: github-oidc` and `provider: anthropic`, along with Anthropic-specific IDs (`federation-rule-id`, `organization-id`, `service-account-id`, `workspace-id`) obtained from the Anthropic Console. Requires `permissions: id-token: write`. Available since v0.79.6. See [Authentication Reference](/gh-aw/reference/auth/#anthropic-workload-identity-federation-wif).

### Engine Auth (`engine.auth`)

An `engine:` configuration field for keyless authentication. When `engine.auth` is configured, the compiler emits authentication environment variables consumed by the AWF api-proxy sidecar and suppresses the static API key requirement. Supports `type: github-oidc` with an optional `provider` for service-specific token exchange — for example, `provider: anthropic` for [Anthropic WIF](#anthropic-workload-identity-federation-wif) or without a provider for Azure Entra ID BYOK mode. Requires `permissions: id-token: write` in the workflow. See [Authentication Reference](/gh-aw/reference/auth/) and [AI Engines Reference](/gh-aw/reference/engines/).

```aw wrap
engine:
  id: claude
  auth:
    type: github-oidc
    provider: anthropic
    federation-rule-id: fdrl_xxxxxxxxxxxx
    organization-id: org_xxxxxxxxxxxx
    service-account-id: svac_xxxxxxxxxxxx
    workspace-id: ws_xxxxxxxxxxxx
```

### Engine Permission Mode (`engine.permission-mode`)

A first-class Claude engine setting that controls how Claude Code enforces tool access boundaries. Accepts one of four values: `acceptEdits` (default — Claude honors `--allowed-tools`; the workflow's declared `tools:` and `mcp-servers: allowed:` list is the effective tool boundary), `bypassPermissions` (Claude ignores `--allowed-tools`; the MCP gateway's `allowed:` filter becomes the sole boundary), `auto` (Claude selects the least-privileged mode that fits the workflow's tool configuration; the default when `tools.edit: false`), and `plan` (Claude presents changes for approval before applying them).

Previously, `bypassPermissions` was derived implicitly whenever a workflow granted unrestricted bash access (`bash: "*"`, `bash: [":*"]`, or `bash: null`), which could silently disable `--allowed-tools` enforcement. Setting `engine.permission-mode` explicitly overrides that implicit derivation and any legacy `--permission-mode` flag in `engine.args`. The compiler validates the value against the fixed enum at compile time. See [AI Engines Reference](/gh-aw/reference/engines/#claude-tool-enforcement-security-model).

```aw wrap
engine:
  id: claude
  permission-mode: acceptEdits
```

### Enterprise API Endpoint (`api-target`)

An `engine` configuration field specifying a custom API endpoint hostname for GitHub Enterprise Cloud (GHEC) or GitHub Enterprise Server (GHES) deployments. When set, the compiler automatically adds both the API domain and the base hostname to the AWF firewall `--allow-domains` list and the `GH_AW_ALLOWED_DOMAINS` environment variable, eliminating the need for manual network configuration after each recompile. The value must be a hostname only — no protocol or path (e.g., `api.acme.ghe.com`). See [Engines Reference](/gh-aw/reference/engines/#enterprise-api-endpoint-api-target).

```aw wrap
engine:
  id: copilot
  api-target: api.acme.ghe.com
```

### Inline Engine Definition

An engine configuration format that specifies a runtime adapter and optional provider settings directly in workflow frontmatter, without requiring a named catalog entry. Uses a `runtime` object (with `id` and optional `version`) to identify the adapter and an optional `provider` object for model selection, authentication, and request shaping. Useful for connecting to self-hosted or third-party AI backends.

```aw wrap
engine:
  runtime:
    id: codex
  provider:
    id: azure-openai
    model: gpt-4o
    auth:
      strategy: oauth-client-credentials
      token-url: https://auth.example.com/oauth/token
      client-id: AZURE_CLIENT_ID
      client-secret: AZURE_CLIENT_SECRET
    request:
      path-template: /openai/deployments/{model}/chat/completions
      query:
        api-version: "2024-10-01-preview"
```

See [Engines Reference](/gh-aw/reference/engines/).

### Experiments (`experiments:`)

A frontmatter section that enables A/B testing of workflow prompt variants across successive runs. Each key in the `experiments:` map names an experiment; the value is either a bare array of variant strings or a rich object with additional fields (`variants`, `description`, `hypothesis`, `metric`, `weight`, `min_samples`, `start_date`, `end_date`). At runtime the activation job selects one variant per experiment using a balanced round-robin counter and exposes the selection as `${{ experiments.<name> }}` for use anywhere in the workflow body.

Experiment state is persisted to dedicated `experiments/<name>` git branches in the workflow repository. Use `gh aw experiments list` and `gh aw experiments analyze` to inspect variant distribution and statistical readiness (chi-square balance test, Bonferroni correction, EXTEND / READY_FOR_ANALYSIS recommendation). See [A/B Experiments](/gh-aw/experimental/experiments/) and the [Experiments Specification](/gh-aw/experimental/experiments-specification/).

```aw wrap
experiments:
  prompt_style: [concise, detailed]
---
Summarize this issue in a **${{ experiments.prompt_style }}** way.
```

### Feature Flags (`features:`)

A frontmatter section that enables experimental or optional compiler and runtime behaviors as key-value pairs. Feature flags provide controlled access to new capabilities before they become defaults or are fully stabilized. Common flags include `action-mode` (controls how custom action references are compiled), `copilot-requests` (enables GitHub Actions token authentication for Copilot; currently in **private preview** — will not work unless your account has been onboarded), `mcp-gateway` (enables the MCP gateway proxy), `integrity-reactions` (enables reaction-based integrity promotion and demotion), `cli-proxy` (enables CLI proxy mode for integrity enforcement at the network boundary), and `awf-diagnostic-logs` (enables AWF Docker operational diagnostics collection on failure). `byok-copilot` is deprecated because Copilot BYOK behavior is now the default for `engine: copilot`. See [Feature Flags Reference](/gh-aw/reference/feature-flags/).

### Force Clean Git Credentials (`force-clean-git-credentials`)

A `checkout:` option that opts into an explicit credential cleanup step instead of relying on `actions/checkout`'s `persist-credentials: false` post-step cleanup. When set to `true`, the compiler emits the checkout with `persist-credentials: true` and injects a dedicated `Clean git credentials after checkout` step immediately after it. That step scrubs the credential helper and `http.*.extraheader` entries from `.git/config` and all nested submodule configs. Use this for submodule-heavy or sparse checkouts where the default post-step cleanup fails. See [Checkout Reference](/gh-aw/reference/checkout/).

### Fuzzy Scheduling

Natural language schedule syntax that automatically distributes workflow execution times to avoid load spikes. Instead of specifying exact times with cron expressions, fuzzy schedules like `daily`, `weekly`, or `daily on weekdays` are converted by the compiler into deterministic but scattered cron expressions. The compiler automatically adds `workflow_dispatch:` trigger for manual runs. Example: `schedule: daily on weekdays` compiles to something like `43 5 * * 1-5` with varied execution times across different workflows.

### Imports

Reusable workflow components shared across multiple workflows. Specified in the `imports:` field, can include tool configurations, common instructions, or security guidelines. Shared files without an `on:` field are validated but not compiled into GitHub Actions — they are only importable by other workflows.

Imports support a parameterized form using `uses`/`with` syntax when the shared file declares an `import-schema`. The compiler validates the passed values, substitutes them into the shared file, and errors on conflicting imports of the same file. See [Imports Reference](/gh-aw/reference/imports/).

### Pre-Agent Steps (`pre-agent-steps:`)

Steps injected into the agent job after artifacts are downloaded and before the engine executes. Defined in the `pre-agent-steps:` frontmatter field and composable via imports — imported pre-agent-steps are prepended to the main workflow's steps in import order. Useful for setup tasks such as installing dependencies or configuring the environment before the AI engine runs. See [Imports Reference](/gh-aw/reference/imports/).

### Post-Steps (`post-steps:`)

Steps injected into the agent job after the engine finishes execution. Defined in the `post-steps:` frontmatter field and composable via imports — imported post-steps are appended after the main workflow's post-steps in import order. Useful for cleanup, reporting, or artifact publishing after the AI engine completes. See [Imports Reference](/gh-aw/reference/imports/).

### Import Schema (`import-schema`)

A typed parameter contract declared in a shared workflow file that enables callers to pass values via `uses`/`with` syntax. The compiler validates each caller's `with` values against the schema and substitutes them into the shared file's frontmatter and body before processing. Supports typed fields with optional defaults; required fields without defaults cause a compile-time error if omitted. See [Imports Reference](/gh-aw/reference/imports/#import-schema-import-schema).

### MCP Gateway Settings (`engine.mcp`)

`engine.mcp` is the subset of `engine:` configuration that controls MCP gateway behavior — specifically `tool-timeout` and `session-timeout`. Shared workflow files can export only these settings (without specifying an engine identifier), allowing importers to inherit MCP timeout configuration without coupling a shared component to a specific engine. The importing workflow's own `engine.mcp` values take precedence; among imports, the first-wins strategy applies. See [Imports Reference — Importing MCP Gateway Settings](/gh-aw/reference/imports/#importing-mcp-gateway-settings).

### Runtime Import (`{{#runtime-import}}`)

A body-level directive that injects the text content of another file at a specific point in the workflow markdown. Unlike the `imports:` frontmatter field (which merges configuration), `{{#runtime-import filepath}}` splices raw markdown text — useful for sharing reusable prompt snippets, tone instructions, or reference material. Use `{{#runtime-import? filepath}}` for an optional include that silently skips a missing file. Paths are resolved within the `.github` folder with or without the `.github/` prefix. See [Runtime Imports](/gh-aw/reference/templating/#runtime-imports).

### Emoji (`emoji:`)

An optional frontmatter field that attaches an emoji to represent the workflow visually in listings and UI surfaces. Accepts a single emoji character (e.g., `"🤖"`). See [Frontmatter Reference](/gh-aw/reference/frontmatter/).

### Label Trigger Shorthand

A compact syntax for label-based triggers: `on: issue labeled bug` or `on: pull_request labeled needs-review`. The compiler expands the shorthand to standard GitHub Actions trigger syntax and automatically includes a `workflow_dispatch` trigger with an `inputs.item_number` parameter, enabling manual dispatch for a specific issue or pull request. Supported for `issue`, `pull_request`, and `discussion` events. See [LabelOps patterns](/gh-aw/patterns/label-ops/).

### Labels

Optional workflow metadata for categorization and organization. Enables filtering workflows in the CLI using the `--label` flag.

### Model Alias

A short human-friendly name (such as `sonnet` or `mini`) that gh-aw resolves to the best available concrete model at compile time. Aliases are defined as ordered lists of provider-scoped glob patterns; the first pattern that matches an available model wins. Meta-aliases reference other aliases and are resolved recursively. Built-in vendor aliases and meta-aliases are listed in the [Model Aliases & Multipliers Reference](/gh-aw/reference/model-tables/). Custom aliases can be defined in workflow frontmatter using the [Model Alias Format Specification](/gh-aw/specs/model-alias-specification/).

### Max AI Credits (`max-ai-credits`)

A top-level frontmatter field that caps the total AI Credits (AIC) the AWF proxy will spend within a single workflow run. Applies to all engines and maps to `apiProxy.maxAiCredits` in the compiled lock file. Defaults to `1000` when omitted. Accepts an integer, an optional `K`/`M` suffix string (for example, `100M`), or a GitHub Actions expression that resolves to an integer at runtime. Example:

```aw wrap
max-ai-credits: 500
```

### Max Daily AI Credits (`max-daily-ai-credits`)

A top-level frontmatter field that sets a 24-hour AI Credits cap for a single workflow, aggregated across all recent runs of the same workflow in the repository. When the activation job detects that the previous 24 hours already exceed this threshold, it warns, creates an issue, skips the agent job, and reports a specialized failure. Disabled by default when omitted. Set to `-1` to explicitly disable it. Accepts plain integers or `K`/`M` suffixes (e.g., `100M`). Skipped for `workflow_call`, `repository_dispatch`, and `workflow_dispatch` runs carrying internal `aw_context` dispatch metadata. Example:

```aw wrap
max-daily-ai-credits: 15M
```

See [Cost Management](/gh-aw/reference/cost-management/) and [Compiler Enterprise Environment Controls](/gh-aw/reference/compiler-enterprise-environment-controls/).

### Max Runs (`max-runs`, deprecated)

Deprecated top-level alias for the AWF invocation cap. Use `max-turns` instead. This cap limits the number of times the AWF proxy invokes the AI engine within a single workflow run and maps to `apiProxy.maxRuns` in the compiled lock file. Defaults to `500` when omitted. Accepts an integer or a GitHub Actions expression that resolves to an integer at runtime. Example:

```aw wrap
max-turns: 10
```

See [Engines Reference](/gh-aw/reference/engines/).

### Max Turns (`max-turns`)

A top-level frontmatter field that caps the number of chat iterations (model responses and tool calls) for a single workflow run. Each additional turn consumes more tokens and Actions compute time, so a turn limit bounds runaway loops and cost. Supported across Claude, Codex, Copilot, and Antigravity engines. Compiles to the `GH_AW_MAX_TURNS` environment variable for the engine runtime. Accepts an integer or a GitHub Actions expression. The deprecated alias `engine.max-turns` continues to compile; use `gh aw fix engine-max-turns-to-top-level` to migrate. Example:

```aw wrap
max-turns: 20
```

See [Cost Management](/gh-aw/reference/cost-management/).

### Network Permissions

Controls over external domains and services a workflow can access. Configured via `network:` section with options: `defaults` (common infrastructure), custom allow-lists, or `{}` (no access).

### Network Allowed Input (`network.allowed-input`)

An opt-in frontmatter flag for `workflow_call` workflows that exposes a `network_allowed` input parameter, allowing callers to extend the compiled workflow's network allowlist at runtime. When enabled with `network.allowed-input: true`, the compiler injects a `network_allowed: string` input into the `workflow_call` interface. Callers provide a comma-separated list of ecosystem identifiers and/or domains that are unioned with the static `network.allowed` baseline before the agent starts. The compiled workflow's static allowlist acts as an immutable floor — callers can only add domains, never remove them. Useful for reusable workflows that serve consumers with varying network requirements without requiring per-consumer forks or recompilation. See ADR-33200 for implementation details.

```aw wrap
on: workflow_call
network:
  allowed: [defaults]
  allowed-input: true  # Caller can extend with network_allowed: "python,rust"
```

### Observability (`observability.otlp`)

A frontmatter field that enables OpenTelemetry trace
export from workflow runs. It supports single-endpoint and
multi-endpoint OTLP export with optional headers.

See [OpenTelemetry](/gh-aw/guides/open-telemetry/) for
setup, runtime variables, and span semantics.

### OTLP If-Missing (`observability.otlp.if-missing`)

Controls behavior when OTLP endpoint or header values resolve to empty at runtime. Accepts `error` (default — fails startup), `warn` (logs a warning and skips MCP gateway OTLP configuration), or `ignore` (silently skips MCP gateway OTLP configuration). Useful in shared imports where OTLP secrets may be absent in some repositories — set to `ignore` to make observability opt-in without breaking workflows that lack the secrets. See [Frontmatter](/gh-aw/reference/frontmatter/#observability-observability).

### Setup-Steps (`jobs.<job-id>.setup-steps`)

Steps injected immediately after the compiler-generated `actions/setup` step for a custom or built-in job. Defined under `jobs.<job-id>.setup-steps` in workflow frontmatter. When both a main workflow and an imported workflow define `setup-steps` for the same job, imported setup-steps run first. `setup-steps` remain distinct from `pre-steps` and are not merged across keys. `jobs.activation.setup-steps` and `jobs.pre_activation`/`jobs.pre-activation` `setup-steps` are refused at compile time because they can short-circuit protections. See [Custom Jobs](/gh-aw/reference/steps-jobs/#jobs-and-steps).

### Pre-Steps (`jobs.<job-id>.pre-steps`)

Steps injected after any compiler setup scaffolding and job-level `setup-steps`, but before the main work for that job. Defined under `jobs.<job-id>.pre-steps` in workflow frontmatter. When both a main workflow and an imported workflow define `pre-steps` for the same job, imported pre-steps run first. This is distinct from both `jobs.<job-id>.setup-steps` and the top-level `pre-steps` field, which injects steps into the agent job only. See [Custom Jobs](/gh-aw/reference/steps-jobs/#jobs-and-steps).

### Pre-Activation Dependencies (`on.needs:`)

A frontmatter field that declares custom jobs that both the `pre_activation` and `activation` built-in jobs depend on. Use this when credentials or secrets must be fetched by a custom job before activation runs — for example, when `on.github-app` tokens come from a secrets-manager job. Values must reference custom jobs defined in the top-level `jobs:` section; built-in job names are rejected at compile time. See [Triggers Reference](/gh-aw/reference/triggers/).

### Stop After

A workflow configuration field (`stop-after:`) that automatically prevents new runs after a specified time limit. Accepts absolute dates (`YYYY-MM-DD`, ISO 8601) or relative time deltas (`+48h`, `+7d`). Minimum granularity is hours. Useful for trial periods, experimental features, and cost-controlled schedules. Recompile with `gh aw compile --refresh-stop-time` to reset the deadline. See [Ephemerals](/gh-aw/reference/ephemerals/).

### `deployment_status` Trigger

A GitHub Actions trigger that fires when an external deployment changes state. Supported states are `error`, `failure`, `pending`, `queued`, `in_progress`, `success`, `inactive`, and `waiting`. The gh-aw compiler accepts an optional `state:` filter in the trigger definition and synthesizes a job-level `if:` condition so that the agent only runs for the specified states. A natural-language shorthand is also supported — `on: "deployment failed"` expands to `deployment_status` with `state: [failure]`. See [Frontmatter Reference](/gh-aw/reference/frontmatter/).

```aw wrap
on:
  deployment_status:
    state: [error, failure]
```

### Triggers

Events that cause a workflow to run, defined in the `on:` section of frontmatter. Includes issue events, pull requests, schedules, manual runs, and slash commands.

### Skip Author Associations (`on.skip-author-associations`)

A pre-activation gating mechanism that skips workflow execution when the triggering event's author has a specific `author_association` value (such as `contributor`, `first_time_contributor`, or `none`). Configured per-event in the `on.skip-author-associations` field. Compiles to a job-level `if` expression — no runtime script step cost for skipped runs. Values are case-insensitive and accept a single string or array of strings per event key. See [Triggers Reference](/gh-aw/reference/triggers/).

### Trigger File

A plain GitHub Actions workflow (`.yml`) that separates trigger definitions from agentic workflow logic. Calls a compiled orchestrator's `workflow_call` entry point in response to any GitHub event (issues, pushes, labels, manual dispatch). Decouples trigger changes from the compilation cycle — updating when an orchestrator runs requires editing only the trigger file, not recompiling the agentic workflow.

Trigger files can live in the **same repository** as the orchestrator or in a **different repository** (cross-repo `workflow_call`). Cross-repo usage requires the callee repository to be public, internal, or to have explicitly granted Actions access. When using `secrets: inherit`, the caller's secrets are passed through — including `COPILOT_GITHUB_TOKEN`, which must be configured in the caller's repository. See [MultiRepoOps](/gh-aw/patterns/multi-repo-ops/).

### User Rate Limit (`user-rate-limit`)

A frontmatter field that prevents individual users from triggering a workflow too frequently. Configured with `max-runs-per-window` (maximum runs per time window, 1–10), an optional `window` in minutes (default 60, max 180), an optional `events` list to restrict which trigger types count, and an optional `ignored-roles` list of exempt roles (default: `[admin, maintain, write]`). The pre-activation job checks recent runs and cancels the current run if the limit is exceeded. Example:

```aw wrap
user-rate-limit:
  max-runs-per-window: 5
  window: 60
  ignored-roles: []
```

See [Rate Limiting Controls](/gh-aw/reference/rate-limiting-controls/).

### Weekday Schedules

Scheduled workflows configured to run only Monday through Friday using `daily on weekdays` syntax. Recommended for daily workflows to avoid the "Monday wall of work" where tasks accumulate over weekends and create a backlog on Monday morning. The compiler converts this to cron expressions with `1-5` in the day-of-week field. Example: `schedule: daily on weekdays` generates a cron like `43 5 * * 1-5`.

### workflow_call

A trigger enabling a compiled workflow to be invoked by another workflow in the same organization. Adding `workflow_call` to the `on:` section exposes the lock file as a callable workflow, with optional inputs callers can pass for context. Commonly used with a [Trigger File](#trigger-file) to decouple trigger definitions from agentic workflow compilation. See [MultiRepoOps](/gh-aw/patterns/multi-repo-ops/).

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

A GitHub App installation used for authentication and token minting in workflows. The `github-app:` field (which replaces the deprecated `app:` key) accepts `client-id` (preferred) or `app-id` (deprecated alias) together with `private-key` to mint short-lived installation access tokens with fine-grained, automatically-revoked permissions. Can be configured in `safe-outputs:` to override the default `GITHUB_TOKEN` for all safe output operations, or in `checkout:` for accessing private repositories. Run `gh aw fix` to automatically migrate `app-id` to `client-id`. See [Authentication Reference](/gh-aw/reference/auth/#using-a-github-app-for-authentication).

### YAML

A human-friendly data format for configuration files using indentation and simple syntax to represent structured data. In agentic workflows, YAML appears in frontmatter and compiled `.lock.yml` files.

### Personal Access Token (PAT)

A token authenticating you to GitHub's APIs with specific permissions. Required for GitHub Copilot CLI to access Copilot services. Created at github.com/settings/personal-access-tokens.

### Skill Files

Markdown files with YAML frontmatter stored in `.github/skills/` for repository-scoped Copilot Chat skills. Created by `gh aw init`, these files can be invoked by calling the skill in Copilot Chat to guide workflow creation, debugging, and updates. The `agentic-workflows` skill is a unified dispatcher routing requests to specialized prompts.

### Fine-grained Personal Access Token

A GitHub Personal Access Token with granular permission control, specifying exactly which repositories the token can access and what permissions it has. Created at github.com/settings/personal-access-tokens.

### `RUNNER_TEMP` / `${{ runner.temp }}`

A GitHub Actions environment variable pointing to a per-job temporary directory on the runner. Agentic workflows store compiled scripts and runtime artifacts under `${RUNNER_TEMP}/gh-aw/` for compatibility with self-hosted runners that may not have write access to system directories. In shell `run:` blocks, use the shell variable form `${RUNNER_TEMP}`; in `with:` or `env:` YAML fields, use the expression form `${{ runner.temp }}`.

## Development and Compilation

### CLI (Command Line Interface)

The `gh aw` extension for GitHub CLI providing commands for managing agentic workflows: compile, run, status, logs, add, deploy, and project management.

### Codemod

An automated transformation script applied by `gh aw fix` that updates workflow markdown files from deprecated syntax to the current format. Codemods rename frontmatter keys, restructure values, or remove obsolete settings without changing workflow behavior. They run in dry-run mode by default; pass `--write` to apply changes. `gh aw upgrade` applies all relevant codemods automatically as part of the upgrade process. List available codemods with `gh aw fix --list-codemods`. See [Upgrading](/gh-aw/guides/upgrading/).

### Playground

An interactive web-based editor for authoring, compiling, and previewing agentic workflows without local installation. The Playground runs the gh-aw compiler in the browser using [WebAssembly](#webassembly-wasm) and auto-saves editor content to `localStorage` so work is preserved across sessions. Available at `/gh-aw/editor/`.

### Audit (`gh aw audit`)

A CLI command that downloads workflow run artifacts and logs, analyzes MCP tool usage and network behavior, and generates a structured Markdown or JSON report. The report covers failure analysis, tool usage, MCP server status, firewall activity, token/cost metrics, behavior fingerprint, and safe-output summary. Accepts a numeric run ID or any GitHub Actions run or job URL. See [Audit Commands](/gh-aw/reference/audit/).

### Audit Diff (multi-run mode)

Passing two or more run IDs to `gh aw audit` activates diff mode: the first ID is the base and the rest are compared against it. Reports domain additions and removals, allowed/denied status changes, request volume drift, and anomaly flags across firewall, MCP tool usage, and run metrics dimensions. Useful for detecting regressions and behavioral drift between runs. See [Audit Commands](/gh-aw/reference/audit/).

### `usage` Artifact

A compact artifact produced by the conclusion job, containing workflow-run metadata and aggregated token-usage data used by lightweight reporting and forecasting paths. Unlike the full `agent` artifact, `usage` carries only summarized usage summaries — making it faster to download when detailed agent logs are not needed. Download with `gh aw logs <run-id> --artifacts usage`. See [Artifacts Reference](/gh-aw/reference/artifacts/).

### Behavior Fingerprint

A multi-dimensional characterization of a single workflow run produced by `gh aw audit`. Captures the task domain, network access patterns, tool usage profile, token consumption, and agentic assessments in a compact summary. Two runs with the same fingerprint exhibit identical observable behavior; diverging fingerprints signal regressions or unexpected changes. See [Audit Commands](/gh-aw/reference/audit/).

### Cross-Run Audit Report (`gh aw logs --format`)

A feature of `gh aw logs` that aggregates firewall, MCP, and metrics data across multiple workflow runs to produce a security and performance report. Includes an executive summary, domain inventory, and per-run breakdown with anomaly detection. Designed for security reviews, compliance checks, and feeding optimization agents. See [Audit Commands](/gh-aw/reference/audit/#gh-aw-logs---format-fmt).

### Deploy (`gh aw deploy`)

A CLI command that orchestrates full workflow rollout to a target repository in a single invocation. `gh aw deploy` clones the target repository, runs `update` to refresh any sourced workflows, runs `add` to install the requested workflows, runs `compile --purge` to regenerate lock files and remove stale outputs, then opens a pull request with all changes for review. Replaces the manual sequence of `clone → update → add → compile → pr` commands and skips the add phase for workflows that already carry a `source:` frontmatter field to prevent duplicate installations. Accepts `--repo` to specify the target repository and `--cool-down` to set the default scheduling interval. See [CLI Reference](/gh-aw/setup/cli/).

### AI Credits (AIC)

The primary inference-cost metric for GitHub Agentic Workflows. One AI Credit equals `0.01 USD` and is computed from input, output, cache-read, cache-write, and reasoning tokens multiplied by per-model pricing weights. AIC provides a model-normalized spend unit across all supported engines, enabling consistent budget governance and cost comparison. Reports from `gh aw audit` and `gh aw logs` expose AIC as `total_aic` (per episode or run) and per-request values. Use `max-ai-credits` and `max-daily-ai-credits` in workflow frontmatter to set budget caps. See [AI Credits Specification](/gh-aw/specs/ai-credits-specification/).

### Effective Tokens (ET)

The predecessor cost metric to [AI Credits (AIC)](#ai-credits-aic), computed as a weighted sum of input and output tokens to approximate relative inference cost. Deprecated in 2026 in favor of AI Credits, which provides direct monetary normalization (1 AIC = 0.01 USD) across all supported model providers. The `effective_tokens` field in audit JSON output is retained for backward compatibility; use `total_aic` for cost analysis in new workflows. See [Effective Tokens Specification](/gh-aw/specs/effective-tokens-specification/).

### Forecast (`gh aw forecast`)

An experimental CLI command that projects future AI Credits (AIC) consumption using a Monte Carlo simulation. It samples historical workflow runs, applies a Poisson-bootstrap algorithm to model run frequency, and returns P10/P50/P90 percentile estimates over a configurable time horizon. Supports both local (`.github/workflows/`) and remote (`--repo`) discovery modes. Output is available as a console table or machine-readable JSON (`--json`). Useful for capacity planning, budget governance, and detecting cost regressions before they occur. See [Forecast Specification](/gh-aw/specs/forecast-specification/).

### Time Between Turns (TBT)

The elapsed time between consecutive LLM API calls in an agentic workflow run. A "turn" is one complete LLM inference request; TBT measures the gap from when the model finishes one response (and tool calls are dispatched) to when the next request is sent (after all tool results are collected). TBT is an important performance and cost metric because LLM inference providers implement prompt caching with a fixed TTL:

- **Anthropic** reduced their cache TTL from 1 hour to **5 minutes**. If the TBT for any turn exceeds 5 minutes, the cached prompt context expires and the full prompt must be re-processed, significantly increasing token costs.
- **OpenAI** has a similar server-side prompt cache with variable TTL.

`gh aw audit` reports both average and maximum TBT in the Session Analysis section. A cache warning is emitted when the TBT used for cache analysis exceeds the Anthropic 5-minute threshold: the maximum observed TBT for Copilot engine runs, where precise per-turn timestamps are available in the `events.jsonl` session log, or the estimated average TBT for other engines, where TBT is derived from total wall time divided by turn count.

To reduce TBT — and keep prompt caches warm — minimize blocking tool calls, parallelize independent tool invocations, and avoid long-running shell commands in the critical path between turns.

### Ambient Context

The token footprint of the first LLM invocation in a workflow run, used as a proxy for the static context loaded at startup (system prompt, tools list, memory). Because the first invocation fires before the agent has accumulated any conversation history, its input token count primarily reflects the overhead of the configured environment rather than task-specific content. Reported as an optional `ambient_context` object in `gh aw audit` and `gh aw logs` JSON output with three fields: `input_tokens`, `cached_tokens`, and `effective_tokens`. Useful for comparing context overhead across different workflow configurations. See [Audit Commands](/gh-aw/reference/audit/).

### Firewall Analysis

A section of the `gh aw audit` report that breaks down all network requests made during a workflow run — showing allowed domains, denied domains, request volumes, and policy attribution. Derived from AWF firewall logs. Pass multiple run IDs to `gh aw audit` (e.g. `gh aw audit <base> <compare>`) to compare firewall behavior across runs and identify new or removed domain accesses. See [Audit Commands](/gh-aw/reference/audit/) and [Network Permissions](/gh-aw/reference/network/).

### Body Hash

A deterministic SHA-256 hash of the markdown body (the natural-language prompt text) of a workflow file. Complements the [Frontmatter Hash](#frontmatter-hash), which covers only configuration fields. The body hash is stored in the lock file metadata and checked at activation time when `on.stale-check: "full"` is set. Use `stale-check: "full"` when prompt-body edits should also trigger recompilation detection, not just configuration changes.

```aw wrap
on:
  stale-check: "full"
```

### Frontmatter Hash

A deterministic SHA-256 hash of a workflow's frontmatter configuration, including all imported workflow frontmatter collected in breadth-first order. The hash covers security-relevant fields (`engine`, `on`, `permissions`, `tools`, `network`, `safe-outputs`, etc.) while excluding the markdown body. Identical configurations produce identical hashes across the Go and JavaScript compiler implementations, enabling change detection, tamper verification, and reproducibility checks. To also hash the prompt body, use `on.stale-check: "full"` (see [Body Hash](#body-hash)). See [Frontmatter Hash Specification](/gh-aw/specs/frontmatter-hash-specification/).

### actionlint

A static analysis tool for GitHub Actions workflow files that detects syntax errors, type mismatches, and other issues. Integrated into `gh aw compile` via the `--actionlint` flag. Runs in a Docker container and reports lint findings separately from tooling/integration errors (such as Docker failures or timeouts) that prevent the linter from running. See `--actionlint --zizmor --poutine` in the [Compilation Reference](/gh-aw/reference/compilation-process/).

### poutine

A security linter for GitHub Actions workflows that detects supply-chain vulnerabilities such as unpinned actions and dangerous use of pull request events. Integrated into `gh aw compile` via the `--poutine` flag. Typically used alongside [actionlint](#actionlint) and [zizmor](#zizmor).

### Validation

Checking workflow files for errors, security issues, and best practices. Occurs during compilation and can be enhanced with strict mode and security scanners.

### `gh aw lint`

A CLI command that runs actionlint on existing `.lock.yml` workflow files without recompiling the source Markdown. Unlike `gh aw compile --actionlint`, it reads lock files directly from disk, skipping `zizmor` and `poutine`. Supports `--shellcheck` and `--pyflakes` flags to enable script integrations for shell and Python analysis. Useful for fast local feedback after manual lock-file edits. See [CLI Reference](/gh-aw/setup/cli/).

### zizmor

A security auditing tool for GitHub Actions workflows that identifies vulnerabilities including script injections, excessive permissions, and unsafe use of GitHub context expressions. Integrated into `gh aw compile` via the `--zizmor` flag. Typically used alongside [actionlint](#actionlint) and [poutine](#poutine).

### Deterministic Lineage

The causal graph of edges between workflow runs computed by `gh aw logs --json`. Each edge connects a source run to a target run and captures how one run triggered another — via `workflow_dispatch`, `workflow_call`, or `workflow_run` events — along with a confidence rating and the reasons the link was established. Available under `.edges[]` in the JSON output. Use lineage data to reconstruct orchestrator-to-worker relationships without manually correlating run IDs.

### Episode

A deterministic rollup of related workflow runs that belong to a single logical execution. When an orchestrator dispatches workers, all participating runs are grouped into one episode with aggregate metrics including `total_runs`, `total_tokens`, `total_aic`, and `risky_node_count`. Available under `.episodes[]` in `gh aw logs --json` output. Episodes are more useful than per-run metrics when one logical job spans multiple workflow runs. For cost analysis, prefer `total_aic` (AI Credits).

```bash
gh aw logs --start-date -30d --json | \
  jq '.episodes[] | {id: .episode_id, workflow: .primary_workflow, aic: .total_aic}'
```

### WebAssembly (Wasm)

A compilation target allowing the gh-aw compiler to run in browser environments without server-side Go installation. The compiler is built as a `.wasm` module that packages markdown parsing, frontmatter extraction, import resolution, and YAML generation into a single file loaded with Go's `wasm_exec.js` runtime. Enables interactive playgrounds, editor integrations, and offline workflow compilation tools. See [WebAssembly Compilation](/gh-aw/reference/wasm-compilation/).

## Advanced Features

### Autoloop

A GitHub Next project that builds on GitHub Agentic Workflows to enable continuous, metric-driven optimization. Define a goal, a set of files the agent may modify, and an evaluation command that outputs a numeric metric — Autoloop runs on a schedule, proposes changes, and retains only those that improve the metric. Useful for continuously improving test coverage, bundle size, build times, or custom research objectives. See [Autoloop on GitHub](https://github.com/githubnext/autoloop).

### ARC (Actions Runner Controller)

A Kubernetes operator that manages GitHub Actions self-hosted runners as pods. When combined with the Docker-in-Docker (DinD) sidecar pattern, the runner container and the Docker daemon container have separate `/tmp` filesystems. AWF detects this topology at runtime by inspecting `DOCKER_HOST` and automatically passes `--docker-host-path-prefix` to bridge the split mount paths. No manual configuration is required for `v0.25.43`+ of AWF. See the [AWF sandbox reference](/gh-aw/reference/sandbox/).

### AWF (Agent Workflow Firewall)

The default coding agent sandbox that isolates AI agent execution in a container with network egress control through domain-based access lists. AWF makes the host filesystem and environment variables available inside the container while restricting outbound network access to configured domains. Enabled with `sandbox.agent: awf` (the default when `sandbox` is not specified). Use `sandbox.agent.version` to pin a specific AWF release for reproducible builds. See [Sandbox Configuration](/gh-aw/reference/sandbox/).

### AWF Reflect Route (`/reflect`)

A runtime HTTP endpoint exposed by the AWF API proxy at `http://api-proxy:10000/reflect`. Returns the currently configured inference providers and their model availability for the active run. Use this route in shared workflows or tools that need to discover gateway endpoints, check provider availability, or select a model dynamically at runtime without hardcoding upstream API URLs. The response includes an `endpoints` array (with `provider`, `base_url`, `configured`, and `models` fields) and a `models_fetch_complete` flag. See [AWF Reflect Route](/gh-aw/experimental/awf-reflect/).

### Bridge Pattern

A cross-repository event forwarding architecture for side repository workflows. Because GitHub Actions only delivers webhook events to the repository where they occur, `slash_command:` triggers cannot fire directly in a side repository. The bridge pattern solves this with two workflows: a thin relay workflow in the main repository that receives the slash command and forwards it to the side repository via `workflow_dispatch`, and a worker workflow in the side repository that performs the actual work. See [Triage from Side Repo](/gh-aw/examples/multi-repo/triage-from-side-repo/).

### Cache Memory

Persistent storage for workflows preserving data between runs using GitHub Actions cache. Cache memory is branch-scoped: runs restore from the current branch and may restore from the default branch (`main` in most repositories). After a non-default branch restores from default, later saves remain branch-local. Configured via `cache-memory:` in the tools section with 7-day retention. See [Cache Memory](/gh-aw/reference/cache-memory/).

### Comment Memory (`tools.comment-memory`)

Persistent agent memory backed by a managed GitHub issue or PR comment. Before each agent run, content from `<gh-aw-comment-memory>` blocks in the target comment is extracted into markdown files under `/tmp/gh-aw/comment-memory/`. Agents edit these files using standard file tools; the safe-output handler automatically upserts the managed comment after the run. Unlike [Cache Memory](#cache-memory) (7-day GitHub Actions cache retention) and [Repo Memory](#repo-memory) (permanent git branch storage), comment memory uses the triggering issue or PR as its backing store — no separate infrastructure required. Configured via `tools.comment-memory:` in frontmatter.

### Command Triggers

Special triggers responding to slash commands in issue and PR comments. Configured using the `slash_command:` section with a command name.

### Centralized Slash-Command Strategy (`strategy: centralized`)

An opt-in compilation mode for `slash_command:` workflows where the compiler generates a single shared `agentic_commands.yml` router workflow. The router listens to merged slash-command events and dispatches matching target workflows via `workflow_dispatch` with an `aw_context` payload. Enables combining slash commands with non-slash events (such as `issues` or `pull_request`) without trigger conflicts. Opt in by setting `on.slash_command.strategy: centralized`. See [Command Triggers](/gh-aw/reference/command-triggers/).

### `aw_context`

A structured context payload passed by the centralized slash-command router (`agentic_commands.yml`) when dispatching target workflows via `workflow_dispatch`. Contains the original GitHub event context (issue number, repository, actor, etc.) so the dispatched workflow can act on the correct resource even though it is triggered as a `workflow_dispatch`. See [Command Triggers](/gh-aw/reference/command-triggers/).

### Conclusion Job

An automatically generated job in compiled workflows that handles post-agent reporting and cleanup. Receives a workflow-specific concurrency group (`gh-aw-conclusion-{workflow-name}`) to prevent collision when multiple agent instances run the same workflow concurrently. Requires no manual configuration — the compiler sets the group automatically. See [Concurrency Control](/gh-aw/reference/concurrency/).

### Concurrency Control

Settings limiting how many workflow instances can run simultaneously. Configured via `concurrency:` field to prevent resource conflicts or rate limiting.

### Custom Agents

Specialized instructions customizing AI agent behavior for specific tasks or repositories. Stored as agent files (`.github/agents/*.agent.md`) for Copilot Chat or instruction files (`.github/copilot/instructions/`) for path-specific Copilot instructions.

### Ephemerals

A category of features for automatically expiring workflow resources to reduce repository noise and control costs. Includes workflow stop-after scheduling, safe output expiration (auto-closing issues, discussions, and pull requests), and hidden older status comments. See [Ephemerals](/gh-aw/reference/ephemerals/).

### Environment Variables (env)

Configuration section in frontmatter defining environment variables for the workflow. Variables can reference GitHub context values, workflow inputs, or static values. Accessible via `${{ env.VARIABLE_NAME }}` syntax.

### `GITHUB_AW`

A system-injected environment variable set to `"true"` in every gh-aw engine execution step (both the agent run and the threat-detection run). Agents can check this variable to confirm they are running inside a GitHub Agentic Workflow. Cannot be overridden by user-defined `env:` blocks. See [Environment Variables Reference](/gh-aw/reference/environment-variables/).

### `GH_AW_PHASE`

A system-injected environment variable identifying the active execution phase. Set to `"agent"` during the main agent run and `"detection"` during the threat-detection safety check run that precedes it. Cannot be overridden by user-defined `env:` blocks. See [Environment Variables Reference](/gh-aw/reference/environment-variables/).

### `GH_AW_VERSION`

A system-injected environment variable containing the gh-aw compiler version that generated the workflow (e.g. `"0.40.1"`). Useful for writing conditional logic that depends on a minimum feature version. Cannot be overridden by user-defined `env:` blocks. See [Environment Variables Reference](/gh-aw/reference/environment-variables/).

### `GH_AW_ALLOWED_DOMAINS`

A system-injected environment variable containing the comma-separated list of domains allowed by the workflow's network configuration. Used by safe output jobs for URL sanitization — URLs from unlisted domains are redacted in AI-generated content before it is applied. Automatically populated from `network.allowed` domains and, when `engine.api-target` is set, includes the GHES/GHEC API hostname and base domain. Cannot be overridden by user-defined `env:` blocks. See [Environment Variables Reference](/gh-aw/reference/environment-variables/).

### `GH_AW_DEFAULT_*`

A family of environment variables set in the compiler process environment or as GitHub Actions `vars.*` to apply organization- or repository-wide defaults without editing individual workflow frontmatter. Compiler-process variables (`GH_AW_DEFAULT_MAX_TURNS`, `GH_AW_DEFAULT_TIMEOUT_MINUTES`, `GH_AW_DEFAULT_DETECTION_MODEL`) inject defaults at compile time by being read when `gh aw compile` runs; runtime repository variables (`GH_AW_DEFAULT_MAX_AI_CREDITS`, `GH_AW_DEFAULT_MAX_DAILY_AI_CREDITS`, `GH_AW_DEFAULT_DETECTION_MAX_AI_CREDITS`, `GH_AW_DEFAULT_MODEL_COPILOT`, `GH_AW_DEFAULT_MODEL_CLAUDE`, `GH_AW_DEFAULT_MODEL_CODEX`) are embedded as `${{ vars.* }}` expressions in the compiled workflow and resolved by the GitHub Actions runner at execution time. Frontmatter settings always take precedence over `GH_AW_DEFAULT_*`. Managed in batch via `gh aw env`. See [Compiler Enterprise Environment Controls](/gh-aw/reference/compiler-enterprise-environment-controls/).

### `GH_HOST`

An environment variable recognized by the `gh` CLI that specifies the GitHub hostname for GitHub Enterprise Server (GHES) or GitHub Enterprise Cloud (GHEC) deployments. When set, `gh` commands target the specified enterprise instance instead of `github.com`. Agentic workflows automatically configure this from `GITHUB_SERVER_URL` at agent job startup; the variable is also propagated to custom frontmatter jobs and the safe-outputs job so all `gh` calls target the correct enterprise host. See [Environment Variables Reference](/gh-aw/reference/environment-variables/).

### Label Command Trigger (`label_command`)

A trigger that activates a workflow when a specific label is added to an issue, pull request, or discussion. Unlike standard label filtering, the label command trigger automatically removes the applied label on activation so it can be reapplied to re-trigger the workflow. Configured via `label_command:` in the `on:` section; exposes `needs.activation.outputs.label_command` with the matched label name for downstream jobs. Can be combined with `slash_command:` to support both label-based and comment-based triggering. See [LabelOps patterns](/gh-aw/patterns/label-ops/).

```yaml wrap
on:
  label_command: deploy
```

### Repo Memory

Persistent file storage via Git branches with unlimited retention. Unlike cache-memory (7-day retention), repo-memory stores files permanently in dedicated Git branches with automatic branch cloning, file access, commits, pushes, and merge conflict resolution. Setting `wiki: true` switches the backing to the GitHub Wiki's git endpoint (`{repo}.wiki.git`), and the agent receives guidance to follow GitHub Wiki Markdown conventions (e.g. `[[Page Name]]` links). See [Repo Memory](/gh-aw/reference/repo-memory/).

### Sandbox

Configuration for the AI agent execution environment, providing two isolation layers: the **Coding Agent Sandbox** ([AWF](#awf-agent-workflow-firewall) by default) for network egress control, and the **MCP Gateway** for routing MCP server calls through a unified HTTP endpoint. Configured via the `sandbox:` field in frontmatter. See [Sandbox Configuration](/gh-aw/reference/sandbox/).

### Strict Mode

Enhanced validation mode enforcing additional security checks and best practices. Enabled via `strict: true` in frontmatter or `--strict` flag when compiling.

### Time Scattering

Automatic distribution of workflow execution times across the day to reduce load spikes on GitHub Actions infrastructure. When using fuzzy scheduling, the compiler deterministically assigns different start times to each workflow based on repository and workflow name. Prevents all scheduled workflows from running simultaneously at common times like midnight or the top of the hour.

### Timeout

Maximum duration a workflow can run before automatic cancellation. Configured via `timeout-minutes:` in frontmatter. The agent execution step defaults to 20 minutes; other jobs (custom jobs, safe-output jobs) use the GitHub Actions platform default of 360 minutes unless explicitly set. Custom runners support longer timeouts beyond the GitHub-hosted runner limit.

### Toolsets

Predefined collections of related MCP tools enabled together. Used with the GitHub MCP server to group capabilities like `repos`, `issues`, and `pull_requests`. Configured in the `toolsets:` field.

### Tracker ID

A unique identifier enabling external monitoring and coordination without bidirectional coupling. Orchestrator workflows use tracker IDs to correlate worker runs and discover outputs while workers operate independently.

### Workflow Inputs

Parameters provided when manually triggering a workflow with `workflow_dispatch`. Defined in the `on.workflow_dispatch.inputs` section with type, description, default value, and required status.

## Operational Patterns

Operational patterns (suffixed with "-Ops") are established workflow architectures for common automation scenarios. Each pattern addresses specific use cases with recommended triggers, tools, and safe outputs.

### AgenticOps

Repository-wide observability pattern where a scheduled workflow inspects other agentic workflows, classifies notable behavior, and publishes a structured report. When it detects repeated failures, abnormal token consumption, or other unhealthy patterns, it escalates findings into issues for follow-up. Creates a durable operational record instead of relying on ad hoc inspection of individual runs. See [MonitorOps](/gh-aw/patterns/monitor-ops/).

### BatchOps

Pattern for processing large volumes of work items efficiently using chunked pagination, matrix fan-out, or rate-limit-aware sub-batching. BatchOps splits a backlog into parallel or sequential chunks, handles partial failures with `fail-fast: false`, and aggregates results into a consolidated report. Use when items are independent and order doesn't matter. See [BatchOps](/gh-aw/patterns/batch-ops/).

### Central Control Plane

A [MultiRepoOps](#multirepoops) topology where a single private repository acts as a control plane for coordinating large-scale operations across many repositories. An orchestrator workflow filters and prioritizes targets, then dispatches per-repo worker workflows. Enables phased rollouts, policy updates, and centralized tracking using cross-repository safe outputs and secure authentication. See [MultiRepoOps — Central Control Plane](/gh-aw/patterns/central-repo-ops/#using-a-central-control-repository).

### CorrectionOps

Pattern for improving workflows from trusted human corrections without retraining the underlying model. CorrectionOps stores predictions, compares them with later authoritative human decisions, and uses grouped diffs to update instructions, routing, thresholds, or rollout policy. See [CorrectionOps](/gh-aw/experimental/correction-ops/).

### ChatOps

Interactive automation triggered by slash commands (`/review`, `/deploy`) in issues and pull requests, enabling human-in-the-loop automation where developers invoke AI assistance on demand. See [ChatOps](/gh-aw/patterns/chat-ops/).

### DailyOps

Scheduled workflows for incremental daily improvements, automating progress toward large goals through small, manageable changes on weekday schedules.

### DispatchOps

Manual workflow execution via GitHub Actions UI or CLI using `workflow_dispatch` trigger. Enables on-demand tasks, testing, and workflows requiring human judgment about timing. Workflows can accept custom input parameters. See [DispatchOps](/gh-aw/patterns/dispatch-ops/).

### IssueOps

Automated issue management that analyzes, categorizes, and responds to issues when created. Uses issue event triggers with safe outputs for secure automated triage without requiring write permissions for the AI job. See [IssueOps Examples](/gh-aw/patterns/issue-ops/).

### LabelOps

Workflows triggered by label changes on issues and pull requests. Uses labels as triggers, metadata, and state markers with filtering for specific label additions or removals. See [LabelOps Examples](/gh-aw/patterns/label-ops/).

### MemoryOps

Stateful workflows that persist data between runs using `cache-memory` and `repo-memory`, enabling progress tracking, resumption after interruptions, and incremental processing to avoid API throttling. See [MemoryOps](/gh-aw/patterns/memory-ops/).

### MultiRepoOps

Cross-repository coordination extending automation patterns across multiple repositories. Uses secure authentication and cross-repository safe outputs to synchronize features, centralize tracking, and enforce organization-wide policies. See [MultiRepoOps](/gh-aw/patterns/multi-repo-ops/).

### ProjectOps

AI-powered GitHub Projects board management automating issue triage, routing, and field updates. Analyzes issue/PR content to make intelligent decisions about project assignment, status, priority, and custom fields using the `update-project` safe output. See [ProjectOps](/gh-aw/patterns/project-ops/).

### Side Repository

A [MultiRepoOps](#multirepoops) topology where workflows run from a separate dedicated automation repository targeting your main codebase. Keeps AI-generated issues, comments, and workflow runs isolated from the main repository for cleaner separation between automation infrastructure and production code. See [MultiRepoOps — Side Repository](/gh-aw/patterns/multi-repo-ops/#using-a-side-repository).

### SpecOps

Maintaining and propagating W3C-style specifications using the `w3c-specification-writer` agent. Creates formal specifications with RFC 2119 keywords and automatically synchronizes changes to consuming implementations. See [SpecOps](/gh-aw/patterns/spec-ops/).

### ResearchPlanAssignOps

Scaffolded AI-powered code improvement strategy with four phases: research agent investigates and publishes findings, developer reviews and invokes planner agent to create actionable issues, developer assigns approved issues to Copilot for automated implementation, then reviews and merges PRs. Keeps developers in control with clear decision points at every transition. See [ResearchPlanAssignOps](/gh-aw/patterns/research-plan-assign-ops/).

### TrialOps

Testing and validation pattern executing workflows in isolated trial repositories before production deployment. Creates temporary private repositories where workflows run safely, capturing safe outputs without modifying your actual codebase. See [TrialOps](/gh-aw/experimental/trial-ops/).

### WorkQueueOps

Pattern for incrementally processing a backlog of work items using a durable queue backend — issue checklists, sub-issues, [cache-memory](#cache-memory), or GitHub Discussions. Each run picks up where the last left off, making it resilient to interruptions and rate limits. Items should be idempotent and independently processable. See [WorkQueueOps](/gh-aw/patterns/workqueue-ops/).

## Related Resources

For detailed documentation on specific topics, see:

- [Frontmatter Reference](/gh-aw/reference/frontmatter/)
- [Tools Reference](/gh-aw/reference/tools/)
- [MCP Scripts Reference](/gh-aw/reference/mcp-scripts/)
- [Safe Outputs Reference](/gh-aw/reference/safe-outputs/)
- [Using MCPs Guide](/gh-aw/guides/mcps/)
- [Security Guide](/gh-aw/introduction/architecture/)
- [AI Engines Reference](/gh-aw/reference/engines/)
