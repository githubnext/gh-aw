---
title: Editing Workflows
description: Learn when you can edit workflows directly on GitHub.com versus when recompilation is required, and best practices for iterating on agentic workflows.
sidebar:
  order: 5
---

Agentic workflows have two parts: the **YAML frontmatter**, which is compiled into the lock file and requires recompilation when changed, and the **markdown body**, which is loaded at runtime and takes effect on the next run. This lets you iterate on instructions quickly while keeping security-sensitive configuration behind compilation.

See [Creating Agentic Workflows](/gh-aw/setup/creating-workflows/) for guidance on creating workflows with AI assistance.

## Editing Without Recompilation

You can edit the **markdown body** directly on GitHub.com or in any editor without recompiling. That includes task instructions, output templates, conditional guidance, context, and examples.

### Example: Adding Instructions

**Before** (in `.github/workflows/issue-triage.md`):
```markdown
---
on:
  issues:
    types: [opened]
---

# Issue Triage

Read issue #${{ github.event.issue.number }} and add appropriate labels.
```

**After** (edited on GitHub.com):
```markdown
---
on:
  issues:
    types: [opened]
---

# Issue Triage

Read issue #${{ github.event.issue.number }} and add appropriate labels.

## Labeling Criteria

Apply these labels based on content:
- `bug`: Issues describing incorrect behavior with reproduction steps
- `enhancement`: Feature requests or improvements
- `question`: Help requests or clarifications needed
- `documentation`: Documentation updates or corrections

For priority, consider:
- `high-priority`: Security issues, critical bugs, blocking issues
- `medium-priority`: Important features, non-critical bugs
- `low-priority`: Nice-to-have improvements, minor enhancements
```

✅ This change takes effect immediately without recompilation.

## Editing With Recompilation Required

> [!WARNING]
> Changes to the **YAML frontmatter** always require recompilation because they affect security-sensitive configuration.

Any change between the `---` markers requires recompilation, including triggers (`on:`), permissions, tools, network settings, safe outputs, MCP scripts, runtimes, imports, custom jobs, engine selection, timeouts, and roles.

### Example: Adding a Tool (Requires Recompilation)

**Before**:
```yaml
---
on:
  issues:
    types: [opened]
---
```

**After** (must recompile):
```yaml
---
on:
  issues:
    types: [opened]

tools:
  github:
    toolsets: [issues]
---
```

⚠️ Run `gh aw compile my-workflow` before committing this change.

## Expressions in Markdown

You can use these expressions in markdown without recompilation:

```markdown
# Process Issue

Read issue #${{ github.event.issue.number }} in repository ${{ github.repository }}.

Issue title: "${{ github.event.issue.title }}"

Use sanitized content: "${{ steps.sanitized.outputs.text }}"

Actor: ${{ github.actor }}
Repository: ${{ github.repository }}
```

These expressions are evaluated at runtime and validated for security. See [Templating](/gh-aw/reference/templating/) for the complete list of allowed expressions.

Arbitrary expressions are blocked for security. This will fail at runtime:

```markdown
# ❌ WRONG - Will be rejected
Run this command: ${{ github.event.comment.body }}
```

Use `steps.sanitized.outputs.text` for sanitized user input instead.

## Recompiling with a Stable Schedule Seed

If a workflow uses fuzzy schedules such as `daily`, `weekly`, or `every 2h`, recompilation can change the generated cron output when the compiler derives its scatter seed from repository metadata that differs across clones.

For shared repositories, pass a canonical repository slug with `--schedule-seed` so every contributor generates the same cron expressions:

```bash
gh aw compile --schedule-seed github/gh-aw
```

The repository `Makefile` uses this pattern in `make recompile`:

```bash
make recompile
```

Use a fixed seed whenever deterministic schedule output matters, especially for workflows committed to version control.

## Quick Rule of Thumb

- Edit the markdown body for instruction changes.
- Recompile after any frontmatter change.
- Use `--schedule-seed` or a project `make recompile` target when fuzzy schedules must compile deterministically across contributors.
- Use sanitized step outputs instead of raw user input in expressions.

## Related Documentation

- [Workflow Structure](/gh-aw/reference/workflow-structure/) - Overall file organization
- [Frontmatter Reference](/gh-aw/reference/frontmatter/) - All configuration options
- [Markdown Reference](/gh-aw/reference/markdown/) - Writing effective instructions
- [Compilation Process](/gh-aw/reference/compilation-process/) - How compilation works
- [Templating](/gh-aw/reference/templating/) - Expression syntax and substitution
