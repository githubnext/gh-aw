---
description: Shared guidance for editing, recompiling, and validating GitHub Agentic Workflow files.
---

# Workflow Editing Basics

Agentic workflows are single markdown files at `.github/workflows/<workflow-id>.md`.

## File Structure

1. **YAML frontmatter** between `---` markers configures triggers, permissions, tools, network, imports, and safe outputs.
2. **Markdown body** after the frontmatter is the agent prompt.

## When Recompilation Is Required

Run `gh aw compile <workflow-id>` after changing frontmatter fields such as:

- `on:`
- `permissions:`
- `tools:`
- `network:`
- `imports:`
- `safe-outputs:`
- `mcp-servers:`
- engine, timeout, concurrency, or other YAML configuration

## When Recompilation Is Not Required

Edit the markdown body directly when changing:

- agent instructions
- task descriptions
- examples
- formatting guidance
- clarifications and guardrails

Markdown body changes take effect on the next run without recompiling.

## Validation Commands

```bash
gh aw compile <workflow-id>
gh aw compile <workflow-id> --strict
gh aw compile --purge
```

Use `--strict` for production-quality validation.

## Editing Rules

- Make the smallest change that satisfies the request.
- Preserve existing structure unless reorganization is part of the task.
- Never leave a workflow in a broken state.
- If compilation fails, fix all errors before stopping.
- After frontmatter changes, review the generated `.lock.yml`.

## Prompt-Authoring Rules

- Keep the prompt specific and imperative.
- Prefer short examples over long tutorials.
- Reference dedicated instruction files instead of duplicating long explanations.
- Tell agents to use `noop` when they complete successfully and no visible action is needed.
