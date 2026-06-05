---
description: Agentic workflow specific frontmatter fields for GitHub Agentic Workflows.
---

# Agentic Workflow Frontmatter Fields

### Agentic Workflow Specific Fields

- **`description:`** - Human-readable workflow description (string)
- **`emoji:`** - Optional single emoji used to represent the workflow visually; recommended for quicker recognition in workflow lists and status output (string)
- **`source:`** - Workflow origin tracking in format `owner/repo/path@ref` (string)
- **`labels:`** - Array of labels to categorize and organize workflows (array)
  - Labels filter workflows in status/list commands
  - Example: `labels: [automation, security, daily]`
- **`metadata:`** - Custom key-value pairs compatible with custom agent spec (object)
  - Key names limited to 64 characters
  - Values limited to 1024 characters
  - Example: `metadata: { team: "platform", priority: "high" }`
- **`github-token:`** - Default GitHub token for workflow (must use `${{ secrets.* }}` syntax)
- **`on.roles:`** - Repository access roles that can trigger workflow (array or "all")
  - Default: `[admin, maintainer, write]`
  - Available roles: `admin`, `maintainer`, `write`, `read`, `all`
- **`bots:`** - Bot identifiers allowed to trigger workflow regardless of role permissions (array)
  - Example: `bots: [dependabot[bot], renovate[bot], github-actions[bot]]`
  - Bot must be active (installed) on repository to trigger workflow
- **`strict:`** - Enable enhanced validation for production workflows (boolean, defaults to `true`)
  - Must be `true`
- **`max-runs:`** - Maximum number of LLM invocations allowed per workflow run (integer or numeric string, minimum: 1)
  - Top-level field mapped to `apiProxy.maxRuns`
  - Supported by all engines
- **`max-turns:`** - AWF turn cap applied consistently across all agentic engines (integer or expression, e.g. `${{ inputs.max-turns }}`). The engine-level `engine.max-turns` is a deprecated alias kept for backward compatibility — prefer this top-level field. Not supported by the `gemini` engine.
- **`max-effective-tokens:`** - Per-run effective-token (ET) budget enforced by the AWF firewall (integer or `K`/`M` short-form string like `100M`; default `25000000`). Set a negative value to disable enforcement and token steering. See [token-optimization.md](token-optimization.md).
- **`max-daily-ai-credits:`** - Per-user 24-hour ET guardrail: activation blocks execution once the triggering user's aggregated ET for this workflow over the last 24h exceeds the threshold (integer or `K`/`M` short-form string, or `-1`). Enabled by default with a system default threshold; set `-1` to disable or an explicit value to override. See [token-optimization.md](token-optimization.md).
- **`user-rate-limit:`** - Rate limiting configuration to prevent users from triggering the workflow too frequently (object)
  - **`max-runs-per-window:`** - Maximum runs allowed per user per time window (required, integer 1-10)
  - **`window:`** - Time window in minutes (integer 1-180, default: 60)
  - **`events:`** - Event types to apply rate limiting to (array; if omitted, applies to all programmatic events)
    - Available: `workflow_dispatch`, `issue_comment`, `pull_request_review`, `pull_request_review_comment`, `issues`, `pull_request`, `discussion_comment`, `discussion`
  - **`ignored-roles:`** - Roles exempt from rate limiting (array, default: `[admin, maintain, write]`). Set to `[]` to apply to all users.
  - Example:

    ```yaml
    user-rate-limit:
      max-runs-per-window: 5
      window: 60
      ignored-roles: [admin, maintain]
    ```

- **`check-for-updates:`** - Control whether the activation job checks if the compiled `gh-aw` version is still supported (boolean, default: `true`)
  - When `true` (default): blocked versions fail fast; below-recommended versions emit a warning
  - When `false`: skips the version check; the compiler emits a warning at compile time
  - Use `check-for-updates: false` only when deploying in isolated environments where version update checks are not feasible

- **`features:`** - Feature flags for experimental or optional features (object)
  - Each flag is a key-value pair; boolean flags (`true`/`false`) or string values are accepted
  - Known feature flags:
    - `copilot-requests: true` - Use GitHub Actions token for Copilot authentication instead of `COPILOT_GITHUB_TOKEN` secret
    - `disable-xpia-prompt: true` - Disable the built-in cross-prompt injection attack (XPIA) system prompt
    - `action-tag: "v0"` - Pin compiled action references to a specific version of the `gh-aw-actions` repository. Accepts version tags (e.g., `"v0"`, `"v1"`, `"v1.0.0"`) or a full 40-character commit SHA. When set, overrides the compiler's default action mode and resolves all action references from the external `github/gh-aw-actions` repository at the specified tag.
    - `action-mode: "script"` - Control how the compiler generates action references: `"dev"` (local paths, default), `"release"` (SHA-pinned remote), `"action"` (gh-aw-actions repo), `"script"` (direct shell calls). Can also be overridden via `--action-mode` CLI flag.
    - `difc-proxy: true` - Enable DIFC (Data Integrity and Flow Control) proxy injection. When set alongside `tools.github.min-integrity`, injects proxy steps around the agent for full network-boundary integrity enforcement.
    - `cli-proxy: true` - Enable AWF CLI proxy sidecar for secure read-only `gh` CLI access without exposing `GITHUB_TOKEN` (requires AWF v0.26.0+). Prerequisite for `integrity-reactions`; the compiler enables it automatically when `integrity-reactions: true` is set.
    - `integrity-reactions: true` - Enable reaction-based integrity promotion/demotion. Maintainers can use 👍/❤️ reactions to promote content to `approved` and 👎/😕 to demote it to `none`. Compiler automatically enables `cli-proxy`. Requires `tools.github.min-integrity` to be set and MCPG >= v0.2.18. Defaults: endorsement reactions THUMBS_UP/HEART, disapproval reactions THUMBS_DOWN/CONFUSED, endorser-min-integrity: approved, disapproval-integrity: none.
    - `mcp-cli: true` - Deprecated. This flag has been removed; MCP CLI mounting is now always enabled when `tools.cli-proxy: true` is set.

- **`experiments:`** - A/B testing experiments for balanced variant selection (object)
  - Maps experiment names to variant lists (bare array) or full config objects
  - Bare array form: `prompt_style: [concise, detailed]` — round-robin balanced across runs
  - Object form for weighted/gated experiments:

    ```yaml
    experiments:
      prompt_style:
        variants: [concise, detailed, step_by_step]
        weight: [2, 1, 1]           # Optional: proportional weights (defaults to round-robin)
        start_date: "2026-05-01"    # Optional: ISO-8601; returns control variant before this date
        end_date: "2026-06-01"      # Optional: ISO-8601; returns control variant after this date
        description: "Verbosity test"  # Optional: experiment description
        metric: "token_count"       # Optional: primary metric name
        issue: "42"                 # Optional: linked tracking issue number
    ```

  - Selected variant available as `${{ experiments.<name> }}` and in `{{#if experiments.<name> }}` template blocks
  - See [A/B Testing Experiments](experiments.md) for full design guidance

- **`imports:`** - Array of workflow specifications to import (array)
  - Format: `owner/repo/path@ref` or local paths like `shared/common.md`
  - Markdown files under `.github/agents/` are treated as custom agent files
  - Only one agent file is allowed per workflow
  - See [Imports Field](#imports-field) section for detailed documentation
- **`inlined-imports:`** - Inline all imports at compile time (boolean, default: `false`)
  - When `true`, all imports (including those without inputs) are inlined in the generated `.lock.yml` instead of using runtime-import macros
  - The frontmatter hash covers the entire markdown body when enabled, so any content change invalidates the hash
  - **Required for repository rulesets**: Workflows used as required status checks in repository rulesets run without access to repository files at runtime. Set `inlined-imports: true` to bundle all imported content at compile time to avoid "Runtime import file not found" errors
  - **Constraint**: Cannot be combined with agent file imports (`.github/agents/` files). Remove any custom agent file imports before enabling
- **`import-schema:`** - Define typed input parameters for this shared workflow (object). Use when other workflows import this one via the `uses:`/`with:` syntax (see [Imports Field](#imports-field)).
  - Parameters are accessible inside the shared workflow via `${{ github.aw.import-inputs.<name> }}` expressions
  - Object inputs (type: `object`) allow one-level deep sub-fields: `${{ github.aw.import-inputs.<name>.<subkey> }}`
  - Fields per parameter:
    - `type:` - Input type: `string`, `number`, `boolean`, `choice`, or `array`
    - `description:` - Human-readable parameter description
    - `required:` - Whether the input is required when imported (default: `false`)
    - `default:` - Default value when not provided
    - `options:` - Allowed values for `choice` type inputs
  - Example:

    ```yaml
    import-schema:
      environment:
        type: choice
        description: "Target environment"
        options: [dev, staging, prod]
        required: true
      max-issues:
        type: number
        default: 5
    ```

- **`mcp-servers:`** - MCP (Model Context Protocol) server definitions (object)
  - Defines custom MCP servers for additional tools beyond built-in ones

- **`private:`** - Mark this workflow as private, preventing it from being shared via `gh aw add` (boolean, default: `false`)
  - Example: `private: true`

- **`redirect:`** - Workflow relocation path for updates (string). When present, `gh aw update` follows this location and rewrites the `source:` field. Format: `owner/repo/path@ref` or full GitHub URL.
  - Example: `redirect: "org/agentics/workflows/my-workflow-v2.md@main"`

- **`resources:`** - Additional workflow or action files fetched alongside this workflow when running `gh aw add` (array). Entries are relative paths from the same directory to `.md` or `.yml`/`.yaml` files.
  - Example: `resources: [shared/tool-setup.md, shared/mcp/tavily.md]`

- **`tracker-id:`** - Optional identifier to tag all created assets (string)
  - Must be at least 8 characters and contain only alphanumeric characters, hyphens, and underscores
  - This identifier is inserted in the body/description of all created assets (issues, discussions, comments, pull requests)
  - Enables searching and retrieving assets associated with this workflow
  - Examples: `"workflow-2024-q1"`, `"team-alpha-bot"`, `"security_audit_v2"`

- **`secret-masking:`** - Configuration for secret redaction behavior in workflow outputs and artifacts (object)
  - `steps:` - Additional secret redaction steps to inject after the built-in secret redaction (array)
  - Use this to mask secrets in generated files using custom patterns
  - Example:

    ```yaml
    secret-masking:
      steps:
        - name: Redact custom secrets
          run: find /tmp/gh-aw -type f -exec sed -i 's/password123/REDACTED/g' {} +
    ```

- **`observability:`** - Workflow observability and telemetry configuration (object)
  - **`otlp:`** - Export OpenTelemetry spans to any OTLP-compatible backend (Honeycomb, Grafana Tempo, Sentry, etc.) (object)
    - `endpoint:` - OTLP collector endpoint URL. When a static URL is provided, its hostname is added to the AWF firewall allowlist automatically. Supports GitHub Actions expressions.
    - `github-app:` - Optional runtime auth configuration.
      - Preferred: provide GitHub App credentials (`app-id`/`client-id` + `private-key`) to mint a token with `actions/create-github-app-token` before `actions/setup`.
      - OIDC mode is used when `github-app` is configured without credentials (`app-id`/`client-id` + `private-key`).
      - OIDC mode requires `permissions.id-token: write` on the workflow/job.
    - `headers:` - Comma-separated `key=value` HTTP headers included in every OTLP export request (e.g. `Authorization=Bearer <token>`). Injected as `OTEL_EXPORTER_OTLP_HEADERS`. Supports GitHub Actions expressions.
  - Example:

    ```yaml
    observability:
      otlp:
        endpoint: ${{ secrets.GH_AW_OTEL_ENDPOINT }}
        github-app:
          app-id: ${{ vars.APP_ID }}
          private-key: ${{ secrets.APP_PRIVATE_KEY }}
        headers: ${{ secrets.GH_AW_OTEL_HEADERS }}
    ```

    Every job emits setup and conclusion spans with rich attributes (`gh-aw.job.name`, `gh-aw.workflow.name`, `gh-aw.engine.id`, token usage). All jobs in a run share one trace ID. Dispatched child workflows inherit the parent's trace context via `aw_context`.

- **`runtimes:`** - Runtime environment version overrides (object)
  - Allows customizing runtime versions (e.g., Node.js, Python) or defining new runtimes
  - Runtimes from imported shared workflows are also merged
  - Each runtime is identified by a runtime ID (e.g., 'node', 'python', 'go')
  - Runtime configuration properties:
    - `version:` - Runtime version as string or number (e.g., '22', '3.12', 'latest', 22, 3.12)
    - `action-repo:` - GitHub Actions repository for setup (e.g., 'actions/setup-node')
    - `action-version:` - Version of the setup action (e.g., 'v4', 'v5')
    - `if:` - Optional GitHub Actions condition to control when runtime setup runs (e.g., `"hashFiles('go.mod') != ''"`)
  - Example:

    ```yaml
    runtimes:
      node:
        version: "22"
      python:
        version: "3.12"
        action-repo: "actions/setup-python"
        action-version: "v5"
      go:
        version: "1.22"
        if: "hashFiles('go.mod') != ''"   # Only install Go when go.mod exists
    ```

- **`runtimes.node.run-install-scripts:`** - Allow npm pre/post install scripts to execute during package installation for the Node.js runtime (boolean, default: `false`)
  - By default, `--ignore-scripts` is added to all generated npm install commands to prevent supply chain attacks via malicious install hooks
  - Set `run-install-scripts: true` under `runtimes.node` to allow scripts for Node.js installs
  - A supply chain security warning is emitted at compile time; in strict mode this is an error

- **`checkout:`** - Override how the repository is checked out in the agent job (object, array, or `false`)
  - By default, the workflow automatically checks out the repository. Use this field to customize checkout behavior.
  - Set to `false` to disable automatic checkout entirely (reduces startup time when repo access is not needed):

    ```yaml
    checkout: false
    ```

  - Single checkout (object):

    ```yaml
    checkout:
      fetch-depth: 0              # Fetch full history (default: 1 = shallow clone)
      github-token: ${{ secrets.MY_PAT }}  # Override token for private repos
    ```

  - Multiple checkouts (array):

    ```yaml
    checkout:
      - path: .
        fetch-depth: 0
      - repository: owner/other-repo
        path: ./libs/other
        ref: main
    ```

  - Supported fields per checkout entry:
    - `repository:` - Repository in `owner/repo` format (defaults to current repository)
    - `ref:` - Branch, tag, or SHA to check out (defaults to triggering ref)
    - `path:` - Relative path within `GITHUB_WORKSPACE` (defaults to workspace root)
    - `fetch-depth:` - Number of commits to fetch; `0` = full history, `1` = shallow (default)
    - `fetch:` - Additional Git refs to fetch after checkout (array of patterns)
      - `"*"` - fetch all remote branches
      - `"refs/pulls/open/*"` - all open pull-request refs
      - Branch names, glob patterns (e.g., `"feature/*"`)
      - Example: `fetch: ["*"]`, `fetch: ["refs/pulls/open/*"]`
    - `sparse-checkout:` - Newline-separated glob patterns for sparse checkout
    - `submodules:` - Submodule handling: `"recursive"`, `"true"`, or `"false"`
    - `lfs:` - Download Git LFS objects (boolean, default: `false`)
    - `wiki:` - Check out the repository's wiki (boolean, default: `false`). When `true`, automatically appends `.wiki` to the repository name. Combine with `repository:` to check out a different repo's wiki.
    - `github-token:` - Token for authentication (`${{ secrets.MY_PAT }}`); credentials removed after checkout

- **`jobs:`** - Groups together all the jobs that run in the workflow (object)
  - Standard GitHub Actions jobs configuration
  - Each job can have: `name`, `runs-on`, `steps`, `needs`, `if`, `env`, `permissions`, `timeout-minutes`, etc.
  - For most agentic workflows, jobs are auto-generated; only specify this for advanced multi-job workflows
  - **Security Notice**: Custom jobs run OUTSIDE the firewall sandbox. Execute with standard GitHub Actions security but NO network egress controls. Use only for deterministic preprocessing, data fetching, or static analysis—not agentic compute or untrusted AI execution.
  - **`pre-steps:`** - Steps injected after compiler-generated setup and before any `steps:` in a custom or built-in job (array). For built-in jobs (`activation`, `pre_activation`), injected after the `id: setup` step and before the first checkout. Imported `pre-steps` run before main workflow `pre-steps`.
  - Example:

    ```yaml
    jobs:
      custom-job:
        runs-on: ubuntu-latest
        pre-steps:
          - name: Pre-flight setup
            run: echo "runs before checkout"
        steps:
          - name: Custom step
            run: echo "Custom job"
    ```

- **`engine:`** - AI processor configuration
  - String format: `"copilot"` (default, recommended), `"claude"`, `"codex"`, `"gemini"`, or `"opencode"` (experimental)
  - Object format for extended configuration:

    ```yaml
    engine:
      id: copilot                       # Required: coding agent identifier (copilot, claude, codex, gemini, or opencode)
      version: beta                     # Optional: version of the action (has sensible default); also accepts GitHub Actions expressions: ${{ inputs.engine-version }}
      model: gpt-5                      # Optional: LLM model to use (has sensible default)
      agent: technical-doc-writer       # Optional: custom agent file (Copilot only, references .github/agents/{agent}.agent.md)
      max-turns: 5                      # Deprecated alias for the top-level `max-turns`; prefer the top-level field
      max-continuations: 3              # Optional: max autopilot continuations (copilot only; >1 enables --autopilot mode, default: 1)
      concurrency: "gh-aw-${{ github.workflow }}"  # Optional: agent job concurrency group (string or GitHub Actions concurrency object)
      env:                              # Optional: custom environment variables (object)
        DEBUG_MODE: "true"
      args: ["--verbose"]               # Optional: custom CLI arguments injected before prompt (array)
      api-target: api.acme.ghe.com      # Optional: custom API endpoint hostname for GHEC/GHES (hostname only, no protocol/path)
      command: /usr/local/bin/copilot   # Optional: override default engine executable (skips installation)
      bare: true                        # Optional: disable automatic context loading (copilot: --no-custom-instructions; claude: --bare; codex: --no-system-prompt; gemini: GEMINI_SYSTEM_MD=/dev/null). Default: false
      user-agent: "myapp/1.0"           # Optional: custom user agent string (codex engine only)
      config: |                         # Optional: additional TOML config appended to config.toml (codex engine only)
        [extra]
        key = "value"
      token-weights:                    # Optional: custom token cost weights for effective token computation
        multipliers:
          my-custom-model: 2.5          # 2.5x the cost of claude-sonnet-4.5 (= 1.0)
        token-class-weights:
          output: 6.0                   # Override output token weight (default: 4.0)
          cached-input: 0.05            # Override cached input weight (default: 0.1)
    ```

  - **Note**: The `version` and `model` fields have sensible defaults and can typically be omitted unless you need specific customization. For turn caps, prefer the top-level `max-turns` field over the deprecated engine-level alias shown above.
  - **`gemini` engine**: Google Gemini CLI. Requires `GEMINI_API_KEY` secret. Does not support `max-turns`, `web-fetch`, or `web-search`. Supports AWF firewall and LLM gateway.
  - **`opencode` engine** (experimental): Provider-agnostic, open-source AI coding agent (BYOK). Defaults to Copilot routing via `COPILOT_GITHUB_TOKEN` (or `${{ github.token }}` with `copilot-requests` feature). Supports 75+ models via `provider/model` format. Supports AWF firewall and LLM gateway.

- **`network:`** - Network access control for AI engines (top-level field)
  - String format: `"defaults"` (curated allow-list of development domains)
  - Empty object format: `{}` (no network access)
  - Object format for custom permissions:

    ```yaml
    network:
      allowed:
        - "example.com"
        - "*.trusted-domain.com"
        - "https://api.secure.com"        # Optional: protocol-specific filtering
      blocked:
        - "blocked-domain.com"
        - "*.untrusted.com"
        - python                          # Block ecosystem identifiers
      firewall: true                      # Optional: Enable AWF (Agent Workflow Firewall) for Copilot engine
    ```

  - **Firewall configuration** (Copilot engine only):

    ```yaml
    network:
      firewall:
        version: "v1.0.0"                 # Optional: AWF version (defaults to latest)
        log-level: debug                  # Optional: debug, info (default), warn, error
        args: ["--custom-arg", "value"]   # Optional: additional AWF arguments
    ```

- **`sandbox:`** - Sandbox configuration for AI engines (string or object)
  - String format: `"default"` (default sandbox), `"awf"` (Agent Workflow Firewall)
  - Object format to pin an AWF version (strict mode requires explicit `id: awf`):

    ```yaml
    sandbox:
      agent:
        id: awf                     # Required in strict mode
        version: "v0.25.29"         # Optional: pin AWF version
        model-fallback: false       # Optional: disable model fallback (default true); set false for BYOK Azure OpenAI to prevent deployment-name rewriting
    ```

  - To disable the agent firewall while keeping MCP gateway enabled (not allowed in strict mode):

    ```yaml
    sandbox:
      agent: false
    ```

  - **Strict mode**: `sandbox.agent` blocks without an explicit `id: awf` are rejected in strict mode. Any non-nil, non-disabled agent config without `id`/`type` defaults to AWF at runtime.

- **`tools:`** - Tool configuration for coding agent
  - `github:` - GitHub API tools
    - `allowed:` - Array of allowed GitHub API functions. Each entry is either a string tool name (e.g., `issue_read`) or an object `{ name: <tool>, max-calls: <n> }` to cap how many times that tool may be called per run. Colon shorthand (`"issue_read:1"`) is **not** a call-limit form.

      ```yaml
      tools:
        github:
          allowed:
            - { name: issue_read, max-calls: 1 }
            - list_labels
            - pull_request_read
      ```
    - `mode:` - GitHub access mode. **Prefer `"gh-proxy"`** — it is faster (no MCP server startup) and lets the agent use `gh` shell commands directly for all GitHub reads (issues, PRs, discussions, commits, etc.):
      - `"gh-proxy"` (**preferred**) — pre-authenticated `gh` CLI available in bash; no GitHub MCP server is registered. Use `gh` commands for all GitHub reads.
      - `"local"` (default) — Docker-based GitHub MCP Server; use GitHub MCP tools for reads, `gh` is not authenticated.
      - **do NOT use `"remote"`** — it does not work with the GitHub Actions token; use `"gh-proxy"` instead.
    - `version:` - MCP server version (local mode only)
    - `args:` - Additional command-line arguments (local mode only)
    - `read-only:` - The GitHub MCP server always operates in read-only mode; this field is accepted but has no effect
    - `github-token:` - Custom GitHub token
    - `lockdown:` - Enable lockdown mode to limit content surfaced from public repositories to items authored by users with push access (boolean, default: false)
    - `github-app:` - GitHub App configuration for token minting; when set, mints an installation access token at workflow start that overrides `github-token`
      - `client-id:` - GitHub App client ID (required, e.g., `${{ vars.APP_ID }}`). Use `app-id:` for legacy compatibility.
      - `private-key:` - GitHub App private key (required, e.g., `${{ secrets.APP_PRIVATE_KEY }}`)
      - `owner:` - Optional installation owner (defaults to current repository owner)
      - `repositories:` - Optional list of repositories to grant access to (array)
      - `permissions:` - Optional extra permissions to include in the minted token for org-level API access (object)
        - Example: `permissions: { members: read, organization-administration: read }` — required when calling org-level APIs (e.g., `orgs`, `users` toolsets) since the default GITHUB_TOKEN does not have org-scoped permissions
    - `min-integrity:` - Minimum integrity level for MCP Gateway guard policy; controls which content the agent may act on based on author trust (`none`, `unapproved`, `approved`, `merged`)
    - `blocked-users:` - Usernames whose content is unconditionally blocked (array or GitHub Actions expression); these users receive integrity below `none` and are always denied
    - `approval-labels:` - Label names that elevate a content item's integrity to `approved` when present (array or GitHub Actions expression); does not override `blocked-users`
    - `trusted-users:` - Usernames elevated to `approved` integrity regardless of `author_association` (array or GitHub Actions expression); useful for contractors who need elevated access without becoming repo collaborators; takes precedence over `min-integrity` but not over `blocked-users`; requires `min-integrity` to be set
    - `toolsets:` - Enable specific GitHub toolset groups (array only)
      - **Default toolsets** (when unspecified): `context`, `repos`, `issues`, `pull_requests` (excludes `users` as GitHub Actions tokens don't support user operations)
      - **All toolsets**: `context`, `repos`, `issues`, `pull_requests`, `actions`, `code_security`, `dependabot`, `discussions`, `experiments`, `gists`, `labels`, `notifications`, `orgs`, `projects`, `secret_protection`, `security_advisories`, `stargazers`, `users`, `search`
      - Use `[default]` for recommended toolsets, `[all]` to enable everything
      - Examples: `toolsets: [default]`, `toolsets: [default, discussions]`, `toolsets: [repos, issues]`
      - **Recommended**: Prefer `toolsets:` over `allowed:` for better organization and reduced configuration verbosity
  - `agentic-workflows:` - GitHub Agentic Workflows MCP server for workflow introspection
    - Provides tools for:
      - `status` - Show status of workflow files in the repository
      - `compile` - Compile markdown workflows to YAML
      - `logs` - Download and analyze workflow run logs
      - `audit` - Investigate workflow run failures and generate reports
      - `checks` - Classify CI check state for a pull request (returns normalized verdict: `success`, `failed`, `pending`, `no_checks`, `policy_blocked`)
    - **Use case**: Enable AI agents to analyze GitHub Actions traces and improve workflows based on execution history
    - **Example**: Configure with `agentic-workflows: true` or `agentic-workflows:` (no additional configuration needed)
  - `edit:` - File editing tools (required to write to files in the repository)
  - `web-fetch:` - Web content fetching tools
  - `web-search:` - Web search tools
  - `bash:` - Shell command tools
    - **Bash allowlist decision rule:**
      - **PR-triggered workflows** processing **untrusted input** (issue/PR body, comment text, user-provided filenames): use a **narrow allowlist** (for example: `[find, cat, grep, wc, jq]`). This limits blast radius if shell injection attempts are embedded in untrusted content.
      - **`schedule` or `workflow_dispatch` workflows** with **no untrusted input** (only trusted API data or internal state): `["*"]` is acceptable.
      - **Rule of thumb**: If the workflow reads issue/PR bodies, comment text, or other user-provided strings, use a narrow list. If it only reads trusted API responses or workflow artifacts, `["*"]` is acceptable.
    - **Examples:**

      ```yaml
      # PR-triggered workflow reading untrusted user text
      on:
        pull_request:
      tools:
        bash: [find, cat, grep, wc, jq]

      # Internal scheduled workflow reading only trusted/internal data
      on:
        schedule:
          - cron: "0 * * * *"
      tools:
        bash: ["*"]
      ```
  - `playwright:` - Browser automation tools for visual regression, accessibility testing, and end-to-end testing. Use `mode: cli` (recommended) — no Docker, runs `playwright-cli <command>` in bash, `localhost` reaches local servers directly. `mode: mcp` is deprecated (Docker-based; requires bridge IP detection for local server access). Pin a specific version with `version:` and restrict network access to `local` + `playwright` for security. See [`visual-regression-checker.md`](../../.github/workflows/visual-regression-checker.md) for a minimal pull-request example.

    ```yaml
    tools:
      playwright:
        mode: cli          # recommended: token-efficient CLI mode
        version: "0.1.11"  # optional: @playwright/cli npm package version
    ```
  - Custom tool names for MCP servers
  - `timeout:` - Per-operation timeout in seconds for all tool and MCP server calls (integer or GitHub Actions expression). Defaults vary by engine (Claude: 60 s, Codex: 120 s).
  - `startup-timeout:` - Timeout in seconds for MCP server initialization (integer or GitHub Actions expression, default: 120). Useful in `workflow_call` reusable workflows: `startup-timeout: ${{ inputs.startup-timeout }}`
  - `cli-proxy:` - Mount each user-facing MCP server as a standalone CLI tool on `PATH` (boolean, default: `false`). When enabled, the agent can call MCP servers via shell commands (e.g. `github issue_read --method get ...`). CLI-mounted servers remain in the MCP gateway so their containers start normally.


- **`safe-outputs:`** - Safe output processing configuration. See [safe-outputs.md](safe-outputs.md) for complete documentation of all output types: `create-issue`, `create-discussion`, `add-comment`, `create-pull-request`, `push-to-pull-request-branch`, `close-issue`, `close-discussion`, `update-issue`, `update-pull-request`, `add-labels`, `remove-labels`, `dispatch-workflow`, `call-workflow`, `create-code-scanning-alert`, `upload-asset`, `upload-artifact`, `assign-to-agent`, `assign-to-user`, and more.

  **Key safe-outputs global fields:**
  - `github-token:` — custom token for all safe-output jobs
  - `github-app:` — GitHub App credentials for minting tokens
  - `staged:` — preview mode (no API calls)
  - `footer:` — global footer control (boolean)
  - `threat-detection:` — auto-enabled threat detection
  - `runs-on:` — runner for safe-output jobs (default: `ubuntu-slim`)
  - `messages:` — custom footer/notification message templates
  - `env:` — environment variables for safe-output jobs
  - `max-patch-size:` — maximum git patch size in KB (default: 1024)


- **`mcp-scripts:`** - Define custom lightweight MCP tools as JavaScript, shell, Python, or Go scripts (object)
  - Tools mounted in MCP server with access to specified secrets
  - Each tool requires `description` and one of: `script` (JavaScript), `run` (shell), `py` (Python), or `go` (Go)
  - Tool configuration properties:
    - `description:` - Tool description (required)
    - `inputs:` - Input parameters with type and description (object)
    - `script:` - JavaScript implementation (CommonJS format)
    - `run:` - Shell script implementation
    - `py:` - Python script implementation
    - `go:` - Go script implementation (executed via `go run`, receives inputs as JSON via stdin)
    - `env:` - Environment variables for secrets (supports `${{ secrets.* }}`)
    - `timeout:` - Execution timeout in seconds (default: 60)
  - Example:

    ```yaml
    mcp-scripts:
      search-issues:
        description: "Search GitHub issues using API"
        inputs:
          query:
            type: string
            description: "Search query"
            required: true
          limit:
            type: number
            description: "Max results"
            default: 10
        script: |
          const { Octokit } = require('@octokit/rest');
          const octokit = new Octokit({ auth: process.env.GH_TOKEN });
          const result = await octokit.search.issuesAndPullRequests({
            q: inputs.query,
            per_page: inputs.limit
          });
          return result.data.items;
        env:
          GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}
    ```

- **`slash_command:`** - Command trigger configuration for /mention workflows (under `on:`)
- **`cache:`** - Cache configuration for workflow dependencies (object or array)
- **`cache-memory:`** - Memory MCP server with persistent cache storage (boolean or object, under `tools:`)
- **`repo-memory:`** - Repository-specific memory storage (boolean, under `tools:`)
- **`comment-memory:`** - Managed issue/PR comment memory with file-based agent editing (boolean or object, under `tools:`)
