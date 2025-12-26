---
title: Templating
description: Expressions and conditional templating in agentic workflows
sidebar:
  order: 350
---

Agentic workflows support three simple templating/substitution mechanisms: 

* GitHub Actions expressions in frontmatter or markdown
* Conditional Templating blocks in markdown
* [Imports](/gh-aw/reference/imports/) in frontmatter or markdown

## GitHub Actions Expressions

Agentic workflows restrict which GitHub Actions expressions can be used in **markdown content**. This prevents potential security vulnerabilities where access to secrets or environment variables is passed to workflows.

> **Note**: These restrictions apply only to expressions in the markdown content portion of workflows. The YAML frontmatter can still use secrets and environment variables as needed for workflow configuration (e.g., `env:` and authentication).

Permitted GitHub Actions expressions in markdown content include:

**Event properties**: Most `github.event.*` properties (issue/PR numbers, titles, states, SHAs, IDs for comments, deployments, releases, discussions, reviews, workflow runs, etc.)

**Repository context**: `github.actor`, `github.owner`, `github.repository`, `github.server_url`, `github.workspace`, and repository event details

**Run metadata**: `github.run_id`, `github.run_number`, `github.job`, `github.workflow`

**Pattern expressions**: `needs.*` (job outputs), `steps.*` (step outputs), and `github.event.inputs.*` (workflow_dispatch inputs) are also permitted.

### Prohibited Expressions

All other expressions are disallowed, including `secrets.*`, `env.*`, `vars.*`, and complex functions like `toJson()` or `fromJson()`.

Expression safety is validated during compilation. Unauthorized expressions produce errors like:

```text
error: unauthorized expressions: [secrets.TOKEN, env.MY_VAR]. 
allowed: [github.repository, github.actor, github.workflow, ...]
```

## Conditional Markdown

Conditional markdown includes or excludes prompt sections based on boolean expressions. The template renderer processes `{{#if ...}} ... {{/if}}` blocks after GitHub Actions interpolates expressions, evaluating them as truthy or falsy while preserving markdown formatting.

### Syntax

Template conditionals use a simple mustache-style syntax:

```markdown wrap
{{#if expression}}
Content to include if expression is truthy
{{/if}}
```

### Automatic Expression Wrapping

The compiler automatically wraps expressions in `{{#if}}` blocks with `${{ }}` for GitHub Actions evaluation. Writing `{{#if github.event.issue.number}}` becomes `{{#if ${{ github.event.issue.number }} }}` automatically, ensuring runtime values are used rather than literal strings. Expressions already wrapped with `${{` are not double-wrapped.

### Truthy and Falsy Values

Falsy values (remove content): `false`, `0`, `null`, `undefined`, `""` (empty string). All other values are truthy (keep content). Evaluation is case-insensitive.

### How It Works

The compiler wraps expressions with `${{ }}` if needed, detects `{{#if` patterns, and adds a rendering step. During execution, GitHub Actions evaluates expressions, then the renderer processes blocks (keeping truthy content, removing falsy content) and updates the prompt file.

### Example

```aw wrap
---
on:
  issues:
    types: [opened]
---

# Issue Analysis

Analyze issue #${{ github.event.issue.number }}.

{{#if github.event.issue.number}}
## Issue-Specific Analysis
This section appears only when processing an issue.
You are analyzing issue #${{ github.event.issue.number }}.
{{/if}}

{{#if github.event.pull_request.number}}
## Pull Request Analysis
This section appears only when processing a pull request.
You are analyzing PR #${{ github.event.pull_request.number }}.
{{/if}}
```

The compiler wraps expressions automatically, so `{{#if github.event.issue.number}}` becomes `{{#if ${{ github.event.issue.number }} }}` before GitHub Actions evaluates them.

### Limitations

The template system intentionally supports only basic conditionals: no nesting, `else` clauses, variables (beyond `${{ }}`), loops, or complex evaluation. This keeps it simple, safe, and predictable.

## Related Documentation

- [Markdown](/gh-aw/reference/markdown/) - Writing effective agentic markdown
- [Workflow Structure](/gh-aw/reference/workflow-structure/) - Overall workflow organization
- [Frontmatter](/gh-aw/reference/frontmatter/) - YAML configuration
- [Imports](/gh-aw/reference/imports/) - Imports
