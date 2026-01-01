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

Agentic workflows restrict expressions in **markdown content** to prevent security vulnerabilities from exposing secrets or environment variables to the LLM.

> **Note**: These restrictions apply only to markdown content. YAML frontmatter can use secrets and environment variables for workflow configuration.

**Permitted expressions** in markdown include:
- Event properties: `github.event.*` (issue/PR numbers, titles, states, SHAs, IDs, etc.)
- Repository context: `github.actor`, `github.owner`, `github.repository`, `github.server_url`, `github.workspace`
- Run metadata: `github.run_id`, `github.run_number`, `github.job`, `github.workflow`
- Pattern expressions: `needs.*`, `steps.*`, `github.event.inputs.*`

### Prohibited Expressions

All other expressions are disallowed, including `secrets.*`, `env.*`, `vars.*`, and complex functions like `toJson()` or `fromJson()`.

Expression safety is validated during compilation. Unauthorized expressions produce errors like:

```text
error: unauthorized expressions: [secrets.TOKEN, env.MY_VAR]. 
allowed: [github.repository, github.actor, github.workflow, ...]
```

## Conditional Markdown

Include or exclude prompt sections based on boolean expressions using `{{#if ...}} ... {{/if}}` blocks.

### Syntax

```markdown wrap
{{#if expression}}
Content to include if expression is truthy
{{/if}}
```

The compiler automatically wraps expressions with `${{ }}` for GitHub Actions evaluation. For example, `{{#if github.event.issue.number}}` becomes `{{#if ${{ github.event.issue.number }} }}`.

**Falsy values:** `false`, `0`, `null`, `undefined`, `""` (empty string)
**Truthy values:** Everything else

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
You are analyzing issue #${{ github.event.issue.number }}.
{{/if}}

{{#if github.event.pull_request.number}}
## Pull Request Analysis
You are analyzing PR #${{ github.event.pull_request.number }}.
{{/if}}
```

### Limitations

The template system supports only basic conditionals—no nesting, `else` clauses, variables, loops, or complex evaluation.

## Related Documentation

- [Markdown](/gh-aw/reference/markdown/) - Writing effective agentic markdown
- [Workflow Structure](/gh-aw/reference/workflow-structure/) - Overall workflow organization
- [Frontmatter](/gh-aw/reference/frontmatter/) - YAML configuration
- [Imports](/gh-aw/reference/imports/) - Imports
