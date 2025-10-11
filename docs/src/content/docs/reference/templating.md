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

The following GitHub Actions context expressions are permitted in the markdown content:

- `${{ github.event.after }}` - The SHA of the most recent commit on the ref after the push
- `${{ github.event.before }}` - The SHA of the most recent commit on the ref before the push
- `${{ github.event.check_run.id }}` - The ID of the check run that triggered the workflow
- `${{ github.event.check_run.number }}` - The number of the check run that triggered the workflow
- `${{ github.event.check_suite.id }}` - The ID of the check suite that triggered the workflow
- `${{ github.event.check_suite.number }}` - The number of the check suite that triggered the workflow
- `${{ github.event.comment.id }}` - The ID of the comment that triggered the workflow
- `${{ github.event.deployment.id }}` - The ID of the deployment that triggered the workflow
- `${{ github.event.deployment.environment }}` - The environment name of the deployment that triggered the workflow
- `${{ github.event.deployment_status.id }}` - The ID of the deployment status that triggered the workflow
- `${{ github.event.head_commit.id }}` - The ID of the head commit for the push event
- `${{ github.event.installation.id }}` - The ID of the GitHub App installation
- `${{ github.event.issue.number }}` - The number of the issue that triggered the workflow
- `${{ github.event.issue.state }}` - The state of the issue (open or closed)
- `${{ github.event.issue.title }}` - The title of the issue that triggered the workflow
- `${{ github.event.discussion.number }}` - The number of the discussion that triggered the workflow
- `${{ github.event.discussion.title }}` - The title of the discussion that triggered the workflow
- `${{ github.event.discussion.category.name }}` - The category name of the discussion
- `${{ github.event.label.id }}` - The ID of the label that triggered the workflow
- `${{ github.event.milestone.id }}` - The ID of the milestone that triggered the workflow
- `${{ github.event.milestone.number }}` - The number of the milestone that triggered the workflow
- `${{ github.event.organization.id }}` - The ID of the organization that triggered the workflow
- `${{ github.event.page.id }}` - The ID of the page build that triggered the workflow
- `${{ github.event.project.id }}` - The ID of the project that triggered the workflow
- `${{ github.event.project_card.id }}` - The ID of the project card that triggered the workflow
- `${{ github.event.project_column.id }}` - The ID of the project column that triggered the workflow
- `${{ github.event.pull_request.number }}` - The number of the pull request that triggered the workflow
- `${{ github.event.pull_request.state }}` - The state of the pull request (open or closed)
- `${{ github.event.pull_request.title }}` - The title of the pull request that triggered the workflow
- `${{ github.event.pull_request.head.sha }}` - The SHA of the head commit of the pull request
- `${{ github.event.pull_request.base.sha }}` - The SHA of the base commit of the pull request
- `${{ github.event.release.assets[0].id }}` - The ID of the first asset in a release
- `${{ github.event.release.id }}` - The ID of the release that triggered the workflow
- `${{ github.event.release.name }}` - The name of the release that triggered the workflow
- `${{ github.event.release.tag_name }}` - The tag name of the release that triggered the workflow
- `${{ github.event.repository.id }}` - The ID of the repository that triggered the workflow
- `${{ github.event.repository.default_branch }}` - The default branch of the repository
- `${{ github.event.review.id }}` - The ID of the pull request review that triggered the workflow
- `${{ github.event.review_comment.id }}` - The ID of the review comment that triggered the workflow
- `${{ github.event.sender.id }}` - The ID of the user who triggered the workflow
- `${{ github.event.workflow_job.id }}` - The ID of the workflow job that triggered the current workflow
- `${{ github.event.workflow_job.run_id }}` - The run ID of the workflow job that triggered the current workflow
- `${{ github.event.workflow_run.id }}` - The ID of the workflow run that triggered the current workflow
- `${{ github.event.workflow_run.number }}` - The number of the workflow run that triggered the current workflow
- `${{ github.event.workflow_run.conclusion }}` - The conclusion of the workflow run that triggered the current workflow
- `${{ github.event.workflow_run.html_url }}` - The URL of the workflow run that triggered the current workflow
- `${{ github.event.workflow_run.head_sha }}` - The head SHA of the workflow run that triggered the current workflow
- `${{ github.event.workflow_run.run_number }}` - The run number of the workflow run that triggered the current workflow
- `${{ github.event.workflow_run.event }}` - The event that triggered the workflow run that triggered the current workflow
- `${{ github.event.workflow_run.status }}` - The status of the workflow run that triggered the current workflow
- `${{ github.actor }}` - The username of the user who triggered the workflow
- `${{ github.job }}` - Job ID of the current workflow run
- `${{ github.owner }}` - The owner of the repository (user or organization name)
- `${{ github.repository }}` - The owner and repository name (e.g., `octocat/Hello-World`)
- `${{ github.run_id }}` - A unique number for each workflow run within a repository
- `${{ github.run_number }}` - A unique number for each run of a particular workflow in a repository
- `${{ github.server_url }}` - Base URL of the server, e.g. https://github.com
- `${{ github.workflow }}` - The name of the workflow
- `${{ github.workspace }}` - The default working directory on the runner for steps

### Special Pattern Expressions
- `${{ needs.* }}` - Any outputs from previous jobs (e.g., `${{ needs.activation.outputs.text }}`)
- `${{ steps.* }}` - Any outputs from previous steps in the same job
- `${{ github.event.inputs.* }}` - Any workflow inputs when triggered by workflow_dispatch (e.g., `${{ github.event.inputs.name }}`)

### Prohibited Expressions

All other expressions are disallowed, including:
- `${{ secrets.* }}` - All secrets
- `${{ env.* }}` - All environment variables
- `${{ vars.* }}` - All repository variables
- Complex functions like `${{ toJson(...) }}`, `${{ fromJson(...) }}`, etc.

Expression safety is validated during compilation with `gh aw compile`. If unauthorized expressions are found, you'll see an error like:

```
error: unauthorized expressions: [secrets.TOKEN, env.MY_VAR]. 
allowed: [github.repository, github.actor, github.workflow, ...]
```

## Conditional Markdown

Conditional markdown allows you to conditionally include or exclude sections of your agentic workflow prompt based on simple boolean expressions. This feature is processed **after** GitHub Actions interpolates `${{ }}` expressions, enabling dynamic prompt content.

The template renderer is a minimalistic, logic-less postprocessor that:
- Processes `{{#if ...}} ... {{/if}}` conditional blocks
- Evaluates expressions as truthy or falsy
- Removes falsy sections, keeps truthy sections
- Preserves all markdown formatting
- Works with any AI engine (Claude, Copilot, Codex, Custom)

### Syntax

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

### How It Works

The template rendering process:

1. **Expression wrapping**: The compiler automatically wraps expressions in `{{#if}}` blocks with `${{ }}` if they don't already start with `${{`
2. **GitHub Actions interpolation**: All `${{ }}` expressions are evaluated during workflow execution
3. **Pattern detection**: The compiler checks for `{{#if` patterns in the markdown
4. **Conditional step**: If patterns are found, a rendering step is automatically added
5. **Evaluation**: Template blocks are processed, keeping truthy content and removing falsy content
6. **Prompt file update**: The rendered markdown is written back to the prompt file

### Example: Dynamic Content with GitHub Expressions

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

### Limitations

- **No nesting**: Conditional blocks cannot be nested
- **No else clauses**: Only `{{#if}}` is supported, no `{{else}}` or `{{else if}}`
- **No variables**: No variable substitution beyond GitHub Actions `${{ }}` expressions
- **No loops**: No iteration constructs
- **Text-only evaluation**: Expressions are evaluated as strings (truthy/falsy checks only)

These limitations are intentional to keep the template system simple, safe, and predictable.

## Related Documentation

- [Markdown](/gh-aw/reference/markdown/) - Writing effective agentic markdown
- [Workflow Structure](/gh-aw/reference/workflow-structure/) - Overall workflow organization
- [Frontmatter](/gh-aw/reference/frontmatter/) - YAML configuration
- [Imports](/gh-aw/reference/imports/) - Imports
