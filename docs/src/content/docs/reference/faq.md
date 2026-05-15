---
title: Frequently Asked Questions
description: Answers to common questions about GitHub Agentic Workflows, including security, costs, privacy, and configuration.
sidebar:
  order: 50
---

> [!NOTE]
> GitHub Agentic Workflows is in early development and may change significantly. Using automated agentic workflows requires careful attention to security considerations and careful human supervision, and even then things can still go wrong. Use it with caution, and at your own risk.

## Determinism

### I like deterministic CI/CD. Isn't this non-deterministic?

Agentic workflows are **100% additive** to your existing CI/CD - they don't replace your deterministic build, test, or release pipelines. Think of it as **Continuous AI** alongside Continuous Integration and Continuous Deployment: a new automation layer running in GitHub Actions where security, permissions, and repository context already exist.

Your deterministic pipelines stay unchanged. Agentic workflows handle tasks where exact reproducibility doesn't matter - triaging issues, drafting documentation, researching dependencies, or proposing code improvements for human review.

## Capabilities

### What's the difference between agentic workflows and regular GitHub Actions workflows?

Agentic workflows use AI to interpret natural language instructions in markdown instead of complex YAML. The AI engine can call pre-approved tools to perform tasks while running with read-only default permissions, safe outputs, and sandboxed execution.

### What's the difference between agentic workflows and just running a coding agent in GitHub Actions?

While you could install and run a coding agent directly in a standard GitHub Actions workflow, agentic workflows provide a structured framework with simpler markdown format, built-in security controls, pre-defined tools for GitHub operations, and easy switching between AI engines.

### Can agentic workflows write code and create pull requests?

Yes! Agentic workflows can create pull requests using the `create-pull-request` safe output. This allows the workflow to propose code changes, documentation updates, or other modifications as pull requests for human review and merging.

Some organizations may completely disable the creation of pull requests from GitHub Actions. In such cases, workflows can still generate diffs or suggestions in issues or comments for manual application.

### Can agentic workflows do more than code?

Yes! Agentic workflows can analyze repositories, generate reports, triage issues, research information, create documentation, and coordinate work. The AI interprets natural language instructions and uses available [tools](/gh-aw/reference/tools/) to accomplish tasks.

### Can agentic workflows mix regular GitHub Actions steps with AI agentic steps?

Yes! Agentic workflows can include both AI agentic steps and traditional GitHub Actions steps. You can add custom steps before the agentic job using the [`steps:` configuration](/gh-aw/reference/frontmatter/#custom-steps-steps). Additionally, [custom safe output jobs](/gh-aw/reference/safe-outputs/#custom-safe-output-jobs-jobs) can be used as consumers of agentic outputs. [MCP Scripts](/gh-aw/reference/mcp-scripts/) allow you to pass data between traditional steps and the AI agent with added checking.

### Can agentic workflows read other repositories?

Not by default, but yes with proper configuration. Cross-repository access requires:

1. A **Personal Access Token (PAT)** with access to target repositories
2. Configuring the token in your workflow

See [MultiRepoOps](/gh-aw/patterns/multi-repo-ops/) for coordinating across repositories, or [SideRepoOps](/gh-aw/patterns/side-repo-ops/) for running workflows from a separate repository.

### Can I use agentic workflows in private repositories?

Yes, and in many cases we recommend it. Private repositories are ideal for proprietary code, creating a "sidecar" repository with limited access, testing workflows, and organization-internal automation. See [SideRepoOps](/gh-aw/patterns/side-repo-ops/) for patterns using private repositories.

### Can I edit workflows directly on GitHub.com without recompiling?

Yes! The **markdown body** (AI instructions) is loaded at runtime and can be edited directly on GitHub.com or in any editor. Changes take effect on the next workflow run without recompilation.

However, **frontmatter configuration** (tools, permissions, triggers, network rules) is embedded in the compiled workflow and requires recompilation when changed. Run `gh aw compile my-workflow` after editing frontmatter.

See [Editing Workflows](/gh-aw/guides/editing-workflows/) for complete guidance on when recompilation is needed.

### Can workflows trigger other workflows?

Yes, using the `dispatch-workflow` safe output:

```yaml wrap
safe-outputs:
  dispatch-workflow:
    max: 1
```

This allows your workflow to trigger up to 1 other workflows with custom inputs. See [Safe Outputs](/gh-aw/reference/safe-outputs/#workflow-dispatch-dispatch-workflow) for details.

### Can I trigger an agentic workflow from an external system like Jira?

Yes. GitHub Actions cannot listen to external events directly, but any external system that can make an HTTP request can trigger a workflow via the [`repository_dispatch`](https://docs.github.com/en/actions/writing-workflows/choosing-when-your-workflow-runs/events-that-trigger-workflows#repository_dispatch) API.

The two-step setup:

**1. Add a `repository_dispatch` trigger to your workflow frontmatter:**

```yaml wrap
on:
  repository_dispatch:
    types: [jira-issue-created]
```

Access the caller's payload in your workflow markdown via `${{ github.event.client_payload.* }}`.

**2. Send a `POST` request to the GitHub dispatch API from the external system:**

```http
POST https://api.github.com/repos/<owner>/<repo>/dispatches
Authorization: Bearer <token>
Content-Type: application/json

{
  "event_type": "jira-issue-created",
  "client_payload": { "issue_key": "PROJ-123", "summary": "Fix the thing" }
}
```

For Jira specifically, use **Project → Automation → Issue created → Send web request** pointing at the dispatch API. Any system with webhook or outbound HTTP support—including Jira, PagerDuty, Slack, or a custom API—can trigger workflows this way.

The `repository_dispatch` token must have `repo` scope (classic PAT) or `contents: write` permission. Store it in the external system's secret or credential store (e.g., Jira Automation secret text, a CI/CD vault), scoped to the single target repository.

See [Repository Dispatch Trigger](/gh-aw/reference/triggers/#repository-dispatch-trigger-repository_dispatch) for the full trigger reference.

### Can I use MCP servers with agentic workflows?

Yes! [Model Context Protocol (MCP)](/gh-aw/reference/glossary/#mcp-model-context-protocol) servers extend workflow capabilities with custom tools and integrations. Configure them in your frontmatter:

```yaml wrap
tools:
  mcp-servers:
    my-server:
      image: "ghcr.io/org/my-mcp-server:latest"
      network:
        allowed: ["api.example.com"]
```

See [Getting Started with MCP](/gh-aw/guides/getting-started-mcp/) and [MCP Servers](/gh-aw/guides/mcps/) for configuration guides.

### The `plugins:` field I was using is gone - how do I install agent plugins now?

The `plugins:` frontmatter field has been removed in favour of the `dependencies:` field backed by [Microsoft APM (Agent Package Manager)](https://microsoft.github.io/apm/). APM provides cross-agent support for all agent primitives – skills, prompts, instructions, hooks, and plugins (including the Copilot `plugin.json` format and the Claude `plugin.json` format).

Run `gh aw fix --write` to automatically migrate your existing `plugins:` fields to `dependencies:`.

Use the `dependencies:` field in your workflow frontmatter to install plugins:

```yaml wrap
# Simple list (public or same-org packages)
dependencies:
  - github/my-copilot-plugin
  - github/awesome-copilot/plugins/context-engineering
```

For cross-org private packages, use `github-app:` authentication:

```yaml wrap
dependencies:
  github-app:
    client-id: ${{ vars.APP_ID }}
    private-key: ${{ secrets.APP_PRIVATE_KEY }}
  packages:
    - acme-org/acme-plugins
```

The `dependencies:` approach works with all supported engines (Copilot, Claude, Codex, Gemini, Crush), whereas the old `plugins:` field was limited to the Copilot engine only.

See [APM Dependencies](/gh-aw/reference/dependencies/) for full configuration options.

### Can I use Claude plugins with APM dependencies?

Yes! APM supports Claude plugins in the `plugin.json` format. When `engine: claude` is set, APM automatically infers the engine target and unpacks only Claude-compatible primitives. Use `#tag` or `#branch` suffixes to pin specific versions:

```yaml wrap
engine: claude

dependencies:
  - owner/repo/plugins/my-plugin#v2.0    # pinned to a tag
  - owner/repo/plugins/my-plugin#main    # pinned to a branch
```

For private cross-org plugins and other configuration options, see [APM Dependencies](/gh-aw/reference/dependencies/).

### Can workflows be broken up into shareable components?

Workflows can import shared configurations and components:

```yaml wrap
imports:
  - shared/github-tools.md
  - githubnext/agentics/shared/common-tools.md
```

This enables reusable tool configurations, network settings, and permissions across workflows. See [Imports](/gh-aw/reference/imports/) and [Packaging Imports](/gh-aw/guides/packaging-imports/) for details.

### Can I run workflows on a schedule?

Yes, use fuzzy schedule expressions in the `on:` trigger (recommended):

```yaml wrap
on: weekly on monday  # Automatically scattered to avoid load spikes
```

Or use standard cron syntax for fixed times:

```yaml wrap
on:
  schedule:
    - cron: "0 9 * * MON"  # Every Monday at 9am UTC
```

See [Schedule Syntax](/gh-aw/reference/schedule-syntax/) for all supported formats.

### Can I run workflows conditionally?

Yes, use the `if:` expression at the workflow level:

```yaml wrap
if: github.event_name == 'push' && github.ref == 'refs/heads/main'
```

See [Conditional Execution](/gh-aw/reference/frontmatter/#conditional-execution-if) in the Frontmatter Reference for details.

## Guardrails

### Agentic workflows run in GitHub Actions. Can they access my repository secrets?

Repository secrets are not available to the agentic step by default. The AI agent runs with read-only permissions and cannot directly access your repository secrets unless explicitly configured. You should review workflows carefully, follow [GitHub Actions security guidelines](https://docs.github.com/en/actions/reference/security/secure-use), use least-privilege permissions, and inspect the compiled `.lock.yml` file. See the [Security Architecture](/gh-aw/introduction/architecture/) for details.

Some MCP tools may be configured using secrets, but these are only accessible to the specific tool steps, not the AI agent itself. Minimize the use of tools equipped with highly privileged secrets.

### Agentic workflows run in GitHub Actions. Can they write to the repository?

By default, the agentic "coding agent" step of agentic workflows runs with read-only permissions. Write operations require explicit approval through [safe outputs](/gh-aw/reference/safe-outputs/) or explicit general `write` permissions (not recommended). This ensures that AI agents cannot make arbitrary changes to your repository.

If safe outputs are configured, the workflow has limited, highly specific write operations that are then sanitized and executed securely.

### What sanitization is done on AI outputs before applying changes?

All safe outputs from the AI agent are sanitized before being applied to your repository. Sanitization includes secret redaction, URL domain filtering, XML escaping, size limits, control character stripping, GitHub reference escaping and HTTPS enforcement.

Additionally, safe outputs enforce permission separation - write operations happen in separate jobs with scoped permissions, never in the agentic job itself.

See [Safe Outputs - Text Sanitization](/gh-aw/reference/safe-outputs/#text-sanitization-allowed-domains-allowed-github-references) for configuration options.

### How do I prevent workflow output from creating backlinks in referenced issues?

When AI-generated content mentions issue or PR numbers (such as `#123` or `owner/repo#456`), GitHub automatically creates "mentioned in..." timeline entries in those issues. Set `allowed-github-references: []` to escape all such references before the content is posted:

```yaml wrap
safe-outputs:
  allowed-github-references: []  # Escape all GitHub references
  create-issue:
```

With an empty list, every `#N` and `owner/repo#N` reference in the output is wrapped in backticks, which prevents GitHub from resolving them as cross-references and avoids cluttering other repositories' timelines. This is especially useful for [SideRepoOps](/gh-aw/patterns/side-repo-ops/) workflows that write content about issues in a main repository from a separate sidecar repository.

To allow references only from the current repository while still escaping all others:

```yaml wrap
safe-outputs:
  allowed-github-references: [repo]
  add-comment:
```

When `allowed-github-references` is not configured at all, all references are left unescaped (default behavior).

See [Text Sanitization](/gh-aw/reference/safe-outputs/#text-sanitization-allowed-domains-allowed-github-references) for full configuration options.

### How are agent actions constrained — commenting, opening PRs, modifying files, and calling external tools?

gh-aw uses defense-in-depth rather than a single control. Four layers work together:

**1. Read-only agent by default.** The AI agent step has read-only GitHub permissions. It cannot comment, open PRs, or push files unless you explicitly configure [safe outputs](/gh-aw/reference/safe-outputs/).

**2. Safe outputs for all writes.** Commenting, creating PRs, and modifying files all go through safe outputs — separate GitHub Actions jobs with scoped write tokens. The agent produces a structured artifact; a downstream job applies the changes after sanitization (secret redaction, URL filtering, size limits). You declare which operations are permitted:

```yaml wrap
safe-outputs:
  add-comment:
```

**3. Threat detection before writes.** [Agentic threat detection](/gh-aw/reference/threat-detection/) runs automatically between the agent job and the safe output jobs. It scans the agent's output for prompt injection attempts, secret leaks, and malicious code patches, blocking the write jobs if a threat is detected.

**4. Network allowlist for external calls.** The [Agent Workflow Firewall](/gh-aw/reference/sandbox/) blocks all outbound network access by default. You must explicitly allow each domain an agent may reach:

```yaml wrap
network:
  allowed:
    - defaults
```

For sensitive operations, you can layer on a [GitHub Environment protection rule](/gh-aw/reference/faq/#can-i-require-external-human-approval-before-safe-outputs-are-applied) so a designated reviewer must approve before any write jobs run.

### Tell me more about guardrails

Guardrails are foundational to the design. Agentic workflows implement defense-in-depth through compilation-time validation (schema checks, expression safety, action SHA pinning), runtime isolation (sandboxed containers with network controls), permission separation (read-only defaults with [safe outputs](/gh-aw/reference/safe-outputs/) for writes), tool allowlisting, and output sanitization. See the [Security Architecture](/gh-aw/introduction/architecture/).

### Can I require external human approval before safe outputs are applied?

Yes. The distinction here is between *guardrail validation* (does the agent output look acceptable?) and *external admission* (is this execution intent authorized to proceed?). gh-aw addresses both.

The safe outputs architecture already enforces permission separation: the agent job runs read-only and never holds write credentials; it only produces a structured artifact. Separate jobs, with scoped write tokens, apply the changes. This boundary is real — a compromised agent cannot directly write to GitHub.

For a fail-closed **external admission gate** before sensitive operations like deployments or credential use, apply **[GitHub Environment protection rules](https://docs.github.com/en/actions/managing-workflow-runs-and-deployments/managing-deployments/managing-environments-for-deployment#required-reviewers)** to a [custom safe output job](/gh-aw/reference/custom-safe-outputs/). The job pauses until a designated reviewer outside the workflow system explicitly approves. No approval means no execution.

```yaml wrap
jobs:
  approval-gate:
    runs-on: ubuntu-latest
    needs: detection          # waits for automated threat scanning to complete
    environment: production-deploy   # configure required reviewers in Settings → Environments
    steps:
      - name: Approved
        run: echo "Execution approved by reviewer"

safe-outputs:
  needs: [approval-gate]      # built-in safe_outputs job waits for manual approval
```

This approval is enforced by GitHub's infrastructure, not by workflow logic the agent could influence. Threat detection still runs before the gate, so the reviewer sees output that has already passed automated scanning.

Note that the *policy* — which environments require approval, what safe outputs are configured — is defined by whoever controls the repository. The admission decision for each run can be external; the admission policy itself is internal to repository owners.

**Fully off-platform admission control**

If your threat model requires an authority completely outside GitHub's control plane — such as an external policy engine, a PAM/PIM system, or a compliance approval workflow — call that system from your gate job before it proceeds:

```yaml wrap
jobs:
  external-admission:
    runs-on: ubuntu-latest
    needs: [agent, detection]        # waits for agent output and threat scanning to complete
    environment: production-deploy   # optional: also adds GitHub-native reviewer gate
    steps:
      - name: Request admission from external authority
        run: |
          curl --fail -X POST https://YOUR_POLICY_ENGINE/v1/admit \
            -H "Authorization: Bearer $POLICY_TOKEN" \
            -d '{"workflow_run": "${{ github.run_id }}"}'
        env:
          POLICY_TOKEN: ${{ secrets.POLICY_TOKEN }}

safe-outputs:
  needs: [external-admission]   # write jobs don't run until external admission is granted
```

If the external call fails or is denied, the safe output jobs never run. This places the final admission decision in a system entirely independent of GitHub.

### How is my code and data processed?

By default, your workflow is run on GitHub Actions, like any other GitHub Actions workflow, and as one if its jobs it invokes your nominated [AI Engine (coding agent)](/gh-aw/reference/engines/), run in a container. This engine may in turn make tool calls and MCP calls. When using the default **GitHub Copilot CLI**, the workflow is processed by the `copilot` CLI tool which uses GitHub Copilot's services and related AI models. The specifics depend on your engine choice:

- **GitHub Copilot CLI**: See [GitHub Copilot documentation](https://docs.github.com/en/copilot) for details.
- **Claude/Codex**: Uses respective providers' APIs with their data handling policies.

See the [Security Architecture](/gh-aw/introduction/architecture/) for details on the execution and data flow.

### Does the underlying AI engine run in a sandbox?

Yes, the [AI engine](/gh-aw/reference/engines/) runs in a containerized sandbox with network egress control via the [Agent Workflow Firewall](/gh-aw/reference/sandbox/), container isolation, GitHub Actions resource constraints, and limited filesystem access to workspace and temporary directories. The sandbox container runs inside a GitHub Actions VM for additional isolation. See [Sandbox Configuration](/gh-aw/reference/sandbox/).

### Can an agentic workflow use outbound network requests?

Yes, but network access is restricted by the [Agent Workflow Firewall](/gh-aw/reference/sandbox/). You must explicitly declare which domains the workflow can access:

```yaml wrap
network:
  allowed:
    - defaults             # Basic infrastructure
    - python               # Python/PyPI ecosystem
    - "api.example.com"    # Custom domain
```

See [Network Permissions](/gh-aw/reference/network/) for complete configuration options.

### How does integrity filtering protect my workflow?

[Integrity filtering](/gh-aw/reference/integrity/) controls which GitHub content the agent can see, filtering by **author trust** and **merge status**. The MCP gateway silently removes content below the configured `min-integrity` threshold before the AI engine sees it.

For **public repositories**, `min-integrity: approved` is automatically applied at runtime — restricting content to owners, members, and collaborators — even without additional authentication.

For triage or spam-detection workflows that need to process content from all users, set `min-integrity: none` explicitly:

```yaml wrap
tools:
  github:
    min-integrity: none
```

See [Integrity Filtering](/gh-aw/reference/integrity/) for available levels, user blocking, and approval labels.

## Configuration & Setup

### Why do slash-command workflows show many "started then skipped" runs on comments?

This is expected behavior. A `slash_command` is compiled into multiple GitHub event listeners (issue/PR bodies, issue comments, PR comments, and review comments, depending on `events:`). GitHub first dispatches the event, then the activation logic checks whether the comment starts with a matching command (for example `/refresh`). If it does not match, the run exits early and appears as a quick skipped/no-op run in Actions.

To reduce this noise, narrow the trigger scope with `events:` so the workflow only listens where you actually use commands, and use [LabelOps](/gh-aw/patterns/label-ops/) for command-style operations that should not activate on every comment. LabelOps (`label_command`) triggers only when a specific label is applied, which produces fewer incidental runs than broad comment listeners.

```yaml wrap
on:
  slash_command:
    name: refresh
    events: [pull_request_comment]   # only listen to PR comments
  label_command:
    name: refresh
    events: [pull_request]           # optional low-noise label trigger
```

### What is a workflow lock file?

A **workflow lock file** (`.lock.yml`) is the compiled GitHub Actions workflow generated from your `.md` file by `gh aw compile`. It contains SHA-pinned actions, resolved imports, configured permissions, and all guardrail hardening - inspect it to see exactly what will run, with no hidden configuration.

Both files should be committed to version control:

- **`.md` file**: Your source - edit the prompt body freely; changes take effect at the next run without recompiling
- **`.lock.yml` file**: The compiled workflow GitHub Actions actually runs; must be regenerated after any frontmatter changes (permissions, tools, triggers)

### What is the actions-lock.json file?

The `.github/aw/actions-lock.json` file is a cache of resolved `action@version` → ref mappings. During compilation, the compiler **tries** to pin each action reference to an immutable commit SHA for security. Resolving a version tag to a SHA requires querying the GitHub API (scanning releases), which can fail when the available token has limited permissions — for example, when compiling via GitHub Copilot Coding Agent (CCA) where the token may not have access to external repositories. In those cases, the compiler may fall back to leaving a stable version tag ref (such as `@v0`) instead of a SHA.

The cache avoids this problem: if a ref (typically a SHA) was previously resolved (using a user PAT or a GitHub Actions token with broader access), the result is stored in `actions-lock.json` and reused on subsequent compilations, regardless of the current token's capabilities. Without this cache, compilation is unstable — it succeeds with a permissive token but fails when token access is restricted.

Commit `actions-lock.json` to version control so that all contributors and automated tools (including CCA) use consistent action refs (SHAs or version tags) without needing to re-resolve them. Refresh the cache periodically with `gh aw update-actions`, or delete it and recompile to force a full re-resolution when you have an appropriate token. See [Action Pinning](/gh-aw/reference/compilation-process/#action-pinning) for details.

### What is `github/gh-aw-actions`?

`github/gh-aw-actions` is the GitHub Actions repository containing all reusable actions that power compiled agentic workflows. Compiled `.lock.yml` files reference these actions as `github/gh-aw-actions/setup@<ref>` (where `<ref>` is usually a commit SHA, but may be a stable version tag such as `v0`). These references are managed entirely by `gh aw compile` — never edit them manually. See [The gh-aw-actions Repository](/gh-aw/reference/compilation-process/#the-gh-aw-actions-repository) for details.

### Why is Dependabot opening PRs to update `github/gh-aw-actions`?

Dependabot scans `.lock.yml` files for action references and treats `github/gh-aw-actions` pins as regular dependencies to update. **Do not merge these PRs.** Action pins in compiled workflows should only be updated by running `gh aw compile` or `gh aw update-actions`.

Suppress these PRs by adding an `ignore` entry in `.github/dependabot.yml`:

```yaml
updates:
  - package-ecosystem: github-actions
    directory: "/.github/workflows"
    ignore:
      - dependency-name: "github/gh-aw-actions/**" # Managed by gh aw compile. Version-locked to the gh-aw compiler; do not bump.
```

See [Dependabot and gh-aw-actions](/gh-aw/reference/compilation-process/#dependabot-and-gh-aw-actions) for more details.

### How does `gh aw upgrade` resolve action versions when no GitHub Releases exist?

`gh aw upgrade` (and `gh aw update-actions`) resolves the latest version of each referenced action using a two-step process:

1. **GitHub Releases API** — queries `/repos/{owner}/{repo}/releases` via the `gh` CLI. If releases are found, the highest compatible semantic version is selected.
2. **Git tag fallback** — if the Releases API returns an empty list (which happens when a repository publishes tags without creating GitHub Releases), the command automatically falls back to scanning tags via `git ls-remote`. This fallback is **safe to ignore** — tags are a valid source for version pinning.

Only if *both* sources return no results does the upgrade produce a warning that cannot be resolved automatically.

> **Note:** `github/gh-aw-actions` intentionally publishes only tags (not GitHub Releases). The `gh aw upgrade` warning `github/gh-aw-actions/setup: no releases found` that appeared in earlier versions was caused by this two-step logic not falling back to tags. It has been fixed — the tag fallback now runs automatically.

### Why do I need a token or key?

When using **GitHub Copilot CLI**, a Personal Access Token (PAT) with "Copilot Requests" permission authenticates and associates automation work with your GitHub account. This ensures usage tracking against your subscription, appropriate AI permissions, and auditable actions. In the future, this may support organization-level association. See [Authentication](/gh-aw/reference/auth/).

### Can I use `CLAUDE_CODE_OAUTH_TOKEN` with the Claude engine?

No. `CLAUDE_CODE_OAUTH_TOKEN` is not supported by GitHub Agentic Workflows. The only supported authentication method for the Claude engine is [`ANTHROPIC_API_KEY`](/gh-aw/reference/auth/#anthropic_api_key), which must be configured as a GitHub Actions secret. Provider-based OAuth authentication for Claude (such as billing through a Claude Teams subscription) is not supported. See [Authentication](/gh-aw/reference/auth/) and [AI Engines](/gh-aw/reference/engines/#available-coding-agents) for setup instructions.

### What hidden runtime dependencies does this have?

The executing agentic workflow uses your nominated coding agent (defaulting to GitHub Copilot CLI), a GitHub Actions VM with NodeJS, pinned Actions from [github/gh-aw](https://github.com/github/gh-aw) releases, and an Agent Workflow Firewall container for network control (optional but default). The exact YAML workflow can be inspected in the compiled `.lock.yml` file - there's no hidden configuration.

### Why are macOS runners not supported?

macOS runners (`macos-*`) are not currently supported in agentic workflows. Agentic workflows rely on containers to build a secure execution sandbox - specifically the [Agent Workflow Firewall](/gh-aw/reference/sandbox/) that provides network egress control and process isolation. GitHub-hosted macOS runners do not support container jobs, which is a hard requirement for this security architecture.

Use `ubuntu-latest` (the default) or another Linux-based runner instead. For tasks that genuinely require macOS-specific tooling, consider running those steps in a regular GitHub Actions job that coordinates with your agentic workflow.

### Can I use agentic workflows on GitHub Enterprise Server (GHES)?

Yes, but you may need to enable GHES compatibility mode to avoid artifact errors. GHES instances that predate `@actions/artifact` v2.0.0 support cannot run `actions/upload-artifact@v4+` or `actions/download-artifact@v4+`. On those instances, compiled workflows fail with a `GHESNotSupportedError` because the compiler emits v4+ artifact actions by default.

Enable GHES compatibility mode so the compiler emits `upload-artifact@v3.2.2` and `download-artifact@v3.1.0` instead:

**`aw.json` (recommended — applies to all workflows in the repository):**

```json
{
  "ghes": true
}
```

**`--ghes` flag (one-off compilation):**

```bash
gh aw compile --ghes my-workflow.md
```

Running `gh aw init` inside a GHES repository automatically detects the deployment and writes `ghes: true` to `.github/workflows/aw.json` for you. For `gh` CLI host setup and Copilot prerequisites on GHES, see [Enterprise Configuration](/gh-aw/reference/enterprise-configuration/).

### I'm not using a supported AI Engine (coding agent). What should I do?

If you want to use a coding agent that isn't currently supported (Copilot, Claude, Codex, Gemini, or Crush), you can contribute support to the [gh-aw repository](https://github.com/github/gh-aw), or open an issue describing your use case. See [AI Engines](/gh-aw/reference/engines/).

### Can I test workflows without affecting my repository?

Yes! Use [TrialOps](/gh-aw/experimental/trial-ops/) to test workflows in isolated trial repositories. This lets you validate behavior and iterate on prompts without creating real issues, PRs, or comments in your actual repository.

### Where can I find help with common issues?

See [Common Issues](/gh-aw/troubleshooting/common-issues/) for detailed troubleshooting guidance including workflow failures, debugging strategies, permission issues, and network problems.

### Why is my create-discussion workflow failing?

Ensure discussions are enabled (**Settings → Features → Discussions**) and the workflow has `discussions: write` permission. For category matching failures, verify spelling (case-insensitive) and use lowercase slugs (e.g., `general`, `announcements`) rather than display names.

Use `fallback-to-issue: true` (the default) to automatically create an issue if discussions aren't available. See [Discussion Creation](/gh-aw/reference/safe-outputs/#discussion-creation-create-discussion) for details.

### How do I turn off discussions in add-comment?

By default, `add-comment` requests `discussions: write` permission. If your GitHub App lacks the Discussions permission (which can cause 422 errors during token generation), set `discussions: false`:

```yaml wrap
safe-outputs:
  add-comment:
    discussions: false   # exclude discussions:write permission
```

This removes the `discussions: write` permission requirement. Discussion targeting itself remains automatic — `discussions: false` only controls the permission scope, not which events trigger the workflow.

Similarly, you can opt out of `issues: write` or `pull-requests: write` using `issues: false` or `pull-requests: false`.

### Why is my create-pull-request workflow failing with "GitHub Actions is not permitted to create or approve pull requests"?

Some organizations block PR creation by GitHub Actions via **Settings → Actions → General → Workflow permissions**. If you can't enable it, use one of these alternatives:

**Automatic issue fallback (default)** — `fallback-as-issue: true` is the default; when PR creation is blocked an issue with the branch link is created instead. Requires `contents: write`, `pull-requests: write`, and `issues: write`.

**Assign to Copilot** — create an issue assigned to `copilot` for automated implementation:

```yaml wrap
safe-outputs:
  create-issue:
    assignees: [copilot]
    labels: [automation, enhancement]
```

**Disable fallback** — set `fallback-as-issue: false` to skip the issue fallback and only attempt PR creation. Requires only `contents: write` and `pull-requests: write`, but the workflow will fail if PR creation is blocked.

See [Pull Request Creation](/gh-aw/reference/safe-outputs/#pull-request-creation-create-pull-request) for details.

### Why don't pull requests created by agentic workflows trigger my CI checks?

This is expected GitHub Actions security behavior. Pull requests created using the default `GITHUB_TOKEN` or by the GitHub Actions bot user **do not trigger workflow runs** on `pull_request`, `pull_request_target`, or `push` events. This is a [GitHub Actions security feature](https://docs.github.com/en/actions/security-for-github-actions/security-guides/automatic-token-authentication#using-the-github_token-in-a-workflow) designed to prevent accidental recursive workflow execution.

The easy way to fix this problem is to set a secret `GH_AW_CI_TRIGGER_TOKEN` with a Personal Access Token (PAT) with 'Contents: Read & Write' permission to your repo.

See [Triggering CI](/gh-aw/reference/triggering-ci/) for more details on how to configure workflows to run CI checks on PRs created by agentic workflows.

### How do I suppress the "Generated by..." text in workflow outputs?

When workflows create or update issues, pull requests, discussions, or post comments, they append a `> Generated by [Workflow Name](run_url) for issue #N` attribution line. Use `footer: false` to hide this visible text while preserving the hidden XML markers used for search and tracking.

**Hide footers globally** (all safe output types):

```yaml wrap
safe-outputs:
  footer: false
  add-comment:
  create-issue:
    title-prefix: "[ai] "
```

**Hide footers for specific output types only:**

```yaml wrap
safe-outputs:
  footer: false            # hide for all by default
  create-pull-request:
    footer: true           # override: show footer for PRs only
```

Even with `footer: false`, the hidden `<!-- gh-aw-workflow-id: ... -->` XML marker is still included in the content for searchability - you can search GitHub for `"gh-aw-workflow-id: my-workflow" in:body` to find all items created by a workflow.

See [Footer Control](/gh-aw/reference/footers/) for complete documentation including per-handler overrides and PR review footer options.

### My workflow fails with "Runtime import file not found" when used in a repository ruleset

This happens because workflows configured as required status checks run in a restricted context without access to the repository file system, so runtime imports cannot be resolved.

The fix is to enable `inlined-imports: true` in your workflow frontmatter so the compiler bundles all imported content into the compiled `.lock.yml` at compile time. See [Self-Contained Lock Files](/gh-aw/reference/imports/#self-contained-lock-files-inlined-imports-true) for the full details.

### My cross-organization `workflow_call` fails with a repository checkout error

When a trigger file in one organization calls an agentic workflow in a **different organization**, the activation job attempts to check out the platform repo's `.github` folder using the caller's `GITHUB_TOKEN`. That token is scoped to the caller's organization and cannot access a private repository in another organization, producing an error such as:

```
fatal: repository 'https://github.com/other-org/platform-repo/' not found
```

The fix is to enable `inlined-imports: true` on the **platform workflow** (the callee). This embeds all imported content into the compiled `.lock.yml` at compile time, eliminating the cross-organization checkout entirely:

```yaml
---
on:
  workflow_call:
engine: copilot
inlined-imports: true
imports:
  - shared/common-tools.md
---
```

See [Cross-Organization `workflow_call`](/gh-aw/reference/imports/#cross-organization-workflow_call) for the full details.

### My workflow checkout is very slow because my repository is a large monorepo. How can I speed it up?

Use **sparse checkout** to only fetch the parts of the repository that your workflow actually needs. This can reduce checkout time from tens of minutes to seconds for large monorepos.

Configure `sparse-checkout` in your workflow frontmatter using the `checkout:` field:

```yaml wrap
checkout:
  sparse-checkout: |
    node/my-package
    .github
```

This generates a checkout step that only downloads the specified paths, dramatically reducing clone size and time.

For cases where you need multiple parts of a monorepo with different settings, you can combine checkouts:

```yaml wrap
checkout:
  - sparse-checkout: |
      node/my-package
      .github
  - repository: org/shared-libs
    path: ./libs/shared
    sparse-checkout: |
      defaults/
```

The `sparse-checkout` field accepts newline-separated path patterns compatible with `actions/checkout`. See [GitHub Repository Checkout](/gh-aw/reference/checkout/#configuration-options) for the full list of checkout configuration options.

## Workflow Design

### Should I focus on one workflow, or write many different ones?

One workflow is simpler to maintain and good for learning, while multiple workflows provide better separation of concerns, different triggers and permissions per task, and clearer audit trails. Start with one or two workflows, then expand as you understand the patterns. See [Peli's Agent Factory](/gh-aw/blog/2026-01-12-welcome-to-pelis-agent-factory/) for examples.

### Should I create agentic workflows by hand editing or using AI?

Either approach works well. AI-assisted authoring using `/agent agentic-workflows create` in GitHub Copilot Chat provides interactive guidance with automatic best practices, while manual editing gives full control and is essential for advanced customizations. See [Creating Workflows](/gh-aw/setup/creating-workflows/) for AI-assisted approach, or [Reference documentation](/gh-aw/reference/frontmatter/) for manual configuration.

### Can the agent use an existing branch specified at runtime (e.g., from a Jira issue)?

The `create-pull-request` safe output always creates a new branch, but you can control its name and make it reuse an existing remote branch. Set these two fields in your workflow frontmatter:

```yaml wrap
safe-outputs:
  create-pull-request:
    preserve-branch-name: true   # omit random salt suffix from agent-specified name
    recreate-ref: true           # force-reset remote branch if it already exists
```

With `preserve-branch-name: true`, the agent's branch name (e.g., `feature/abc-123-my-change`) is used as-is instead of having a random hex suffix appended. With `recreate-ref: true`, if that branch already exists remotely, it is force-reset to the agent's current HEAD rather than falling back to creating an issue.

To pass the branch name from a Jira issue body (or any issue body), instruct the agent in your workflow's markdown:

```markdown
Read the issue body and extract the branch name from the line starting with
"Use existing branch:". Use that name when calling `create_pull_request`.
```

The agent reads the triggering issue body as part of its context, so no extra integration is needed when the branch name is embedded there. For richer Jira data (status, custom fields), use a [custom safe output](/gh-aw/reference/custom-safe-outputs/) or Jira MCP server.

> [!NOTE]
> `recreate-ref` requires `preserve-branch-name: true` to take effect. The agent always starts from the configured base branch — it doesn't literally check out the named branch before making changes.

See [Safe Outputs (Pull Requests)](/gh-aw/reference/safe-outputs-pull-requests/) for full configuration details.

### You use 'agent' and 'agentic workflow' interchangeably. Are they the same thing?

Yes, for the purpose of this technology. An **"agent"** is an agentic workflow in a repository - an AI-powered automation that can reason, make decisions, and take actions. We use **"agentic workflow"** as it's plainer and emphasizes the workflow nature of the automation, but the terms are synonymous in this context.

### How do I forward agent and detection artifacts to a third-party server after the workflow finishes?

Add a custom job with `needs: [conclusion]` in the frontmatter `jobs:` block. The `conclusion` job is the last auto-generated job to run, so depending on it guarantees both the `agent` and `detection` artifacts are fully uploaded before your job starts.

```yaml wrap
jobs:
  forward-artifacts:
    needs: [conclusion]
    if: always()
    runs-on: ubuntu-latest
    steps:
      - uses: actions/download-artifact@v4
        with:
          name: agent
          path: artifacts/agent
      - uses: actions/download-artifact@v4
        with:
          name: detection
          path: artifacts/detection
        continue-on-error: true
      - name: Upload to third-party server
        env:
          INGEST_TOKEN: ${{ secrets.INGEST_TOKEN }}
        run: |
          tar -czf artifacts.tar.gz artifacts/
          curl --fail --retry 3 -X POST https://ingest.example.com/artifacts \
            -H "Authorization: ******" \
            -F "file=@artifacts.tar.gz" \
            -F "run_id=${{ github.run_id }}"
```

`if: always()` ensures the job runs even when the agent or safe-output jobs fail. The `detection` artifact is only present when [threat detection](/gh-aw/reference/threat-detection/) is enabled; `continue-on-error: true` on that step makes the job continue when the artifact doesn't exist.

See [Artifacts](/gh-aw/reference/artifacts/) for a full list of artifact names and their contents.

## Costs & Usage

### Who pays for the use of AI?

This depends on the AI engine (coding agent) you use:

- **GitHub Copilot CLI** (default): Usage is currently associated with the individual GitHub account of the user supplying the [`COPILOT_GITHUB_TOKEN`](/gh-aw/reference/auth/#copilot_github_token), and is drawn from the monthly quota of premium requests for that account. See [GitHub Copilot billing](https://docs.github.com/en/copilot/about-github-copilot/subscription-plans-for-github-copilot).
- **Claude**: Usage is billed to the Anthropic account associated with [`ANTHROPIC_API_KEY`](/gh-aw/reference/auth/#anthropic_api_key) Actions secret in the repository.
- **Codex**: Usage is billed to your OpenAI account associated with [`OPENAI_API_KEY`](/gh-aw/reference/auth/#openai_api_key) Actions secret in the repository.

### What's the approximate cost per workflow run?

Costs vary depending on workflow complexity, AI model, and execution time. GitHub Copilot CLI uses 1-2 premium requests per workflow execution with agentic processing. Track usage with `gh aw logs` for runs and metrics, `gh aw audit <run-id>` for detailed token usage and costs, or check your AI provider's usage portal. Consider creating separate PAT/API keys per repository for tracking.

Reduce costs by optimizing prompts, using smaller models, limiting tool calls, reducing run frequency, and caching results.

### Are GitHub Actions minutes charged in addition to AI costs?

Yes. Every agentic workflow run is a GitHub Actions workflow run, so it consumes Actions minutes alongside AI inference. These are billed separately:

- **Actions minutes**: Standard GitHub Actions billing applies — free for public repos, metered for private repos based on your plan. Set a [spending limit](https://docs.github.com/en/billing/managing-billing-for-your-products/managing-billing-for-github-actions/managing-your-spending-limit-for-github-actions) at the org level to cap Actions spend.
- **AI inference**: Billed through your AI engine account (see [Who pays for the use of AI?](#who-pays-for-the-use-of-ai)).

### How do retries and agent loops affect costs?

gh-aw has no automatic retry mechanism — each workflow trigger produces exactly one run. However, you can control reasoning depth and autopilot continuation, which directly affects how many tokens and how much wall-clock time (Actions minutes) a run consumes:

- `max-turns` (Claude only) — limits the number of AI chat iterations per run
- `max-continuations` (Copilot only) — enables autopilot mode with multiple consecutive triggered runs

```yaml
engine:
  id: claude
max-turns: 5   # limit reasoning depth per run
```

Keep these values low for cost-sensitive workflows. For scheduled workflows, run frequency is the primary cost lever — an hourly schedule at 1–2 premium requests per run adds up quickly across many repositories.

### How do I control spend and set budgets?

Spend controls live at the provider level, not inside gh-aw:

- **Actions minutes**: Set an org spending limit in GitHub Billing settings.
- **Claude / Codex / Gemini**: Configure spend limits in the Anthropic Console or OpenAI platform. These apply at the API key or project level.
- **Copilot**: Usage is quota-based (premium requests per month) rather than dollar-metered, so the natural cap is the plan's monthly request quota.

For per-repository cost tracking, use a dedicated API key per repository so provider dashboards show usage broken down by key. You can also use `gh aw audit <run-id>` for per-run token and cost detail, and `gh aw logs` for run history and aggregate metrics.

### Can I change the model being used, e.g., use a cheaper or more advanced one?

Yes! You can configure the model in your workflow frontmatter:

```yaml wrap
engine:
  id: copilot
  model: gpt-5                    # or claude-sonnet-4
```

Or switch to a different engine entirely:

```yaml wrap
engine: claude
```

See [AI Engines](/gh-aw/reference/engines/) for all configuration options.
