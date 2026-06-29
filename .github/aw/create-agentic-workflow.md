---
description: Create new agentic workflows using GitHub Agentic Workflows (gh-aw) with concise guidance on triggers, tools, and security.
disable-model-invocation: true
---

# GitHub Agentic Workflow Creator

Create new workflow files under `.github/workflows/` using the installed `gh aw` CLI.

## Load These References First

- [github-agentic-workflows.md](github-agentic-workflows.md)
- [workflow-editing.md](workflow-editing.md)
- [workflow-constraints.md](workflow-constraints.md)
- [workflow-patterns.md](workflow-patterns.md)
- [safe-outputs.md](safe-outputs.md)
- [syntax.md](syntax.md)
- [mcp-clis.md](mcp-clis.md)

Load these topic files only when relevant:

- [campaign.md](campaign.md) for campaign, KPI, pacing, cadence, or `stop-after`
- [experiments.md](experiments.md) for experiments, A/B tests, variants, or prompt comparisons
- [visual-regression.md](visual-regression.md) for screenshot comparison workflows
- [deployment-status.md](deployment-status.md) for external deployment monitoring
- [charts.md](charts.md) for chart-generation workflows
- [report.md](report.md) for reporting output structure and recurring report lifecycle

## Two Modes

### Interactive mode

Start with exactly:

> What do you want to automate today?

Then ask only the next question needed.

### Issue-form mode

When triggered from a workflow-creation issue form, read the form fields and generate the workflow without further conversation.

## Conversation Rules

- Keep the conversation short and iterative.
- Translate user intent into workflow structure.
- Ask about the trigger, desired action, and required write outputs.
- Do not overwhelm the user with long option dumps unless they ask.
- If the request exceeds the single-job model, explain the constraint and recommend traditional GitHub Actions.

## Design Checklist

### 1. Pick the workflow ID

- Derive kebab-case from the workflow name.
- Before creating the file, check whether `.github/workflows/<workflow-id>.md` already exists.
- If it exists, choose a more specific ID instead of overwriting.

### 2. Choose the trigger

Use the smallest trigger that matches the request.

Common mappings:

- issue automation → `on: issues:`
- pull request automation → `on: pull_request:`
- scheduled reporting → fuzzy `schedule:` such as `daily on weekdays`
- on-demand comments → `slash_command`
- UI-driven actions → `label_command`
- GitHub Actions pipeline monitoring → `workflow_run`
- external deployment monitoring → `deployment_status`

Quick decision matrix:

| User intent | Trigger | Typical read tools | Typical safe output |
|---|---|---|---|
| Review PR changes, comment on quality, suggest fixes | `pull_request` | `github` (`gh-proxy`), optional `playwright` for UI diffs | `add-comment` |
| Investigate failed CI/Actions runs and summarize incident | `workflow_run` | `github` (`gh-proxy`) with `actions: read` | `create-issue` |
| Monitor external service deployment failures (Heroku, Vercel, Fly.io) | `deployment_status` | `github` (`gh-proxy`) with `deployments: read` | `create-issue` |
| Run visual regression checks on PR UI changes | `pull_request` | `playwright` + `cache-memory` | `add-comment` |
| Publish weekly stakeholder/product digest | `schedule` | `github` (`gh-proxy`) | `create-issue` (default), `create-discussion` only if explicitly requested |
| Review dependency licenses or design-token governance on PRs | `pull_request` with `paths:` | `github` (`gh-proxy`) | `add-comment`; `create-issue` only for blocked/policy-violating findings |

> **`workflow_run` vs `deployment_status`**: Use `workflow_run` when monitoring another GitHub Actions workflow in the same repository. Use `deployment_status` when an external service (Heroku, Vercel, Fly.io) reports deployment results back to GitHub via the Deployments API. See [deployment-status.md](deployment-status.md) for the full pattern.
>
> For `workflow_run`, always scope explicitly: set `workflows:` to named upstream workflow(s), use `types: [completed]`, and gate outcomes with an `if:` guard on `${{ github.event.workflow_run.conclusion }}` (for incident triage, usually `failure`, `timed_out`, `cancelled`, `action_required`) unless the user asked for success reporting.

Use [workflow-patterns.md](workflow-patterns.md) for trigger-selection guidance.

Compact scenario examples:

- **Schema/API review on PRs**: trigger `pull_request` with `paths:` scoped to backend contract files (for example `db/migrate/**`, `migrations/**`, `schema/**`, `openapi/**`, `api/**`), read via `github` (`gh-proxy`), publish findings with `add-comment`, call `noop` when contracts are unchanged.
- **Visual regression on UI changes**: trigger `pull_request`, use only `playwright` + `cache-memory` (no extra tools), keep network minimal (allowlist only target preview/app hosts if required), state the exact baseline source (`cache-memory` key, artifact, or branch path), publish via `add-comment`, call `noop` when UI paths are unchanged.
- **Deployment incident triage**: use `deployment_status` for external provider failures and `workflow_run` for GitHub Actions failures, publish incident reports via `create-issue`, derive a stable failure key (for example workflow + job + failing step or error signature), and call `noop` when a failure self-recovers or matches an existing open incident.
- **Product/stakeholder digest**: use fuzzy `schedule` plus optional `workflow_dispatch`, define an explicit window (for example `last 7 full days ending at run start (UTC)`), choose grouping dimensions up front (for example team, service, owner, severity, or status), publish with `create-issue` by default, and call `noop` when there are no updates in that window.
- **Dependency-license compliance review on PRs**: trigger `pull_request` with `paths:` scoped to dependency manifests (for example `package.json`, `go.mod`, `requirements.txt`, `Cargo.toml`), read new dependency additions via `github` (`gh-proxy`), classify each addition by license tier (allowed / needs-review / blocked), publish findings with `add-comment`, escalate blocked additions with `create-issue` for team-wide follow-up, call `noop` when no new dependencies were added or all additions are pre-approved.

Pattern-specific `noop` examples:

- **PR reviewer (`pull_request`)**: `noop` when only docs/metadata changed outside scoped `paths:`.
- **Failure triage (`workflow_run`)**: `noop` when rerun succeeds, signal is flake-only, or an open incident already exists for the same failure key.
- **Scheduled digest (`schedule`)**: `noop` when the exact reporting window (for example `since previous successful run`) has zero qualifying updates.
- **Deployment monitor (`deployment_status`)**: `noop` when non-terminal statuses (`queued`, `in_progress`) arrive without a terminal failure.

For compact prefetch + duplicate suppression patterns in reporting/incident workflows, follow [workflow-patterns.md](workflow-patterns.md) and [report.md](report.md) instead of embedding long inline instructions.

### 2a. Reporting and digest compact guidance

For recurring reports, audits, and stakeholder digests:

- default to `create-issue`; use `create-discussion` only when the requester explicitly wants threaded discussion
- use `add-comment` only when updating an existing issue or pull request instead of creating a new report destination
- define the report window explicitly (for example `last 7 full days ending at workflow start (UTC)` or `since previous successful run`)
- define the grouping dimensions explicitly (for example by team, service, owner, severity, status, or repository)
- add `workflow_dispatch` when manual reruns, backfills, or preview runs should be possible
- require `noop` when the selected window has no qualifying updates

**Duplicate-suppression checklist for recurring reports and audits:**

- [ ] Define a stable deduplication key (for example `workflow + window-date` or `scope + YYYY-W01` using ISO week notation)
- [ ] Search for an existing open issue with that key before creating a new one (for example by title prefix or label)
- [ ] If a matching open issue exists, update it with `add-comment` instead of creating a duplicate
- [ ] If the window has zero qualifying updates, call `noop` — never create an empty or placeholder report
- [ ] Escalate with a new issue only when no open issue covers the same scope and window

### 2b. Backend review compact guidance

For backend-focused PR automation (schema migrations and API compatibility):

- scope `pull_request.paths` to backend contract indicators instead of whole-repo review
- instruct the agent to classify changes as additive, backward-compatible, or breaking, then report only actionable risks
- include explicit `noop` criteria when no migration/API contract files changed

### 2c. PR analyzer escalation guidance

For PR-triggered automation that must decide between commenting, creating an issue, or doing nothing:

| Condition | Action |
|---|---|
| Findings affect only this PR (style, quality, risk) | `add-comment` on the PR |
| Finding is a cross-cutting or team-wide concern requiring follow-up beyond this PR | `create-issue` |
| No findings, or only docs/metadata changed outside scoped `paths:` | `noop` |

Rules:

- prefer `add-comment` over `create-issue` for PR-local findings; issues outlive the PR and create noise
- before creating an issue, search for an existing open issue covering the same concern (use a stable title prefix or label to avoid duplicates)
- if a matching open issue already exists, add a linked `add-comment` on the PR referencing it instead of opening a duplicate issue
- call `noop` explicitly whenever no actionable finding exists — do not comment with "no issues found" text

### 2d. Compliance review compact guidance

For dependency-license compliance and policy review on PRs:

- scope `pull_request.paths` to dependency manifest files (for example `package.json`, `go.mod`, `requirements.txt`, `Cargo.toml`, `pyproject.toml`, `composer.json`)
- classify each new dependency by license tier using the project's configured policy (the example tiers below represent a common MIT-compatible policy; adjust for your project): **allowed** (MIT, Apache-2.0, BSD, ISC), **needs-review** (unknown, dual-licensed, weak-copyleft), **blocked** (strong-copyleft such as GPL/AGPL, proprietary, or licenses incompatible with your project's license)
- publish per-tier findings with `add-comment` listing each dependency, its version, and detected license
- escalate to `create-issue` only when a **blocked** dependency was added or a policy violation requires team-wide follow-up beyond this PR
- before creating a new issue, search for an existing open issue with a stable key (for example `license-violation + dependency-name + version`) to avoid duplicates; if found, link to it from the PR comment instead
- call `noop` when no new dependencies were added or all additions are confirmed in the allowed tier

**Compliance escalation decision table:**

| Finding | Action |
|---|---|
| No dependency manifest files changed | `noop` immediately |
| All new dependencies in allowed tier | `noop` (or brief `add-comment` confirmation when the workflow prompt explicitly requests a confirmation comment) |
| Dependencies in needs-review tier | `add-comment` listing them with license details and requesting maintainer confirmation |
| Blocked dependency added | `add-comment` flagging the violation + `create-issue` for team-wide record (skip `create-issue` if a matching open issue already exists) |

### 3. Keep permissions read-only

The main agent job must stay read-only.

- Do not grant `issues: write`, `pull-requests: write`, or `contents: write` to the agent job.
- Route GitHub writes through `safe-outputs:`.
- When targeting the Copilot coding agent, recommend `permissions: { copilot-requests: write }` so Copilot can authenticate with `${{ github.token }}`.
- If the user asks for direct writes, explain why the safe-output pattern is required.

### 4. Select tools

- `bash` and `edit` are enabled by default in sandboxed workflows; do not add them unless you are restricting them.
- For GitHub reads, prefer `tools.github.mode: gh-proxy` and instruct the agent to use `gh` commands.
- For non-GitHub MCP servers, prefer `tools.cli-proxy: true` and instruct the agent to use the mounted `mcp-clis` commands.
- Combined configuration example for GitHub reads plus non-GitHub MCP CLI access:

  ```yaml
  tools:
    github:
      mode: gh-proxy
      toolsets: [default]
    cli-proxy: true
  ```

  Omit `cli-proxy: true` when the workflow only needs GitHub reads.

- Suggest `playwright` for browser automation.
- Suggest dedicated topic files rather than embedding long tutorials in the prompt.

### 5. Infer network access from repository files

Do not ask for the ecosystem if it can be inferred from the repository.

Common mappings:

- `.csproj`, `.fsproj`, `*.sln`, `*.slnx`, `global.json` → `dotnet`
- `requirements.txt`, `pyproject.toml`, `setup.py`, `uv.lock` → `python`
- `package.json`, `.nvmrc`, `yarn.lock`, `pnpm-lock.yaml` → `node`
- `go.mod`, `go.sum` → `go`
- `pom.xml`, `build.gradle`, `build.gradle.kts` → `java`
- `Gemfile`, `*.gemspec` → `ruby`
- `Cargo.toml`, `Cargo.lock` → `rust`
- `Package.swift`, `*.podspec` → `swift`
- `composer.json` → `php`
- `pubspec.yaml` → `dart`

Never use `network: defaults` alone for workflows that build, test, or install packages.

### 6. Configure safe outputs

Map write behavior to `safe-outputs:`.

Common mappings:

- create issues → `create-issue`
- add comments → `add-comment`
- create PRs → `create-pull-request`
- add labels → `add-labels`
- attach downloadable files → `upload-artifact`
- publish embeddable assets → `upload-asset`

Rules:

- always restrict `create-pull-request.allowed-files`
- prefer the dedicated safe output instead of shelling out to `gh` for the same mutation
- include `noop` guidance in the prompt so successful no-op runs are explicit
- when using `create-issue`, instruct the agent to provide a meaningful body (20-65000 characters; avoid placeholder-only text)

### 7. Decide who can trigger the workflow

- Default behavior is team-only triggering.
- For community-facing issue triage or other public entrypoints, recommend `roles: all`.

### 8. Add cost-aware triage and context flow

- For high-volume inputs, design a cheap triage step before expensive analysis.
- Require explicit `noop` or safe-output behavior for known, duplicate, stale, or low-value cases.
- Reserve frontier-model reasoning for ambiguous/high-value cases and final synthesis.
- Prefer pull-on-demand context retrieval over prompt-stuffing large logs or API payloads.
- Use deterministic `steps:` plus compact files under `/tmp/gh-aw/` when large data must be preprocessed.

See also: [workflow-patterns.md](workflow-patterns.md), [subagents.md](subagents.md), and [token-optimization.md](token-optimization.md).

### 9. Omit unnecessary defaults

Avoid adding fields just to restate defaults.

Usually omit:

- `engine: copilot`
- unrestricted `bash`
- `edit`
- `timeout-minutes:` unless a custom timeout is needed

## Prompt Requirements

The markdown body should:

- state the workflow goal clearly
- reference the triggering context explicitly
- name the allowed safe outputs when write actions are expected
- instruct the agent to call `noop` when no visible change is needed
- stay concise and task-focused

When the workflow generates reports or markdown output, include these formatting rules only when relevant:

- use GitHub-flavored markdown
- start nested report headings at `###`
- use `<details><summary>...</summary>` for long collapsible sections
- format workflow run links as `[§12345](https://github.com/owner/repo/actions/runs/12345)`

## Issue-Form Mode Procedure

When processing a workflow-creation issue form:

1. extract the workflow name, description, and additional context
2. derive a unique workflow ID
3. infer the trigger, tools, network access, and safe outputs
4. create exactly one workflow markdown file
5. compile it with `gh aw compile <workflow-id>`
6. include the generated `.lock.yml` in the PR

## Recommended Workflow Skeleton

```markdown
---
emoji: 🏷️
description: <brief description>
on:
  issues:
    types: [opened]
permissions:
  contents: read
  issues: read
tools:
  github:
    mode: gh-proxy
    toolsets: [default]
  cli-proxy: true
safe-outputs:
  add-comment:
---

# <Workflow Name>

## Task

<clear instructions>

## Safe Outputs

- Use the configured safe outputs for visible actions.
- Use `noop` with a short explanation when no action is required.
```

## PR-Report Checklist

Before finalizing any `pull_request`-triggered reporting workflow, verify:

- [ ] **Permissions**: `contents: read` + `pull-requests: read` in the agent job; no write permissions
- [ ] **Safe outputs**: `add-comment` for inline findings; `create-issue` for incidents needing follow-up
- [ ] **Network**: infer ecosystem from repository lock files; never use `defaults` alone when packages are installed
- [ ] **`noop` required**: prompt instructs the agent to call `noop` with a brief explanation when no issues are found

## Generated Workflow Quality Checklist

Before finalizing any newly generated workflow, verify:

- [ ] **Trigger fit**: trigger matches user intent and event granularity (for example `pull_request`, `workflow_run`, `deployment_status`, `schedule`, `slash_command`)
- [ ] **Tool fit**: enabled tools are the minimal set needed for reads/analysis (prefer `gh-proxy`; add `playwright`/`cache-memory` only when required)
- [ ] **Safe outputs**: all visible writes route through `safe-outputs:` and include `noop` for explicit no-op outcomes
- [ ] **Permissions**: agent job remains read-only; no direct write scopes granted
- [ ] **Network**: access is inferred from repository ecosystem and avoids `network: defaults` alone for install/build/test workflows
- [ ] **Prompt clarity**: prompt is concise, context-aware, and clearly states expected outputs and stop/no-op behavior

## Generated Workflow Scoping Checklist

Before finalizing any newly generated workflow, verify:

- [ ] **Paths scope**: include `paths:`/`paths-ignore:` when the automation should ignore unrelated files (for backend reviews, include migration/schema/API contract globs)
- [ ] **Labels scope**: define required labels (for example `label_command` names or PR/issue label filters) when label-based routing is expected
- [ ] **Workflow-name scope**: for `workflow_run`, explicitly set `workflows:` to named targets and gate conclusions via `if:` on `${{ github.event.workflow_run.conclusion }}` (for incident triage, prefer failure-only outcomes)
- [ ] **Date-window scope**: for reporting/triage, state the exact window (for example `last 24h`, `since previous run`, `current week`)
- [ ] **Safe-output write contract**: name which safe output is used for each outcome and when `noop` is required instead of a write

## Multi-Repository Requests

For cross-repository workflows:

- enable the GitHub toolsets needed to read external repositories
- configure cross-repo authentication in `safe-outputs:`
- tell the agent to set `target-repo`
- explain that the workflow still cannot wait for external workflows or create multi-job orchestration

Use [workflow-patterns.md](workflow-patterns.md) for the compact cross-repo pattern.

## Final Steps

1. create `.github/workflows/<workflow-id>.md`
2. compile with `gh aw compile <workflow-id>`
3. fix all compile errors
4. create a PR with the workflow file and `.lock.yml`

## Guidelines

- create exactly one workflow `.md` file as the primary deliverable
- keep prompts short, specific, and imperative
- prefer dedicated reference files over repeating large explanations inline
- always compile before finishing
- keep responses concise after the workflow is created
