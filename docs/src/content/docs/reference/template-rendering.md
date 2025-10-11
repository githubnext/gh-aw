---
title: Template Rendering
description: Conditional content rendering in agentic workflows
sidebar:
  order: 1200
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

### Automatic Expression Wrapping

The compiler automatically wraps expressions in template conditionals with `${{ }}` so they are evaluated by GitHub Actions before template rendering. This means you can write:

```markdown
{{#if github.event.issue.number}}
This appears only when there's an issue number
{{/if}}
```

And the compiler automatically converts it to:

```markdown
{{#if ${{ github.event.issue.number }} }}
This appears only when there's an issue number
{{/if}}
```

This ensures expressions are evaluated to their actual runtime values (e.g., "123" or empty string) rather than being treated as literal strings.

**Key Points:**
- Any expression in `{{#if ...}}` that doesn't already start with `${{` is automatically wrapped
- Prevents double-wrapping: `{{#if ${{ github.actor }} }}` remains unchanged
- Works with all expression types: GitHub context, needs, steps, env, and literals

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

1. **Expression wrapping**: The compiler automatically wraps expressions in `{{#if}}` blocks with `${{ }}` if they don't already start with `${{`
2. **GitHub Actions interpolation**: All `${{ }}` expressions are evaluated during workflow execution
3. **Pattern detection**: The compiler checks for `{{#if` patterns in the markdown
4. **Conditional step**: If patterns are found, a rendering step is automatically added
5. **Evaluation**: Template blocks are processed, keeping truthy content and removing falsy content
6. **Prompt file update**: The rendered markdown is written back to the prompt file

## Examples

### Dynamic Content with GitHub Expressions

The compiler automatically wraps GitHub expressions in template conditionals, so you can write them naturally:

```aw
---
on:
  issues:
    types: [opened]
engine: claude
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

The compiler automatically wraps the expressions in `{{#if}}` blocks with `${{ }}`, so `{{#if github.event.issue.number}}` becomes `{{#if ${{ github.event.issue.number }} }}`. GitHub Actions then evaluates these expressions to their actual values before the template renderer processes the conditionals.

### Multiple Conditionals with Workflow Inputs

Use workflow_dispatch inputs to control conditional content:

```aw
---
on: 
  workflow_dispatch:
    inputs:
      security_review:
        description: 'Include security review'
        required: false
        type: boolean
      code_quality:
        description: 'Include code quality checks'
        required: false
        type: boolean
engine: copilot
tools:
  github:
    allowed: [get_pull_request]
---

# PR Review

{{#if github.event.inputs.security_review}}
## Security Checks
- Verify no secrets are committed
- Check for SQL injection vulnerabilities
- Review authentication logic
{{/if}}

{{#if github.event.inputs.code_quality}}
## Code Quality
- Check for code style consistency
- Verify test coverage
- Review documentation updates
{{/if}}

## Standard Review
Always perform these basic checks regardless of inputs.
```

The compiler wraps these expressions automatically, so you don't need to write `{{#if ${{ ... }} }}`.

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

{{#if env.DETAILED_LOGGING}}
## Debugging Instructions
Enable verbose logging for all operations.
Include stack traces in error messages.
{{/if}}

Proceed with the main task.
```

Again, the compiler automatically wraps `env.DETAILED_LOGGING` with `${{ }}` for you.

## Performance

The template rendering step is only added when conditional patterns are detected in the markdown content. If your workflow contains no `{{#if` patterns, no rendering step is generated, keeping the workflow lean.

## Advanced: Manual Expression Wrapping

While the compiler automatically wraps simple expressions, you may want to manually use `${{ }}` for complex expressions or comparisons:

```aw
{{#if ${{ github.event.issue.labels[0].name == 'bug' }}}}
## Bug-Specific Analysis
This appears when the first label is 'bug'
{{/if}}

{{#if ${{ contains(github.event.issue.labels.*.name, 'security') }}}}
## Security Review
This appears when any label contains 'security'
{{/if}}
```

For these complex expressions with operators or functions, you should manually wrap them in `${{ }}` to ensure proper evaluation.

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
