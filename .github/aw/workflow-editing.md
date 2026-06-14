---
description: Shared guidance for editing, recompiling, and validating GitHub Agentic Workflow files.
---

# Workflow Editing Basics

Agentic workflows are single markdown files at `.github/workflows/<workflow-id>.md`.

## File Structure

1. **YAML frontmatter** between `---` markers: triggers, permissions, tools, network, imports, safe outputs.
2. **Markdown body**: the agent prompt.

## Recompile When Changing Frontmatter Fields

Run `gh aw compile <workflow-id>` after changing:

- `on:`
- `permissions:`
- `tools:`
- `network:`
- `imports:`
- `safe-outputs:`
- `mcp-servers:`
- engine, timeout, concurrency, or other YAML configuration

## No Recompile Needed

Edit the markdown body directly for:

- agent instructions
- task descriptions
- examples
- formatting guidance
- clarifications and guardrails

Body changes take effect on the next run.

## Validation Commands

```bash
gh aw compile <workflow-id>
gh aw compile <workflow-id> --strict
gh aw compile --purge
```

Use `--strict` for production-quality validation.

## Editing Rules

- Smallest change that satisfies the request.
- Preserve structure unless reorganization is the task.
- Never leave a workflow broken.
- If compile fails, fix all errors before stopping.
- After frontmatter changes, review the generated `.lock.yml`.

## Prompt-Authoring Rules

- Specific and imperative.
- Short examples over long tutorials.
- Reference dedicated instruction files instead of duplicating.
- Tell agents to use `noop` when no visible action is needed.
