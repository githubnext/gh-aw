---
title: Frontmatter
description: Complete guide to all available frontmatter configuration options for GitHub Agentic Workflows, including triggers, permissions, AI engines, and workflow settings.
sidebar:
  order: 200
---

The [frontmatter](/gh-aw/reference/glossary/#frontmatter) (YAML configuration section between `---` markers) of GitHub Agentic Workflows includes the triggers, permissions, AI [engines](/gh-aw/reference/glossary/#engine) (which AI model/provider to use), and workflow settings. For example:

```yaml wrap
---
on:
  issues:
    types: [opened]

tools:
  edit:
  bash: ["gh issue comment"]
---
...markdown instructions...
```

## Frontmatter Elements

Below is a comprehensive reference to all available frontmatter fields for GitHub Agentic Workflows.

### Description (`description:`)

Provides a human-readable description of the workflow rendered as a comment in the generated lock file.

```yaml wrap
description: "Workflow that analyzes pull requests and provides feedback"
```

### Emoji (`emoji:`)

An optional emoji to represent the workflow visually, for example in listings and UI surfaces.

```yaml wrap
emoji: "🤖"
```

### Labels (`labels:`)

Optional array of strings for categorizing workflows by purpose, team, or functionality. Labels appear in `gh aw status` output as `[automation ci diagnostics]` (or a JSON array in `--json` mode) and can be filtered with `gh aw status --label automation`.

```yaml wrap
labels: ["automation", "ci", "diagnostics"]
```

### Metadata (`metadata:`)

Optional key-value pairs for storing custom metadata compatible with the [GitHub Copilot custom agent spec](https://docs.github.com/en/copilot/reference/custom-agents-configuration).

```yaml wrap
metadata:
  author: John Doe
  version: 1.0.0
  category: automation
```

Keys must be 1–64 characters; values are string-only, up to 1024 characters.

### Trigger Events (`on:`)

The `on:` section uses standard GitHub Actions syntax to define workflow triggers, with additional fields for security and approval controls:

- Standard GitHub Actions triggers (push, pull_request, issues, schedule, etc.)
- `reaction:` - Add emoji reactions to triggering items
- `status-comment:` - Post a started/completed comment with a workflow run link (automatically enabled for `slash_command` and `label_command` triggers; must be explicitly set to `true` for other trigger types). Accepts a boolean or an object with optional `issues`, `pull-requests`, and `discussions` toggle fields to selectively disable status comments for specific target types.
- `stop-after:` - Automatically disable triggers after a deadline
- `manual-approval:` - Require manual approval using environment protection rules
- `forks:` - Configure fork filtering for pull_request triggers
- `skip-roles:` - Skip workflow execution for specific repository roles
- `skip-bots:` - Skip workflow execution for specific GitHub actors
- `skip-author-associations:` - Skip execution for configured event + `author_association` combinations
- `roles:` - Restrict which repository roles can trigger the workflow (default: `[admin, maintainer, write]`)
- `bots:` - Allow specific bot accounts to trigger the workflow
- `skip-if-match:` - Skip execution when a search query has matches (supports `scope: none`; use top-level `on.github-token` / `on.github-app` for custom auth)
- `skip-if-no-match:` - Skip execution when a search query has no matches (supports `scope: none`; use top-level `on.github-token` / `on.github-app` for custom auth)
- `steps:` - Inject custom deterministic steps into the pre-activation job (saves one workflow job vs. multi-job pattern)
- `restore-memory:` - Opt in to restoring memory stores before `on.steps` in pre-activation (default: `false`)
- `permissions:` - Grant additional GitHub token scopes to the pre-activation job (for use with `on.steps:` API calls)
- `needs:` - Add custom job dependencies that both `pre_activation` and `activation` must wait for
- `github-token:` - Custom token for activation job reactions, status comments, and skip-if search queries
- `github-app:` - GitHub App for minting a short-lived token used by the activation job and all skip-if search steps

See [Trigger Events](/gh-aw/reference/triggers/) for complete documentation.

### Conditional Execution (`if:`)

Standard GitHub Actions `if:` syntax:

```yaml wrap
if: github.event_name == 'push'
```

### Imports (`imports:`)

Share and reuse workflow components across multiple workflows. The `imports:` field in frontmatter (or `{{#import ...}}` in markdown) composes shared tools, steps, MCP servers, and prompts from other workflow files.

```yaml wrap
imports:
  - shared/common-tools.md
  - shared/mcp/tavily.md
```

See [Imports](/gh-aw/reference/imports/) for complete documentation on syntax, shared components, APM package dependencies, and composition patterns.

### Custom Steps and Jobs (`pre-steps:`, `steps:`, `pre-agent-steps:`, `post-steps:`, `jobs:`)

Add deterministic steps before or after agentic execution, or define full custom GitHub Actions jobs that run before the agent. See [Custom Steps and Jobs](/gh-aw/reference/steps-jobs/) for complete documentation.

### Cache Configuration (`cache:`)

Cache configuration using standard GitHub Actions `actions/cache` syntax:

Single cache:

```yaml wrap
cache:
  key: node-modules-${{ hashFiles('package-lock.json') }}
  path: node_modules
  restore-keys: |
    node-modules-
```

For secure Go-specific cache guidance, see [FAQ: How should I configure Go caches safely in agentic workflows?](/gh-aw/reference/faq/#how-should-i-configure-go-caches-safely-in-agentic-workflows).

### Repository Checkout (`checkout:`)

Configure how `actions/checkout` is invoked in the agent job. Override default checkout settings or check out multiple repositories for cross-repository workflows.

Set `checkout: false` to disable the default repository checkout entirely — useful for workflows that access repositories through MCP servers or other mechanisms that do not require a local clone:

```yaml wrap
checkout: false
```

See [Cross-Repository Operations](/gh-aw/reference/cross-repository/) for complete documentation on checkout configuration options (including `fetch:`, `checkout: false`), merging behavior, and cross-repo examples.

### Permissions (`permissions:`)

The `permissions:` section uses a syntax similar to standard GitHub Actions permissions syntax to specify the GitHub read permissions relevant to the agentic (natural language) part of the execution of the workflow. See [GitHub Tools Read Permissions](/gh-aw/reference/permissions/).

### AI Engine (`engine:`)

Specifies which AI engine interprets the markdown section. See [AI Engines](/gh-aw/reference/engines/) for details.

```yaml wrap
engine: copilot
```

### Engine Driver (`engine.driver:`)

Overrides the built-in engine runtime driver for engines that support driver mode. For Copilot, setting `engine.driver` also enables SDK mode. See [AI Engines](/gh-aw/reference/engines/#copilot-sdk-support) for driver requirements and supported formats.

```yaml wrap
engine:
  id: copilot
  driver: .github/drivers/custom-copilot-driver.js
```

### Network Permissions (`network:`)

Controls network access using ecosystem identifiers and domain allowlists. See [Network Permissions](/gh-aw/reference/network/) for full documentation.

```yaml wrap
network:
  allowed:
    - defaults              # Basic infrastructure
    - python               # Python/PyPI ecosystem
    - "api.example.com"    # Custom domain
```

### Tools (`tools:`)

Specifies which GitHub API calls, bash commands, browser automation, and MCP servers are available to the AI agent.

```yaml wrap
tools:
  edit:
  bash: ["gh issue comment"]
  github:
    toolsets: [default]
```

See [Tools](/gh-aw/reference/tools/) for complete documentation on built-in tools, GitHub toolsets, and MCP server configuration.

### Frontmatter Skills (`skills:`)

Installs external Copilot skills in the activation job before the agent runs.
Each entry must be pinned to a 40-character lowercase commit SHA.

Supported entry formats:

- String form (shared authentication):
  - `owner/repo@<40-char-sha>`
  - `owner/repo/skill/path@<40-char-sha>`
- Object form (per-skill authentication):
  - `skill` (required)
  - `github-token` (optional)
  - `github-app` (optional)

`github-token` and `github-app` are mutually exclusive for each object entry.
`github-token` must be an expression such as `${{ secrets.NAME }}` or
`${{ needs.auth.outputs.token }}`.

```yaml wrap
skills:
  # Shared auth via workflow-level activation token
  - mattpocock/skills/tdd@801dca688564c529fa84f247f64472520d9ebe28

  # Per-skill PAT (or fallback) for private skill repositories
  - skill: mattpocock/skills/diagnosing-bugs@801dca688564c529fa84f247f64472520d9ebe28
    github-token: ${{ secrets.MATT_SKILLS_PAT || secrets.GITHUB_TOKEN }}

  # Per-skill GitHub App credentials
  - skill: mattpocock/skills/domain-modeling@801dca688564c529fa84f247f64472520d9ebe28
    github-app:
      client-id: ${{ vars.MATT_SKILLS_APP_CLIENT_ID }}
      private-key: ${{ secrets.MATT_SKILLS_APP_PRIVATE_KEY }}
```

See [Glossary: Frontmatter Skills](/gh-aw/reference/glossary/#frontmatter-skills-skills)
for terminology, and
[`mattpocock-skills-reviewer.md`](https://github.com/github/gh-aw/blob/main/.github/workflows/mattpocock-skills-reviewer.md)
for a full workflow example using `skills:`.

### MCP Scripts (`mcp-scripts:`)

Enables defining custom MCP tools inline using JavaScript or shell scripts. See [MCP Scripts](/gh-aw/reference/mcp-scripts/) for complete documentation on creating custom tools with controlled secret access.

### Safe Outputs (`safe-outputs:`)

Enables automatic issue creation, comment posting, and other safe outputs. See [Safe Outputs Processing](/gh-aw/reference/safe-outputs/).

### Run Configuration (`run-name:`, `runs-on:`, `runs-on-slim:`, `timeout-minutes:`)

Standard GitHub Actions properties:

```yaml wrap
run-name: "Custom workflow run name"  # Defaults to workflow name
runs-on: ubuntu-latest               # Defaults to ubuntu-latest (main job only)
runs-on-slim: ubuntu-slim            # Defaults to ubuntu-slim (framework jobs only)
timeout-minutes: 30                  # Defaults to 20 minutes
```

`runs-on` applies to the main agent job only. `runs-on-slim` applies to all framework/generated jobs (activation, safe-outputs, unlock, etc.), accepts the same string, array, or runner-group object forms as `runs-on`, and defaults to `ubuntu-slim`. `safe-outputs.runs-on` and `safe-outputs.threat-detection.runs-on` also accept the same runner forms and take precedence where applicable.

`timeout-minutes` accepts an integer or a GitHub Actions expression string (e.g. `${{ inputs.timeout }}`), letting a reusable `workflow_call` workflow parameterize its own timeout from caller inputs. It applies to the workflow being compiled, **not** to plain caller jobs that invoke a reusable workflow with job-level `uses:` — GitHub rejects `timeout-minutes` there.

**Supported runners for `runs-on:`**

| Runner | Status |
|--------|--------|
| `ubuntu-latest` | ✅ Default. Recommended for most workflows. |
| `ubuntu-24.04` / `ubuntu-22.04` | ✅ Supported. |
| `ubuntu-24.04-arm` | ✅ Supported. Linux ARM64 runner. |
| `macos-*` | ❌ Not supported. Docker is unavailable on macOS runners (no nested virtualization). See [FAQ](/gh-aw/reference/faq/). |
| `windows-*` | ❌ Not supported. AWF requires Linux. |

### Workflow Concurrency Control (`concurrency:`)

Automatically generates concurrency policies for the agent job. See [Concurrency Control](/gh-aw/reference/concurrency/).

### Environment Variables (`env:`)

Standard GitHub Actions `env:` syntax for workflow-level environment variables:

```yaml wrap
env:
  CUSTOM_VAR: "value"
```

Environment variables can be defined at multiple scopes (workflow, job, step, engine, safe-outputs, etc.) with clear precedence rules. See [Environment Variables](/gh-aw/reference/environment-variables/) for complete documentation on all 13 env scopes and precedence order.

> [!WARNING]
> Do not use `${{ secrets.* }}` expressions in the workflow-level `env:` section. Environment variables defined here are passed directly to the agent container, which means secret values would be visible to the AI model. In strict mode, this is a compilation error. In non-strict mode, it emits a warning.
>
> Use engine-specific secret configuration instead of the `env:` section to pass secrets securely.

### AI Credits Guardrail (`max-ai-credits:`)

Sets the AWF AI Credits budget used for cost enforcement. It is enabled by default and defaults to `1000` (`1k`) when omitted. Steering (budget-warning messages at 80%, 90%, 95%, and 99% of the budget) is enabled by default. Use plain integers or `K`/`M` suffixes such as `100000K` or `100M`. Set to a negative value to disable both budget enforcement and steering.

```yaml wrap
max-ai-credits: 500
```

```yaml wrap
# Disable budget enforcement and steering
max-ai-credits: -1
```

### Daily Per-Workflow AI Credits Guardrail (`max-daily-ai-credits:`)

Sets a 24-hour AI Credits cap for a single workflow, aggregated across recent runs of the same workflow in the repository. When the activation job detects that the previous 24 hours already exceed this threshold, it warns, creates an issue, skips the agent job, and lets the conclusion job report the specialized failure context. Use plain integers or `K`/`M` suffixes such as `100000K` or `100M`.

This guardrail is disabled by default when omitted, and `-1` explicitly disables it. This guardrail is skipped for `workflow_call`, `repository_dispatch`, and `workflow_dispatch` runs that carry internal `aw_context` dispatch metadata.

```yaml wrap
max-daily-ai-credits: 10000
```

```yaml wrap
# Disable the guardrail explicitly
max-daily-ai-credits: -1
```

### Secrets (`secrets:`)

Defines secret values passed to workflow execution. Secrets are typically used to provide sensitive configuration to MCP servers or workflow components. Values must be GitHub Actions expressions that reference secrets (e.g., `${{ secrets.API_KEY }}`).

```yaml wrap
secrets:
  API_TOKEN: ${{ secrets.API_TOKEN }}
  DATABASE_URL: ${{ secrets.DB_URL }}
```

Secrets can also include descriptions for documentation:

```yaml wrap
secrets:
  API_TOKEN:
    value: ${{ secrets.API_TOKEN }}
    description: "API token for external service"
  DATABASE_URL:
    value: ${{ secrets.DB_URL }}
    description: "Production database connection string"
```

Always reference secrets through `${{ secrets.NAME }}` expressions, never plaintext; prefer environment-specific secrets (via the `environment:` field) and limit access to the components that need them.

**Note:** For passing secrets to reusable workflows, use the `jobs.<job_id>.secrets` field instead. The top-level `secrets:` field is for workflow-level secret configuration.

### Environment Protection (`environment:`)

Specifies the environment for deployment protection rules and environment-specific secrets. Standard GitHub Actions syntax.

```yaml wrap
environment: production
```

See [GitHub Actions environment docs](https://docs.github.com/en/actions/deployment/targeting-different-environments/using-environments-for-deployment).

### Container Configuration (`container:`)

Specifies a container to run job steps in.

```yaml wrap
container: node:18
```

See [GitHub Actions container docs](https://docs.github.com/en/actions/how-tos/write-workflows/choose-where-workflows-run/run-jobs-in-a-container).

### Service Containers (`services:`)

Defines service containers that run alongside your job (databases, caches, etc.).

```yaml wrap
services:
  postgres:
    image: postgres:13
    env:
      POSTGRES_PASSWORD: postgres
    ports:
      - 5432:5432
```

> [!NOTE]
> The AWF agent runs inside an isolated Docker container. Service containers expose ports on the runner host, not within the agent's network namespace. To connect to a service from the agent, use `host.docker.internal` as the hostname instead of `localhost`. For example, a Postgres service configured with port `5432:5432` is accessible at `host.docker.internal:5432`.

See [GitHub Actions service docs](https://docs.github.com/en/actions/using-containerized-services).

### Observability (`observability:`)

Use `observability.otlp` to export distributed traces from workflow runs to an OpenTelemetry Protocol (OTLP) compatible backend.

```yaml wrap
observability:
  otlp:
    endpoint: ${{ secrets.OTLP_ENDPOINT }}
    headers:
      Authorization: ${{ secrets.OTLP_TOKEN }}
      X-Tenant: my-org
```

`endpoint` accepts a string, a `{url, headers}` object, or an array of endpoint objects for fan-out; `headers` accepts a map or comma-separated `key=value` string; `if-missing` supports `error` (default), `warn`, and `ignore`; `attributes` is an optional map of custom span attributes (values support GitHub Actions expressions); and `resource-attributes` appends custom OTel resource attributes to the built-in gh-aw/GitHub set. Use static strings or GitHub Actions expressions for `resource-attributes`, but do not use `secrets.*` or `vars.*` values because resource attributes are exported to external observability backends and are not treated as secret values. See the [OpenTelemetry guide](/gh-aw/guides/open-telemetry/) for setup and the [OpenTelemetry attribute reference](/gh-aw/reference/open-telemetry/) for emitted fields.

### Resources (`resources:`)

Declares additional workflow or action files to fetch alongside this workflow when running `gh aw add`. Use this field when the workflow depends on companion workflows or custom actions stored in the same directory.

```yaml wrap
resources:
  - triage-issue.md          # companion workflow
  - label-issue.md           # companion workflow
  - shared/helper-action.yml # supporting GitHub Action
```

Entries are relative paths from the workflow's location in the source repository (GitHub Actions expression syntax `${{` is not allowed). When `gh aw add` installs this workflow, each listed file is downloaded alongside it so dependencies are available after installation. `gh aw add` also automatically fetches workflows referenced in the [`dispatch-workflow`](/gh-aw/reference/safe-outputs/#workflow-dispatch-dispatch-workflow) safe output, even when not listed here.

### Runtimes (`runtimes:`)

Override default runtime versions for languages and tools used in workflows. The compiler detects which runtimes are needed from tool configurations (e.g. `bash: ["node"]`) and workflow steps, then installs the specified versions. Pin versions for reproducibility, opt into preview releases, or point at custom setup actions (forks, enterprise mirrors).

Each runtime takes a required `version` string, plus optional `action-repo` and `action-version` to override the default setup action:

| Runtime | Default Version | Default Setup Action |
|---------|----------------|---------------------|
| `node` | 24 | `actions/setup-node@v6` |
| `python` | 3.12 | `actions/setup-python@v5` |
| `go` | 1.25 | `actions/setup-go@v5` |
| `uv` | latest | `astral-sh/setup-uv@v5` |
| `bun` | 1.1 | `oven-sh/setup-bun@v2` |
| `deno` | 2.x | `denoland/setup-deno@v2` |
| `ruby` | 3.3 | `ruby/setup-ruby@v1` |
| `java` | 21 | `actions/setup-java@v4` |
| `dotnet` | 8.0 | `actions/setup-dotnet@v4` |
| `elixir` | 1.17 | `erlef/setup-beam@v1` |
| `haskell` | 9.10 | `haskell-actions/setup@v2` |

Override one or more runtimes, optionally with a custom setup action:

```yaml wrap
runtimes:
  node:
    version: "20"
  python:
    version: "3.12"
    action-repo: "actions/setup-python"
    action-version: "v5"
```

Omitted runtimes use the defaults above. Runtimes from imported shared workflows are merged with your workflow's configuration.

### `run-install-scripts`

Controls whether npm pre/post-install scripts are allowed during package installation. Configure this under `runtimes.node.run-install-scripts`. The default is `false`.

```yaml wrap
runtimes:
  node:
    run-install-scripts: true
```

Enabling this increases supply chain risk because install hooks from dependencies can execute arbitrary code. In strict mode, `run-install-scripts: true` is rejected.

### Source Tracking (`source:`)

Tracks workflow origin in format `owner/repo/path@ref`. Automatically populated when using `gh aw add` to install workflows from external repositories. Optional for manually created workflows.

```yaml wrap
source: "githubnext/agentics/workflows/ci-doctor.md@v1.0.0"
```

### Redirect (`redirect:`)

Specifies a new canonical location, using the same `owner/repo/path@ref` format as `source:`, when a workflow has been moved or renamed. `gh aw add`, `gh aw add-wizard`, and `gh aw update` follow redirect chains transitively (up to a depth limit) to the resolved location, rewrite the local `source` field accordingly, and report redirect loops as errors. `gh aw compile` emits an informational message when a `redirect` is configured.

```yaml wrap
redirect: "githubnext/agentics/workflows/new-workflow-name.md@main"
```

Use `gh aw update --no-redirect` to fail the update instead of following the redirect — useful for auditing or controlling exactly when redirects are applied.

> [!NOTE]
> The `redirect` field is set by workflow *authors* to signal that a workflow has moved. It is not typically set by end-users. If you see a redirect when running `gh aw update`, it means the upstream workflow has been relocated.

### Tracker ID (`tracker-id:`)

Tags every asset (issues, pull requests, discussions, comments) the workflow creates with a hidden HTML comment — `<!-- gh-aw-tracker-id: … -->` — enabling GitHub search to find all items associated with this workflow.

```yaml wrap
tracker-id: code-simplifier
```

Accepts 8–128 alphanumeric characters, hyphens, and underscores. Most workflows use their filename as the tracker ID.

Search for all assets created by a specific workflow:

```
repo:owner/repo "gh-aw-tracker-id: code-simplifier" in:body
```

See [Footers](/gh-aw/reference/footers/) for marker details and footer visibility control.

### Private Workflows (`private:`)

Mark a workflow as private to prevent it from being installed into other repositories via `gh aw add`.

```yaml wrap
private: true
```

Adding the workflow from another repository then fails with `workflow 'owner/repo/internal-tooling' is private and cannot be added to other repositories`. Use this for internal tooling, sensitive automation, or repository-specific workflows not intended for reuse.

This only blocks installation via `gh aw add`; the visibility of the workflow file itself is controlled by your repository's access settings.

### `check-for-updates`

Controls whether the compile-agentic version update check runs in the activation job.

```yaml wrap
check-for-updates: true
```

When `true` (default), the activation job verifies the compiled version is not blocked and meets the minimum supported version. Set to `false` to disable this check (not allowed in strict mode).

### Feature Flags (`features:`)

Enable experimental or optional compiler and runtime behaviors as key-value pairs. See [Feature Flags](/gh-aw/reference/feature-flags/) for complete documentation.

### Strict Mode (`strict:`)

Disables enhanced security validation for production workflows.

```yaml wrap
strict: false  # Disable for development/testing
```

Workflows compiled with `strict: false` cannot run on public repositories. The workflow fails at runtime with an error message prompting recompilation with strict mode.

See [Network Permissions - Strict Mode Validation](/gh-aw/reference/network/#strict-mode-validation) for details on network validation and [CLI Commands](/gh-aw/setup/cli/#compile) for compilation options.

## Related Documentation

See also: [Trigger Events](/gh-aw/reference/triggers/), [AI Engines](/gh-aw/reference/engines/), [CLI Commands](/gh-aw/setup/cli/), [Workflow Structure](/gh-aw/reference/workflow-structure/), [Network Permissions](/gh-aw/reference/network/), [Feature Flags](/gh-aw/reference/feature-flags/), [Custom Steps and Jobs](/gh-aw/reference/steps-jobs/), [OpenTelemetry Guide](/gh-aw/guides/open-telemetry/), [Command Triggers](/gh-aw/reference/command-triggers/), [MCPs](/gh-aw/guides/mcps/), [Tools](/gh-aw/reference/tools/), [Imports](/gh-aw/reference/imports/)
