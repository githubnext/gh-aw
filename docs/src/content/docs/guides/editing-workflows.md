---
title: Editing Workflows
description: Learn when you can edit workflows directly on GitHub.com versus when recompilation is required, and best practices for iterating on agentic workflows.
sidebar:
  order: 5
---

Agentic workflows have two parts: the **YAML frontmatter**, which is compiled into the lock file and requires recompilation when changed, and the **markdown body**, which is loaded at runtime and takes effect on the next run. This lets you iterate on instructions quickly while keeping security-sensitive configuration behind compilation.

See [Creating Agentic Workflows](/gh-aw/setup/creating-workflows/) for guidance on creating workflows with AI assistance.

> [!TIP]
> Working in `github/gh-aw` itself? Treat any edit under `.github/workflows/*.md` as a cue to run `make recompile` (a private Makefile target for this repo, not a `gh aw` CLI command) before committing. CI checks the generated `.lock.yml` files for drift, so this is the safest default even when you're only changing markdown instructions. For early feedback while iterating locally, you can run `gh aw compile --watch --schedule-seed github/gh-aw`, then finish with `make recompile`.

## Editing Without Recompilation

You can edit the **markdown body** directly on GitHub.com or in any editor without recompiling, including instructions, output templates, conditional guidance, context, and examples.

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

Apply labels for bugs, enhancements, questions, and documentation updates. For priority, use `high-priority` for security or blocking issues, `medium-priority` for important but non-critical work, and `low-priority` for minor improvements.
```

✅ This change takes effect immediately. Contributors to `github/gh-aw` itself should still run `make recompile` (a private Makefile target for this repo, not a `gh aw` CLI command) before committing so CI sees fresh `.lock.yml` files alongside the markdown change.

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

Markdown can include runtime expressions without recompilation:

```markdown
# Process Issue

Read issue #${{ github.event.issue.number }} in repository ${{ github.repository }}.

Issue title: "${{ github.event.issue.title }}"

Use sanitized content: "${{ steps.sanitized.outputs.text }}"

Actor: ${{ github.actor }}
Repository: ${{ github.repository }}
```

These expressions are evaluated at runtime and validated for security. See [Templating](/gh-aw/reference/templating/) for the complete list of allowed expressions.

Arbitrary expressions are blocked. This will fail at runtime:

```markdown
# ❌ WRONG - Will be rejected
Run this command: ${{ github.event.comment.body }}
```

Use `steps.sanitized.outputs.text` for sanitized user input instead.

## Recompiling with a Stable Schedule Seed

If a workflow uses fuzzy schedules such as `daily`, `weekly`, or `every 2h`, recompilation can change the generated cron output when the compiler derives its scatter seed from repository metadata that differs across clones.

For shared repositories, pass a canonical repository slug with `--schedule-seed` so contributors generate the same cron expressions:

```bash
gh aw compile --schedule-seed github/gh-aw
```

In `github/gh-aw`, this pattern is wrapped by the private `make recompile` Makefile target (not a `gh aw` CLI command):

```bash
make recompile
```

Use a fixed seed whenever deterministic schedule output matters, especially for workflows committed to version control.

## Quick Rule of Thumb

Edit the markdown body for instruction changes, recompile with `gh aw compile` after any frontmatter change, use `--schedule-seed` when fuzzy schedules must stay deterministic across contributors (`github/gh-aw` contributors can use the private `make recompile` target for this), and use sanitized step outputs instead of raw user input in expressions.

## Related Documentation

See [Workflow Structure](/gh-aw/reference/workflow-structure/) for file organization, [Frontmatter Reference](/gh-aw/reference/frontmatter/) for configuration options, [Markdown Reference](/gh-aw/reference/markdown/) for writing instructions, [Compilation Process](/gh-aw/reference/compilation-process/) for compilation details, [Templating](/gh-aw/reference/templating/) for expression syntax, and the [Workshop](https://github.com/githubnext/gh-aw-workshop) for hands-on exercises.
