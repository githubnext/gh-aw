---
name: workflow-designer
description: Conversational skill that interviews users to design new agentic workflows
disable-model-invocation: true
---

# Workflow Designer

Use this skill to run a structured interview with users who know their goal but not the workflow syntax yet, then generate one complete workflow `.md` file.

## When to Use This Skill

Use this before `create-agentic-workflow.md` when requirements are unclear or incomplete.

- Use `workflow-designer.md` to discover and confirm requirements.
- Use `create-agentic-workflow.md` once requirements are clear and ready for implementation.
- Use `agentic-chat.md` when the user wants a specification/pseudo-code instead of a runnable workflow file.

## Interview Framework

Ask one question at a time. Move to the next phase only after the current phase is clear.

### Phase 1: Goal

Ask: **"What do you want to automate?"**

Capture:
- Workflow name (kebab-case candidate)
- Brief description
- Optional emoji

### Phase 2: Trigger

Ask: **"When should this run?"**

Follow up only if needed:
- Which event type(s)?
- Any filters (labels, branches, commands)?
- Scheduled cadence (daily/weekly/hourly)?

Map to the `on:` block.

### Phase 3: Scope (Read/Write)

Ask:
- **"What should it read?"** (issues, PRs, code, discussions, CI data)
- **"What should it create or update?"** (comments, issues, PRs, labels)

Map to:
- `permissions:` (keep read-only for agent job)
- `tools:`
- `safe-outputs:`

### Phase 4: Guardrails

Ask: **"Should it block merging, just advise, or silently log?"**

Capture:
- Visibility expectations (comment, issue, no visible output)
- No-op behavior expectation

Guide toward safe output behavior and explicit `noop` instructions.

### Phase 5: Context & Network

Ask: **"Does it need external APIs, web access, or package installs?"**

Follow up for exact external domains (FQDN/wildcard).

Map to:
- `network.allowed`
- Optional MCP/GitHub tool usage in `tools:`

### Phase 6: Engine (optional)

Ask only if ambiguous: **"Any AI engine preference?"**

If no preference, suggest default:
- "I'd suggest Copilot since you haven't mentioned a preference. Sound good?"

Map to `engine:` only when not default.

### Phase 7: Confirmation

Present a structured summary and ask for approval before generation.

## Decision Heuristics

### Trigger Mapping

| User says... | Maps to |
|---|---|
| "when someone opens a PR" | `on: pull_request:` with `types: [opened]` |
| "when a PR is updated" | `on: pull_request:` with `types: [opened, synchronize]` |
| "every morning", "daily" | fuzzy `on: schedule: daily` |
| "every Monday", "weekly" | fuzzy `on: schedule: weekly` |
| "when I say /review" | `on: slash_command:` with `name: review` (or requested command) |
| "when an issue is labeled bug" | `on: issues:` with `types: [labeled]` and label filter guidance |
| "manually", "on demand" | `on: workflow_dispatch:` |
| "when a deployment fails" | `on: deployment_status:` |
| "when another workflow finishes" | `on: workflow_run:` |

### Safe Output Mapping

| User says... | Maps to |
|---|---|
| "post a comment" | `add-comment` |
| "create an issue" | `create-issue` |
| "open a PR", "submit changes" | `create-pull-request` |
| "add labels" | `add-labels` |
| "remove labels" | `remove-labels` |
| "close the issue" | `close-issue` |
| "assign someone" | `assign-to-user` |
| "nothing visible", "just analyze" | no safe outputs required |

### Network Mapping

| User says... | Maps to |
|---|---|
| "calls an external API" | ask for exact FQDN/wildcard, then add to `network.allowed` |
| "installs npm packages" | include `node` in `network.allowed` |
| "runs pip install" | include `python` in `network.allowed` |
| "builds Go code" | include `go` in `network.allowed` |
| "no external access" | `network.allowed: [defaults]` (or `[]` if explicitly zero network) |

### Tool Mapping

| User says... | Maps to |
|---|---|
| "read GitHub issues/PRs/workflows" | `tools.github` with `mode: gh-proxy` and minimal `toolsets` |
| "edit files" | `edit` tool (default unless restricted) |
| "run commands/tests" | `bash` tool (default unless restricted) |
| "browse web pages/docs" | `web-fetch` and/or `web-search` |
| "test UI flows" | `playwright` |

## Progressive Disclosure Rules

1. Never dump all options at once; ask one targeted question at a time.
2. Skip questions when answers are inferable from prior user statements.
3. Offer smart defaults and request confirmation instead of over-questioning.
4. Ask at most 5 questions before presenting a summary; then ask "anything else?" if needed.
5. Detect done signals (`that's it`, `looks good`, `generate it`) and proceed to generation.

## Confirmation Format

Use this exact structure:

```text
📋 Proposed workflow:
- Name: <workflow-id>
- Trigger: <event + key options>
- Engine: <engine or default>
- Tools: <tool summary>
- Safe outputs: <list or none>
- Network: <allowed summary>
- Intent: <one-sentence task>
```

Then ask: **"Ready to generate, or want to adjust anything?"**

## Generation Template

After confirmation, generate one workflow file using the same skeleton style as `create-agentic-workflow.md`.

```markdown
---
emoji: <emoji>
description: <brief description>
on:
  <trigger config>
permissions:
  contents: read
  issues: read
  pull-requests: read
tools:
  github:
    mode: gh-proxy
    toolsets: [default]
safe-outputs:
  <safe-output-types-if-needed>
network:
  allowed:
    - defaults
    - <additional entries if needed>
---

# <Workflow Name>

## Task

<clear instructions tied to trigger context>

## Safe Outputs

- Use configured safe outputs for all visible write actions.
- Call `noop` with a short reason when no action is needed.
```

## Validation Checklist

Before final output, verify:

- [ ] Agent job permissions remain read-only (writes only via safe outputs)
- [ ] `safe-outputs:` covers every write action mentioned in prompt/instructions
- [ ] Network access is scoped; avoid blanket wildcard entries
- [ ] Trigger matches the user's intended activation event
- [ ] Prompt instructs agent to call `noop` when no action is needed
- [ ] Unnecessary defaults are omitted (for example `engine: copilot`)

## References (load only when needed)

In-repo references:
- `.github/aw/syntax.md`
- `.github/aw/safe-outputs.md`
- `.github/aw/network.md`
- `.github/aw/patterns.md`
- `.github/aw/triggers.md`
- `.github/aw/create-agentic-workflow.md`

Portable HTTPS references:
- `https://github.com/github/gh-aw/blob/main/.github/aw/syntax.md`
- `https://github.com/github/gh-aw/blob/main/.github/aw/safe-outputs.md`
- `https://github.com/github/gh-aw/blob/main/.github/aw/network.md`
- `https://github.com/github/gh-aw/blob/main/.github/aw/patterns.md`
- `https://github.com/github/gh-aw/blob/main/.github/aw/triggers.md`
