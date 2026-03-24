---
title: Editing Workflows
description: Learn when you can edit workflows directly on GitHub.com versus when recompilation is required, and best practices for iterating on agentic workflows.
sidebar:
  order: 5
---

Agentic workflows consist of two parts: the **YAML frontmatter** (compiled into the lock file; changes require recompilation) and the **markdown body** (loaded at runtime; changes take effect immediately). This lets you iterate on AI instructions without recompilation while maintaining strict control over security-sensitive configuration.

See [Creating Agentic Workflows](/gh-aw/setup/creating-workflows/) for guidance on creating workflows with AI assistance.

## Editing Without Recompilation

> [!TIP]
> You can edit the **markdown body** directly on GitHub.com or in any editor without recompiling. Changes take effect on the next workflow run.

### What You Can Edit

The markdown body is loaded at runtime from the original `.md` file. You can freely edit task instructions, output templates, conditional logic ("If X, then do Y"), context explanations, and examples.

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
> Changes to the **YAML frontmatter** always require recompilation. These are security-sensitive configuration options.

### What Requires Recompilation

Any changes to the frontmatter configuration between `---` markers:

- **Triggers** (`on:`): Event types, filters, schedules
- **Permissions** (`permissions:`): Repository access levels
- **Tools** (`tools:`): Tool configurations, MCP servers, allowed tools
- **Network** (`network:`): Allowed domains, firewall rules
- **Safe outputs** (`safe-outputs:`): Output types, threat detection
- **MCP Scripts** (`mcp-scripts:`): Custom MCP tools defined inline
- **Runtimes** (`runtimes:`): Node, Python, Go version overrides
- **Imports** (`imports:`): Shared configuration files
- **Custom jobs** (`jobs:`): Additional workflow jobs
- **Engine** (`engine:`): AI engine selection (copilot, claude, codex)
- **Timeout** (`timeout-minutes:`): Maximum execution time
- **Roles** (`roles:`): Permission requirements for actors

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

## Expressions and Environment Variables

### Allowed Expressions

You can safely use these expressions in markdown without recompilation:

```markdown
# Process Issue

Read issue #${{ github.event.issue.number }} in repository ${{ github.repository }}.

Issue title: "${{ github.event.issue.title }}"

Use sanitized content: "${{ steps.sanitized.outputs.text }}"

Actor: ${{ github.actor }}
Repository: ${{ github.repository }}
```

These expressions are evaluated at runtime and validated for security. See [Templating](/gh-aw/reference/templating/) for the complete list of allowed expressions.

### Prohibited Expressions

Arbitrary expressions are blocked for security. This will fail at runtime:

```markdown
# ❌ WRONG - Will be rejected
Run this command: ${{ github.event.comment.body }}
```

Use `steps.sanitized.outputs.text` for sanitized user input instead.

## Related Documentation

- [Workflow Structure](/gh-aw/reference/workflow-structure/) - Overall file organization
- [Frontmatter Reference](/gh-aw/reference/frontmatter/) - All configuration options
- [Markdown Reference](/gh-aw/reference/markdown/) - Writing effective instructions
- [Compilation Process](/gh-aw/reference/compilation-process/) - How compilation works
- [Templating](/gh-aw/reference/templating/) - Expression syntax and substitution
