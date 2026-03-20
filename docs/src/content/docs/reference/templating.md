---
title: Templating
description: Expressions and conditional templating in agentic workflows
sidebar:
  order: 350
---

Agentic workflows support four simple templating/substitution mechanisms: 

* GitHub Actions expressions in frontmatter or markdown
* Conditional Templating blocks in markdown
* [Imports](/gh-aw/reference/imports/) in frontmatter or markdown (compile-time)
* Runtime imports in markdown (runtime file/URL inclusion)

## GitHub Actions Expressions

Agentic workflows restrict expressions in **markdown content** to prevent security vulnerabilities from exposing secrets or environment variables to the LLM.

> **Note**: These restrictions apply only to markdown content. YAML frontmatter can use secrets and environment variables for workflow configuration.

**Permitted expressions** in markdown include:
- Event properties: `github.event.*` (issue/PR numbers, titles, states, SHAs, IDs, etc.)
- Repository context: `github.actor`, `github.owner`, `github.repository`, `github.server_url`, `github.workspace`
- Run metadata: `github.run_id`, `github.run_number`, `github.job`, `github.workflow`
- Pattern expressions: `needs.*`, `steps.*`, `github.event.inputs.*`

### Activation Outputs

Use `steps.sanitized.outputs.text/title/body` in your markdown prompts to access sanitized event content:

- `steps.sanitized.outputs.text` — sanitized full context (title + body for issues/PRs, body for comments)
- `steps.sanitized.outputs.title` — sanitized title of the triggering issue or PR
- `steps.sanitized.outputs.body` — sanitized body of the triggering issue or PR

:::caution[Deprecated: `needs.activation.outputs.*`]
Using `${{ needs.activation.outputs.text }}`, `${{ needs.activation.outputs.title }}`, or `${{ needs.activation.outputs.body }}` in workflow markdown is **deprecated**. These expressions still work but produce a deprecation warning during compilation. Use `${{ steps.sanitized.outputs.text }}` etc. directly instead.

**Why:** The prompt is generated _inside_ the activation job, which cannot reference its own `needs.activation.*` outputs in GitHub Actions. The compiler automatically rewrites the deprecated form to `steps.sanitized.outputs.*`, but writing the correct form directly is preferred.
:::

Other activation outputs like `comment_id`, `comment_repo`, and `slash_command` are available as `needs.activation.outputs.*` in _downstream_ jobs (not in the markdown prompt itself).

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

The template system supports only basic conditionals - no nesting, `else` clauses, variables, loops, or complex evaluation.

## Runtime Imports

Runtime imports include content from files and URLs in workflow prompts **at runtime** (unlike [compile-time imports](/gh-aw/reference/imports/)). File paths are restricted to the `.github` folder. Use `{{#runtime-import filepath}}` or `{{#runtime-import? filepath}}` for optional imports.

### Macro Syntax

Use `{{#runtime-import filepath}}` to include file content at runtime. Optional imports use `{{#runtime-import? filepath}}` which don't fail if the file is missing.

**Important:** All file paths are resolved within the `.github` folder. You can specify paths with or without the `.github/` prefix:

```aw wrap
---
on: issues
engine: copilot
---

# Code Review Agent

Follow these coding guidelines:

{{#runtime-import coding-standards.md}}
<!-- Same as: {{#runtime-import .github/coding-standards.md}} -->

Review the code changes and provide feedback.
```

**Line range extraction:**

```aw wrap
# Bug Fix Validator

The original buggy code was (from .github/docs/auth.go):

{{#runtime-import docs/auth.go:45-52}}

Verify the fix addresses the issue.
```

**Optional imports:**

```aw wrap
# Issue Analyzer

{{#runtime-import? shared-instructions.md}}

Analyze issue #${{ github.event.issue.number }}.
```

### URL Imports

The macro syntax supports HTTP/HTTPS URLs. URLs are **not restricted to `.github` folder** and content is cached for 1 hour.

```aw wrap
{{#runtime-import https://raw.githubusercontent.com/org/repo/main/checklist.md}}
{{#runtime-import https://example.com/standards.md:10-50}}
```

### Security Features

All runtime imports include automatic security protections.

**Content Sanitization:** YAML front matter and HTML/XML comments are automatically stripped. GitHub Actions expressions (`${{ ... }}`) are **rejected with error** to prevent template injection and unintended variable expansion.

**Path Validation:** File paths are restricted to the `.github` folder to prevent access to arbitrary repository files. Path traversal and absolute paths are rejected:

```aw wrap
{{#runtime-import ../src/config.go}}  # Error: Relative traversal outside .github
{{#runtime-import /etc/passwd}}       # Error: Absolute path not allowed
```

### Caching

Fetched URLs are cached for 1 hour per workflow run at `/tmp/gh-aw/url-cache/` (keyed by SHA256 hash). The first fetch adds ~500ms–2s latency; subsequent accesses use cached content.

### Processing Order

Runtime imports are processed before other substitutions:

1. `{{#runtime-import}}` macros processed (files and URLs)
2. `${GH_AW_EXPR_*}` variable interpolation
3. `{{#if}}` template conditionals rendered

### Limitations

- **`.github` folder only:** File paths are restricted to `.github` folder for security
- **No authentication:** URL fetching doesn't support private URLs with tokens
- **No recursion:** Imported content cannot contain additional runtime imports
- **Per-run cache:** URL cache doesn't persist across workflow runs
- **Line numbers:** Refer to raw file content before front matter removal

### Error Handling

| Error | Message |
|-------|---------|
| File not found | `Runtime import file not found: missing.txt` |
| Invalid line range | `Invalid start line 100 for file docs/main.go (total lines: 50)` |
| Path traversal | `Security: Path ../src/main.go must be within .github folder` |
| GitHub Actions macros | `File template.md contains GitHub Actions macros (${{ ... }}) which are not allowed in runtime imports` |
| URL fetch failure | `Failed to fetch URL https://example.com/file.txt: HTTP 404` |

## Related Documentation

- [Markdown](/gh-aw/reference/markdown/) - Writing effective agentic markdown
- [Workflow Structure](/gh-aw/reference/workflow-structure/) - Overall workflow organization
- [Frontmatter](/gh-aw/reference/frontmatter/) - YAML configuration
- [Imports](/gh-aw/reference/imports/) - Compile-time imports in frontmatter
