---
description: Update existing agentic workflows using GitHub Agentic Workflows (gh-aw) with concise guidance on minimal changes and validation.
disable-model-invocation: true
---

# GitHub Agentic Workflow Updater

Update existing workflow files in `.github/workflows/`.

## Load These References First

- [github-agentic-workflows.md](github-agentic-workflows.md)
- [workflow-editing.md](workflow-editing.md)
- [workflow-constraints.md](workflow-constraints.md)
- [safe-outputs.md](safe-outputs.md)
- [syntax.md](syntax.md)

Load these additional files only when relevant:

- [campaign.md](campaign.md)
- [experiments.md](experiments.md)
- [visual-regression.md](visual-regression.md)
- [serena-tool.md](serena-tool.md)

## Scope

This prompt is for **updating existing workflows only**. For new workflows, use the creator prompt.

## Start the Conversation

1. Ask which workflow to update.
2. Ask what change is needed.
3. Then inspect the existing file before proposing edits.

## First Decision: Frontmatter or Prompt Body?

Use [workflow-editing.md](workflow-editing.md) as the source of truth.

- frontmatter change → recompilation required
- markdown-body-only change → no recompilation required

## Update Rules

- make the smallest possible change
- preserve existing style and structure unless reorganization is required
- do not rewrite unrelated frontmatter sections
- keep the agent job read-only
- when targeting the Copilot coding agent, recommend `permissions: { copilot-requests: write }` for Copilot authentication
- use `safe-outputs:` for writes
- prefer `toolsets:` for GitHub tools

## Common Update Categories

See [workflow-editing.md](workflow-editing.md) for the full frontmatter-vs-body recompilation taxonomy.

- **Prompt-only updates** (clarifying instructions, tightening wording, adding or removing examples, adding guardrails or output-format guidance): do not recompile; the change applies on the next run.
- **Frontmatter updates** (triggers, permissions, tools and MCP servers, network, safe outputs, imports, timeouts or engine configuration): run `gh aw compile <workflow-id>`, fix every error, then review the `.lock.yml`.

## Cost-Oriented Update Checks

When refining existing workflows, keep edits minimal and confirm the design still follows the [High-Volume Triage and Escalation Pattern](workflow-patterns.md#high-volume-triage-and-escalation-pattern): cheap triage before escalation, `noop`/safe output for known/duplicate/stale cases, frontier reasoning reserved for high-value cases, and context pulled on demand. Keep sub-agent fan-out bounded (see [subagents.md](subagents.md)), then measure the change with `gh aw audit` and treat token or quality regressions as failures (see [token-optimization.md](token-optimization.md)).

## Security Rules

- never suggest GitHub mutation through raw GitHub tools when a safe output exists
- do not recommend `mode: remote` for GitHub tools unless explicitly required and properly configured
- do not replace `pull_request` with `pull_request_target` unless the user explicitly needs a `pull_request_target` design
- do not use `post-steps:` for agent-driven write behavior that belongs in a safe-output job

## Safer-Alternatives Pattern

Follow the "Safer Alternatives First" pattern in [workflow-constraints.md](workflow-constraints.md) when a requested change raises risk.

## Minimal Examples

### Add a GitHub toolset

```yaml
tools:
  github:
    toolsets: [default]
```

### Add a safe output

```yaml
safe-outputs:
  add-comment:
    max: 1
```

### Add network access

```yaml
network:
  allowed:
    - defaults
    - node
```

## Validation Flow

- always inspect the workflow before editing
- compile after frontmatter changes
- keep the workflow valid at every step
- summarize what changed and whether recompilation was needed

## Final Message Rules

At the end, tell the user:

- what changed
- whether the change touched frontmatter or prompt body
- whether recompilation was required
- any next step they should take

Keep the summary short.
