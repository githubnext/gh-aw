---
title: Frequently Asked Questions
description: Answers to common questions about GitHub Agentic Workflows, including security, costs, privacy, and configuration.
sidebar:
  order: 50
---

> [!NOTE]
> GitHub Agentic Workflows is in Public Preview and may change significantly.

## Determinism

### I like deterministic CI/CD. Isn't this non-deterministic?

Agentic workflows are **100% additive** — your deterministic build, test, and release pipelines stay unchanged. Think of it as **Continuous AI** alongside CI/CD: a new automation layer in GitHub Actions for tasks where exact reproducibility doesn't matter, such as triaging issues, drafting documentation, researching dependencies, or proposing code improvements for human review.

## Capabilities

### What's the difference between agentic workflows and regular GitHub Actions workflows?

Agentic workflows use AI to interpret natural language instructions in markdown instead of complex YAML. The AI engine can call pre-approved tools to perform tasks while running with read-only default permissions, safe outputs, and sandboxed execution.

### What's the difference between agentic workflows and just running a coding agent in GitHub Actions?

While you could install and run a coding agent directly in a standard GitHub Actions workflow, agentic workflows provide a structured framework with simpler markdown format, built-in security controls, pre-defined tools for GitHub operations, and easy switching between AI engines.

### Can agentic workflows write code and create pull requests?

Yes — use the `create-pull-request` safe output to propose code changes, documentation updates, or other modifications for human review. If your organization blocks PR creation from GitHub Actions, workflows can still generate diffs or suggestions in issues or comments for manual application.

### Can agentic workflows do more than code?

Yes — analyze repositories, generate reports, triage issues, research information, create documentation, and coordinate work. The AI interprets natural language instructions and uses available [tools](/gh-aw/reference/tools/) to accomplish tasks.

### Can agentic workflows mix regular GitHub Actions steps with AI agentic steps?

Yes. Add custom steps before the agentic job via [`steps:`](/gh-aw/reference/steps-jobs/#custom-steps-steps), consume agentic outputs through [custom safe output jobs](/gh-aw/reference/safe-outputs/#custom-safe-output-jobs-jobs), and pass typed data between steps and the agent with [MCP Scripts](/gh-aw/reference/mcp-scripts/).

### Can agentic workflows read other repositories?

Yes, with a **Personal Access Token (PAT)** that has access to target repositories, configured in your workflow. See [MultiRepoOps](/gh-aw/patterns/multi-repo-ops/) for coordinating across repositories, including running workflows from a separate side repository.

### Can I use agentic workflows in private repositories?

Yes, and in many cases we recommend it. Private repositories are ideal for proprietary code, creating a "sidecar" repository with limited access, testing workflows, and organization-internal automation. See [MultiRepoOps — Side Repository](/gh-aw/patterns/multi-repo-ops/#using-a-side-repository) for patterns using private repositories.

### Can I edit workflows directly on GitHub.com without recompiling?

Yes, for the **markdown body** (AI instructions) — loaded at runtime, takes effect on the next run. **Frontmatter** (tools, permissions, triggers, network rules) is embedded at compile time and requires `gh aw compile my-workflow` after edits. See [Editing Workflows](/gh-aw/guides/editing-workflows/).

### Can workflows trigger other workflows?

Yes, using the `dispatch-workflow` safe output (default `max: 1`):

```yaml wrap
safe-outputs:
  dispatch-workflow:
    max: 1
```

See [Safe Outputs](/gh-aw/reference/safe-outputs/#workflow-dispatch-dispatch-workflow).

### Can I trigger an agentic workflow from an external system like Jira?

Yes. Any system that can make an HTTP request — Jira, PagerDuty, Slack, custom APIs — can trigger a workflow via the [`repository_dispatch`](https://docs.github.com/en/actions/writing-workflows/choosing-when-your-workflow-runs/events-that-trigger-workflows#repository_dispatch) API. Add the trigger to your workflow and access the payload via `${{ github.event.client_payload.* }}`:

```yaml wrap
on:
  repository_dispatch:
    types: [jira-issue-created]
```

Then `POST` to the dispatch API from the external system:

```http
POST https://api.github.com/repos/<owner>/<repo>/dispatches
Authorization: Bearer <token>
Content-Type: application/json

{
  "event_type": "jira-issue-created",
  "client_payload": { "issue_key": "PROJ-123", "summary": "Fix the thing" }
}
```

For Jira, use **Project → Automation → Issue created → Send web request**. The token needs `repo` scope (classic PAT) or `contents: write` permission, stored in the external system's secret store and scoped to the target repository.

See [Repository Dispatch Trigger](/gh-aw/reference/triggers/#repository-dispatch-trigger-repository_dispatch). For runtime branch control from Jira issue content, see [Can the agent use an existing branch specified at runtime?](#can-the-agent-use-an-existing-branch-specified-at-runtime-eg-from-a-jira-issue)

### Can I use MCP servers with agentic workflows?

Yes — [Model Context Protocol (MCP)](/gh-aw/reference/glossary/#mcp-model-context-protocol) servers extend capabilities with custom tools. Configure them in frontmatter:

```yaml wrap
tools:
  mcp-servers:
    my-server:
      image: "ghcr.io/org/my-mcp-server:latest"
      network:
        allowed: ["api.example.com"]
```

See [Using MCPs](/gh-aw/guides/mcps/).

### If my agent can use a skill, can agentic workflows use it too?

Usually yes. For reusable packaging, use [imports](/gh-aw/reference/imports/) for workflow-level config and prompts, and [APM (Agent Package Manager)](https://microsoft.github.io/apm/) for skills and other agent primitives. See [APM Dependencies](/gh-aw/reference/dependencies/).

### The `plugins:` or `dependencies:` field I was using is gone - how do I install agent plugins now?

These fields were replaced by the import-based approach using [Microsoft APM](https://microsoft.github.io/apm/), which supports all agent primitives — skills, prompts, instructions, hooks, and plugins (Copilot and Claude `plugin.json` formats). Use `imports` with the `packages:` parameter:

```yaml wrap
imports:
  - uses: shared/apm.md
    with:
      packages:
        - microsoft/apm-sample-package
        - github/awesome-copilot/skills/review-and-refactor
```

See [APM Dependencies](/gh-aw/reference/dependencies/).

### Can I use Claude plugins with APM?

Yes. When `engine: claude` is set, APM infers the engine target and unpacks only Claude-compatible primitives. See [APM Dependencies](/gh-aw/reference/dependencies/).

### Can workflows be broken up into shareable components?

Yes — import shared configurations:

```yaml wrap
imports:
  - shared/github-tools.md
  - githubnext/agentics/shared/common-tools.md
```

See [Imports](/gh-aw/reference/imports/) and [Packaging Imports](/gh-aw/guides/reusing-workflows/).

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

### How should I configure Go caches safely in agentic workflows?

For Go workflows, cache module downloads and build artifacts explicitly, and scope cache keys tightly:

```yaml wrap
cache:
  key: go-${{ runner.os }}-${{ hashFiles('**/go.sum') }}
  path: |
    ~/go/pkg/mod
    ~/.cache/go-build
  restore-keys: |
    go-${{ runner.os }}-

jobs:
  setup:
    steps:
      - uses: actions/setup-go@v5
        with:
          go-version: '1.25'
          cache: false
      - run: |
          echo "GOMODCACHE=$HOME/go/pkg/mod" >> "$GITHUB_ENV"
          echo "GOCACHE=$HOME/.cache/go-build" >> "$GITHUB_ENV"
```

Security guidance:

- Keep keys specific to OS and dependency lock state (`go.sum`) to reduce accidental cross-context restores.
- Do not share writeable cache keys across trust boundaries (for example, untrusted fork PR runs and protected branch runs).
- Never place secrets in `GOMODCACHE`/`GOCACHE`; these directories should contain only modules and build outputs.

## Guardrails

### Agentic workflows run in GitHub Actions. Can they access my repository secrets?

Not by default — the AI agent runs with read-only permissions. Some MCP tools may be configured with secrets, but those are accessible only to the specific tool steps, not the agent itself. Review workflows carefully, follow [GitHub Actions security guidelines](https://docs.github.com/en/actions/reference/security/secure-use), use least-privilege permissions, inspect the compiled `.lock.yml`, and minimize tools equipped with highly privileged secrets. See the [Security Architecture](/gh-aw/introduction/architecture/).

### Agentic workflows run in GitHub Actions. Can they write to the repository?

The agent step runs read-only by default. Writes require explicit [safe outputs](/gh-aw/reference/safe-outputs/) — limited, specific operations that are sanitized and applied in separate jobs — or explicit general `write` permissions (not recommended).

### What sanitization is done on AI outputs before applying changes?

All safe outputs are sanitized before being applied: secret redaction, URL domain filtering, XML escaping, size limits, control character stripping, GitHub reference escaping, and HTTPS enforcement. Permission separation means write operations happen in separate jobs with scoped permissions, never in the agentic job. See [Text Sanitization](/gh-aw/reference/safe-outputs/#text-sanitization-allowed-domains-allowed-github-references).

### How do I prevent workflow output from creating backlinks in referenced issues?

GitHub creates "mentioned in..." timeline entries when content references issue/PR numbers like `#123` or `owner/repo#456`. Set `allowed-github-references: []` to wrap all references in backticks so GitHub doesn't resolve them — useful when writing about a main repo from a sidecar:

```yaml wrap
safe-outputs:
  allowed-github-references: []   # escape all references
  create-issue:
```

Use `[repo]` to allow only same-repo references. Default (unset) leaves all references unescaped. See [Text Sanitization](/gh-aw/reference/safe-outputs/#text-sanitization-allowed-domains-allowed-github-references).

### How are agent actions constrained — commenting, opening PRs, modifying files, and calling external tools?

gh-aw uses defense-in-depth with four layers:

1. **Read-only agent by default** — no comments, PRs, or pushes unless you configure [safe outputs](/gh-aw/reference/safe-outputs/).
2. **Safe outputs for all writes** — separate jobs with scoped write tokens apply sanitized changes (secret redaction, URL filtering, size limits) from a structured artifact produced by the agent.
3. **Threat detection before writes** — [agentic threat detection](/gh-aw/reference/threat-detection/) runs between the agent and safe output jobs, blocking writes on prompt injection, secret leaks, or malicious patches.
4. **Network allowlist** — the [Agent Workflow Firewall](/gh-aw/reference/sandbox/) blocks outbound traffic except to domains you explicitly allow.

For sensitive operations, layer on a [GitHub Environment protection rule](#can-i-require-external-human-approval-before-safe-outputs-are-applied) so a reviewer must approve before write jobs run. Compilation-time validation (schema checks, expression safety, action SHA pinning) and tool allowlisting add further defense — see the [Security Architecture](/gh-aw/introduction/architecture/).

### Can I require external human approval before safe outputs are applied?

Yes. Apply **[GitHub Environment protection rules](https://docs.github.com/en/actions/managing-workflow-runs-and-deployments/managing-deployments/managing-environments-for-deployment#required-reviewers)** to a [custom safe output job](/gh-aw/reference/custom-safe-outputs/) that the built-in `safe_outputs` job depends on. The job pauses until a designated reviewer approves — enforced by GitHub's infrastructure, not workflow logic the agent could influence. Threat detection runs before the gate, so reviewers see output that already passed automated scanning.

```yaml wrap
jobs:
  approval-gate:
    runs-on: ubuntu-latest
    needs: detection
    environment: production-deploy   # configure reviewers in Settings → Environments
    steps:
      - run: echo "Execution approved"

safe-outputs:
  needs: [approval-gate]
```

For **fully off-platform admission control** (an external policy engine, PAM/PIM, or compliance workflow), call that system from the gate job — if the call fails or is denied, the safe output jobs never run:

```yaml wrap
jobs:
  external-admission:
    runs-on: ubuntu-latest
    needs: [agent, detection]
    steps:
      - name: Request admission
        env:
          POLICY_TOKEN: ${{ secrets.POLICY_TOKEN }}
        run: |
          curl --fail -X POST https://YOUR_POLICY_ENGINE/v1/admit \
            -H "Authorization: Bearer $POLICY_TOKEN" \
            -d '{"workflow_run": "${{ github.run_id }}"}'

safe-outputs:
  needs: [external-admission]
```

### How is my code and data processed?

The workflow runs on GitHub Actions, invoking your chosen [AI Engine](/gh-aw/reference/engines/) in a container which may make tool and MCP calls. Data handling depends on the engine: **GitHub Copilot CLI** uses GitHub Copilot's services ([Copilot docs](https://docs.github.com/en/copilot)); **Claude/Codex** use their providers' APIs and policies. See the [Security Architecture](/gh-aw/introduction/architecture/) for execution and data flow details.

### Does the underlying AI engine run in a sandbox?

Yes — the [AI engine](/gh-aw/reference/engines/) runs in a container inside a GitHub Actions VM, with network egress control via the [Agent Workflow Firewall](/gh-aw/reference/sandbox/), container isolation, Actions resource constraints, and filesystem access limited to workspace and temp directories. See [Sandbox Configuration](/gh-aw/reference/sandbox/).

### Can an agentic workflow use outbound network requests?

Yes, but the [Agent Workflow Firewall](/gh-aw/reference/sandbox/) blocks outbound traffic by default — declare allowed domains:

```yaml wrap
network:
  allowed:
    - defaults             # basic infrastructure
    - python               # PyPI
    - "api.example.com"    # custom domain
```

See [Network Permissions](/gh-aw/reference/network/).

### How does integrity filtering protect my workflow?

[Integrity filtering](/gh-aw/reference/integrity/) controls which GitHub content the agent sees, filtering by **author trust** and **merge status**. The MCP gateway removes content below the configured `min-integrity` threshold before the agent sees it.

For **public repositories**, `min-integrity: approved` is auto-applied at runtime — restricting content to owners, members, and collaborators. For triage or spam-detection workflows that need all users' content, set `min-integrity: none` explicitly:

```yaml wrap
tools:
  github:
    min-integrity: none
```

See [Integrity Filtering](/gh-aw/reference/integrity/).

## Configuration & Setup

### Why do slash-command workflows show many "started then skipped" runs on comments?

Expected behavior. A `slash_command` compiles into multiple event listeners (issue/PR bodies, comments, review comments). GitHub dispatches every event, then activation logic checks whether the comment starts with the matching command — non-matches exit early and appear as skipped runs. Narrow the trigger with `events:`, and use [LabelOps](/gh-aw/patterns/label-ops/) for command-style operations that shouldn't activate on every comment:

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

The `.lock.yml` file is the compiled GitHub Actions workflow generated from your `.md` by `gh aw compile`. It contains SHA-pinned actions, resolved imports, permissions, and all guardrail hardening — inspect it to see exactly what will run. Commit both files:

- **`.md`**: source; edit the prompt body freely without recompiling
- **`.lock.yml`**: what GitHub Actions runs; regenerate after any frontmatter change

### What is the actions-lock.json file?

The `.github/aw/actions-lock.json` file caches resolved `action@version` → ref mappings. The compiler tries to pin each action to an immutable commit SHA, but resolving a tag to a SHA requires GitHub API access — which can fail under limited-permission tokens (e.g., Copilot Coding Agent). The cache reuses previously resolved refs regardless of the current token's capabilities; without it, compilation is unstable.

Commit it to version control. Refresh with `gh aw update-actions`, or delete and recompile to force re-resolution. See [Action Pinning](/gh-aw/reference/compilation-process/#action-pinning).

### What is `github/gh-aw-actions`?

The GitHub Actions repository containing all reusable actions that power compiled agentic workflows. Compiled `.lock.yml` files reference them as `github/gh-aw-actions/setup@<ref>` (usually a SHA, sometimes a stable version tag). Managed entirely by `gh aw compile` — never edit manually. See [The gh-aw-actions Repository](/gh-aw/reference/compilation-process/#the-gh-aw-actions-repository).

### Why is Dependabot opening PRs to update `github/gh-aw-actions`?

Dependabot scans `.lock.yml` files for action references and treats `github/gh-aw-actions` pins as regular dependencies to update. **Do not merge these PRs.** Action pins in compiled workflows should only be updated by running `gh aw compile` or `gh aw update-actions`.

Suppress these PRs by adding an `ignore` entry in `.github/dependabot.yml`:

```yaml
updates:
  - package-ecosystem: github-actions
    directory: "/.github/workflows"
    ignore:
      - dependency-name: "github/gh-aw-actions" # Managed by gh aw compile. Version-locked to the gh-aw compiler; do not bump.
```

See [Dependabot and gh-aw-actions](/gh-aw/reference/compilation-process/#dependabot-and-gh-aw-actions) for more details.

### How does `gh aw upgrade` resolve action versions when no GitHub Releases exist?

`gh aw upgrade` (and `gh aw update-actions`) tries the **GitHub Releases API** first via the `gh` CLI; if no releases exist, it falls back to **git tags** via `git ls-remote`. Tags are a valid source for version pinning, so the fallback is safe to ignore. A warning appears only when both sources are empty.

`github/gh-aw-actions` intentionally publishes only tags. The earlier `no releases found` warning has been fixed — the tag fallback now runs automatically.

### Why do I need a token or key?

**GitHub Copilot CLI** requires a Personal Access Token with "Copilot Requests" permission to authenticate, track usage against your subscription, and audit actions. See [Authentication](/gh-aw/reference/auth/).

### Can I use `CLAUDE_CODE_OAUTH_TOKEN` with the Claude engine?

No. The Claude engine only supports [`ANTHROPIC_API_KEY`](/gh-aw/reference/auth/#anthropic_api_key) as a GitHub Actions secret. Provider-based OAuth (e.g., Claude Teams billing) is not supported. See [Authentication](/gh-aw/reference/auth/) and [AI Engines](/gh-aw/reference/engines/#available-coding-agents).

### What hidden runtime dependencies does this have?

None hidden — the executing workflow uses your chosen coding agent (default: Copilot CLI), a GitHub Actions VM with NodeJS, pinned Actions from [github/gh-aw](https://github.com/github/gh-aw) releases, and the Agent Workflow Firewall container (optional but default). The compiled `.lock.yml` shows the exact YAML.

### Why are macOS runners not supported?

macOS runners (`macos-*`) don't support container jobs, which agentic workflows require for the [Agent Workflow Firewall](/gh-aw/reference/sandbox/) sandbox. Use `ubuntu-latest` or another Linux runner. For genuine macOS-only tooling, run those steps in a separate regular GitHub Actions job that coordinates with your agentic workflow.

### Can I use agentic workflows on GitHub Enterprise Server (GHES)?

Yes, but enable GHES compatibility mode on instances predating `@actions/artifact` v2.0.0 — otherwise compiled workflows fail with `GHESNotSupportedError` because the compiler emits `upload-artifact@v4+` by default. Compatibility mode emits `v3.2.2`/`v3.1.0` instead:

```json
// aw.json (applies to all workflows)
{ "ghes": true }
```

```bash
# or one-off:
gh aw compile --ghes my-workflow.md
```

`gh aw init` auto-detects GHES and writes `ghes: true` for you. See [Enterprise Configuration](/gh-aw/reference/enterprise-configuration/) for CLI and Copilot prerequisites.

### I'm not using a supported AI Engine (coding agent). What should I do?

Supported engines are Copilot, Claude, Codex, Gemini, and Crush. Contribute support to the [gh-aw repository](https://github.com/github/gh-aw) or open an issue describing your use case. See [AI Engines](/gh-aw/reference/engines/).

### Can I test workflows without affecting my repository?

Yes — use [TrialOps](/gh-aw/experimental/trial-ops/) to run workflows in isolated trial repositories without creating real issues, PRs, or comments.

### Where can I find help with common issues?

See [Common Issues](/gh-aw/troubleshooting/common-issues/).

### Why is my create-discussion workflow failing?

Ensure discussions are enabled (**Settings → Features → Discussions**) and the workflow has `discussions: write` permission. For category matching failures, verify spelling (case-insensitive) and use lowercase slugs (e.g., `general`, `announcements`) rather than display names.

Use `fallback-to-issue: true` (the default) to automatically create an issue if discussions aren't available. See [Discussion Creation](/gh-aw/reference/safe-outputs/#discussion-creation-create-discussion) for details.

### How do I enable discussions in add-comment?

By default, `add-comment` does not request `discussions: write` — the permission is opt-in. To comment on discussions, set `discussions: true`:

```yaml wrap
safe-outputs:
  add-comment:
    discussions: true
```

`issues: write` and `pull-requests: write` are requested by default; opt out per-permission with `issues: false` or `pull-requests: false`.

### Why is my create-pull-request workflow failing with "GitHub Actions is not permitted to create or approve pull requests"?

Some organizations block PR creation by GitHub Actions (**Settings → Actions → General → Workflow permissions**). If you can't enable it:

- **Automatic issue fallback (default)**: `fallback-as-issue: true` creates an issue with the branch link when PR creation is blocked. Requires `contents: write`, `pull-requests: write`, `issues: write`.
- **Assign to Copilot**: create an issue assigned to `copilot` for automated implementation (`assignees: [copilot]` under `create-issue`).
- **Disable fallback**: set `fallback-as-issue: false` to fail when PR creation is blocked (requires only `contents: write` and `pull-requests: write`).

See [Pull Request Creation](/gh-aw/reference/safe-outputs/#pull-request-creation-create-pull-request).

### Why don't pull requests created by agentic workflows trigger my CI checks?

PRs created with the default `GITHUB_TOKEN` or the GitHub Actions bot don't trigger `pull_request`, `pull_request_target`, or `push` workflows — a [GitHub Actions security feature](https://docs.github.com/en/actions/security-for-github-actions/security-guides/automatic-token-authentication#using-the-github_token-in-a-workflow) preventing recursive execution. Fix by setting `GH_AW_CI_TRIGGER_TOKEN` to a PAT with 'Contents: Read & Write'. See [Triggering CI](/gh-aw/reference/triggering-ci/).

### How do I suppress the "Generated by..." text in workflow outputs?

Set `footer: false` to hide the `> Generated by [Workflow](run_url) for issue #N` line while preserving the hidden XML markers used for search:

```yaml wrap
safe-outputs:
  footer: false            # hide for all
  create-pull-request:
    footer: true           # override per type
```

The hidden `<!-- gh-aw-workflow-id: ... -->` marker remains — search GitHub for `"gh-aw-workflow-id: my-workflow" in:body`. See [Footer Control](/gh-aw/reference/footers/).

### My workflow fails with "Runtime import file not found" when used in a repository ruleset

Required-status-check workflows run without filesystem access, so runtime imports can't be resolved. Set `inlined-imports: true` in frontmatter to bundle imports into `.lock.yml` at compile time. See [Self-Contained Lock Files](/gh-aw/reference/imports/#self-contained-lock-files-inlined-imports-true).

### My cross-organization `workflow_call` fails with a repository checkout error

The activation job tries to check out the callee's `.github` folder with the caller's `GITHUB_TOKEN`, which can't access a private repo in another organization (`fatal: repository '...' not found`). Set `inlined-imports: true` on the **platform workflow** (callee) to embed imports at compile time and eliminate the cross-org checkout:

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

See [Self-Contained Lock Files](/gh-aw/reference/imports/#self-contained-lock-files-inlined-imports-true).

### My workflow checkout is very slow because my repository is a large monorepo. How can I speed it up?

Use **sparse checkout** in the `checkout:` field to fetch only the paths your workflow needs — often reducing checkout from minutes to seconds:

```yaml wrap
checkout:
  sparse-checkout: |
    node/my-package
    .github
```

For multiple paths with different settings, combine checkouts:

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

One workflow is simpler to maintain; multiple workflows give better separation of concerns, per-task triggers and permissions, and clearer audit trails. Start with one or two and expand as patterns emerge. See [Peli's Agent Factory](/gh-aw/blog/2026-01-12-welcome-to-pelis-agent-factory/).

### Should I create agentic workflows by hand editing or using AI?

Both work. AI-assisted authoring via `agentic-workflows create` in Copilot Chat gives interactive guidance and best practices; manual editing gives full control for advanced customization. See [Creating Workflows](/gh-aw/setup/creating-workflows/) or [Frontmatter Reference](/gh-aw/reference/frontmatter/).

### Can the agent use an existing branch specified at runtime (e.g., from a Jira issue)?

`create-pull-request` always creates a new branch, but you can control the name and reuse an existing remote branch:

```yaml wrap
safe-outputs:
  create-pull-request:
    preserve-branch-name: true   # use agent name as-is, no random suffix
    recreate-ref: true           # force-reset remote branch if it exists
```

To pass the branch name from a Jira issue body (or any issue body), instruct the agent in markdown:

```markdown
Read the issue body and extract the branch name from the line starting with
"Use existing branch:". Use that name when calling `create_pull_request`.
```

The agent has the issue body in context, so no extra integration is needed. For richer Jira data (status, custom fields), use a [custom safe output](/gh-aw/reference/custom-safe-outputs/) or Jira MCP server.

> [!NOTE]
> `recreate-ref` requires `preserve-branch-name: true`. The agent always starts from the configured base branch — it doesn't check out the named branch before making changes.

See [Safe Outputs (Pull Requests)](/gh-aw/reference/safe-outputs-pull-requests/).

### You use 'agent' and 'agentic workflow' interchangeably. Are they the same thing?

Yes — an **"agent"** is an agentic workflow in a repository. We use **"agentic workflow"** to emphasize the workflow nature, but the terms are synonymous.

### How do I forward agent and detection artifacts to a third-party server after the workflow finishes?

Add a custom job with `needs: [conclusion]` in the frontmatter `jobs:` block. The `conclusion` job is the last auto-generated job, so depending on it guarantees both `agent` and `detection` artifacts are fully uploaded:

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

`if: always()` runs the job even on upstream failure. The `detection` artifact only exists when [threat detection](/gh-aw/reference/threat-detection/) is enabled — `continue-on-error: true` handles its absence. See [Artifacts](/gh-aw/reference/artifacts/) for artifact names and contents.

## Costs & Usage

### Who pays for the use of AI?

Depends on the engine:

- **GitHub Copilot CLI** (default): the account supplying [`COPILOT_GITHUB_TOKEN`](/gh-aw/reference/auth/#copilot_github_token) — drawn from its inference quota. See [Copilot billing](https://docs.github.com/en/copilot/about-github-copilot/subscription-plans-for-github-copilot).
- **Claude**: the Anthropic account tied to the [`ANTHROPIC_API_KEY`](/gh-aw/reference/auth/#anthropic_api_key) secret.
- **Codex**: the OpenAI account tied to the [`OPENAI_API_KEY`](/gh-aw/reference/auth/#openai_api_key) secret.

### What's the approximate cost per workflow run?

Costs vary by workflow complexity, model, and execution time. Track usage with `gh aw logs`, `gh aw audit <run-id>`, or your AI provider's portal — use separate PAT/API keys per repository for granular tracking. Reduce costs by optimizing prompts, using smaller models, limiting tool calls, reducing run frequency, and caching results.

### Are GitHub Actions minutes charged in addition to AI costs?

Yes — every run consumes Actions minutes (free for public repos, metered for private) alongside AI inference. Set an [org spending limit](https://docs.github.com/en/billing/managing-billing-for-your-products/managing-billing-for-github-actions/managing-your-spending-limit-for-github-actions) to cap Actions spend. AI inference is billed separately (see [Who pays for the use of AI?](#who-pays-for-the-use-of-ai)).

### How do retries and agent loops affect costs?

gh-aw has no automatic retries — each trigger produces exactly one run. Control reasoning depth and continuation to bound tokens and wall-clock time:

- `max-turns` (Claude) — limits AI chat iterations per run
- `max-continuations` (Copilot) — autopilot mode with consecutive triggered runs

```yaml
engine:
  id: claude
max-turns: 5
```

For scheduled workflows, run frequency is the primary cost lever — an hourly schedule adds up quickly.

### How do I control spend and set budgets?

Spend controls live at the provider level:

- **Actions minutes**: org spending limit in GitHub Billing.
- **Claude / Codex / Gemini**: API key or project-level limits in Anthropic Console / OpenAI platform.
- **Copilot**: quota-based — the plan's monthly request quota is the natural cap.

For per-repository tracking, use a dedicated API key per repository. Use `gh aw audit <run-id>` for per-run detail and `gh aw logs` for aggregate metrics.

### Can I change the model being used, e.g., use a cheaper or more advanced one?

Yes — set the model in frontmatter, or switch engines:

```yaml wrap
engine:
  id: copilot
  model: gpt-5                    # or claude-sonnet-4
```

```yaml wrap
engine: claude
```

See [AI Engines](/gh-aw/reference/engines/).
