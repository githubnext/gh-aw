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

Use [workflow-patterns.md](workflow-patterns.md) for trigger-selection guidance.

### 3. Keep permissions read-only

The main agent job must stay read-only.

- Do not grant `issues: write`, `pull-requests: write`, or `contents: write` to the agent job.
- Route GitHub writes through `safe-outputs:`.
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

### 7. Decide who can trigger the workflow

- Default behavior is team-only triggering.
- For community-facing issue triage or other public entrypoints, recommend `roles: all`.

### 8. Omit unnecessary defaults

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
