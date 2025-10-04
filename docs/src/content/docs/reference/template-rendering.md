---
title: Template Rendering
description: Conditional content rendering in agentic workflows
sidebar:
  order: 12
---

Template rendering allows you to conditionally include or exclude sections of your agentic workflow prompt based on simple boolean expressions. This feature is processed **after** GitHub Actions interpolates `${{ }}` expressions, enabling dynamic prompt content.

## Overview

The template renderer is a minimalistic, logic-less postprocessor that:
- Processes `{{#if ...}} ... {{/if}}` conditional blocks
- Evaluates expressions as truthy or falsy
- Removes falsy sections, keeps truthy sections
- Preserves all markdown formatting
- Works with any AI engine (Claude, Copilot, Codex, Custom)

## Syntax

Template conditionals use a simple mustache-style syntax:

```markdown
{{#if expression}}
Content to include if expression is truthy
{{/if}}
```

### Truthy and Falsy Values

| Expression | Result |
|------------|--------|
| `true` | Truthy - content is kept |
| `false` | Falsy - content is removed |
| `0` | Falsy - content is removed |
| `null` | Falsy - content is removed |
| `undefined` | Falsy - content is removed |
| `""` (empty string) | Falsy - content is removed |
| Any other value | Truthy - content is kept |

The evaluation is case-insensitive, so `TRUE`, `False`, `NULL`, etc. work as expected.

## How It Works

The template rendering process:

1. **GitHub Actions interpolation**: All `${{ }}` expressions are evaluated first
2. **Pattern detection**: Compiler checks for `{{#if` patterns in the markdown
3. **Conditional step**: If patterns found, a rendering step is automatically added
4. **Evaluation**: Template blocks are processed, keeping truthy content and removing falsy content
5. **Prompt file update**: The rendered markdown is written back to the prompt file

## Examples

### Basic Conditional

```aw
---
on: issues
engine: copilot
---

# Issue Triage

{{#if true}}
## Active Features
These features are currently enabled and ready to use.
{{/if}}

{{#if false}}
## Deprecated Features
This section is hidden from the prompt.
{{/if}}

## Standard Triage Steps
This content is always visible.
```

After rendering, the prompt contains:

```markdown
# Issue Triage

## Active Features
These features are currently enabled and ready to use.

## Standard Triage Steps
This content is always visible.
```

### Dynamic Content with GitHub Expressions

Combine GitHub Actions expressions with template conditionals:

```aw
---
on:
  issues:
    types: [opened]
engine: claude
---

# Issue Analysis

Analyze issue #${{ github.event.issue.number }}.

{{#if ${{ github.event.issue.labels[0].name == 'bug' }}}}
## Bug-Specific Analysis
Focus on error messages, stack traces, and reproduction steps.
{{/if}}

{{#if ${{ github.event.issue.labels[0].name == 'feature' }}}}
## Feature Request Analysis
Evaluate scope, complexity, and alignment with project goals.
{{/if}}
```

In this example, GitHub Actions first evaluates the `${{ }}` expressions to `true` or `false`, then the template renderer processes the conditional blocks.

### Multiple Conditionals

```aw
---
on: pull_request
engine: copilot
tools:
  github:
    allowed: [get_pull_request]
---

# PR Review

{{#if true}}
## Security Checks
- Verify no secrets are committed
- Check for SQL injection vulnerabilities
- Review authentication logic
{{/if}}

{{#if true}}
## Code Quality
- Check for code style consistency
- Verify test coverage
- Review documentation updates
{{/if}}

{{#if false}}
## Legacy Compatibility (Deprecated)
This section is no longer needed.
{{/if}}
```

### Environment-Based Conditionals

Use environment variables to control content:

```aw
---
on: workflow_dispatch
engine: claude
env:
  DETAILED_LOGGING: true
---

# Workflow Task

{{#if ${{ env.DETAILED_LOGGING }}}}
## Debugging Instructions
Enable verbose logging for all operations.
Include stack traces in error messages.
{{/if}}

Proceed with the main task.
```

## Performance

The template rendering step is only added when conditional patterns are detected in the markdown content. If your workflow contains no `{{#if` patterns, no rendering step is generated, keeping the workflow lean.

## Limitations

- **No nesting**: Conditional blocks cannot be nested
- **No else clauses**: Only `{{#if}}` is supported, no `{{else}}` or `{{else if}}`
- **No variables**: No variable substitution beyond GitHub Actions `${{ }}` expressions
- **No loops**: No iteration constructs
- **Text-only evaluation**: Expressions are evaluated as strings (truthy/falsy checks only)

These limitations are intentional to keep the template system simple, safe, and predictable.

## Related Documentation

- [Markdown Content](/gh-aw/reference/markdown/) - Writing effective agentic markdown
- [Workflow Structure](/gh-aw/reference/workflow-structure/) - Overall workflow organization
- [Frontmatter Options](/gh-aw/reference/frontmatter/) - YAML configuration
