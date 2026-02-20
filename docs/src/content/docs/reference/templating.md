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

### Automatic Expression Transformations

The compiler automatically transforms certain expressions to ensure they work correctly in the activation job context:

**Activation Output Transformations:**
- `needs.activation.outputs.text` → `steps.sanitized.outputs.text`
- `needs.activation.outputs.title` → `steps.sanitized.outputs.title`
- `needs.activation.outputs.body` → `steps.sanitized.outputs.body`

**Why this transformation occurs:**

The prompt is generated within the activation job, which cannot reference its own `needs.activation.*` outputs (a job cannot reference its own needs outputs in GitHub Actions). Instead, the compiler automatically rewrites these expressions to reference the `sanitized` step within the activation job, which computes sanitized versions of the issue/PR text, title, and body.

**Example:**

```markdown
Analyze this content: "${{ needs.activation.outputs.text }}"
```

Is automatically transformed during compilation to:

```markdown
Analyze this content: "${{ steps.sanitized.outputs.text }}"
```

This transformation is particularly important for [runtime imports](#runtime-imports), which allow you to edit markdown content without recompilation. The compiler ensures all necessary expressions are available for runtime substitution.

:::note
Only `text`, `title`, and `body` outputs are transformed. Other activation outputs like `comment_id` and `comment_repo` are not transformed and remain as `needs.activation.outputs.*`.
:::

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

Runtime imports allow you to include content from files and URLs directly within your workflow prompts **at runtime** during GitHub Actions execution. This differs from [frontmatter imports](/gh-aw/reference/imports/) which are processed at compile-time.

**Security Note:** File imports are **restricted to the `.github` folder** in your repository. This ensures workflow configurations cannot access arbitrary files in your codebase.

Runtime imports use the macro syntax: `{{#runtime-import filepath}}`

The macro supports:
- Line range extraction (e.g., `:10-20` for lines 10-20)
- URL fetching with automatic caching
- Content sanitization (front matter removal, macro detection)
- Automatic `.github/` prefix handling

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

**Path Validation:**

File paths are **restricted to the `.github` folder** to prevent access to arbitrary repository files:

```aw wrap
# ✅ Valid - Files in .github folder
{{#runtime-import shared-instructions.md}}           # Loads .github/shared-instructions.md
{{#runtime-import .github/shared-instructions.md}}  # Same - .github/ prefix is trimmed

# ❌ Invalid - Security violations
{{#runtime-import ../src/config.go}}                # Error: Relative traversal outside .github
{{#runtime-import /etc/passwd}}                     # Error: Absolute path not allowed
```

### Caching

Fetched URLs are cached for 1 hour per workflow run at `/tmp/gh-aw/url-cache/` (keyed by SHA256 hash). The first fetch adds ~500ms–2s latency; subsequent accesses use cached content.

### Processing Order

Runtime imports are processed before other substitutions:

1. `{{#runtime-import}}` macros processed (files and URLs)
2. `${GH_AW_EXPR_*}` variable interpolation
3. `{{#if}}` template conditionals rendered

### Common Use Cases

**Shared instructions from a file:**

```aw wrap
# Code Review Agent

{{#runtime-import workflows/shared/review-standards.md}}
<!-- Loads .github/workflows/shared/review-standards.md -->

Review the pull request changes.
```

**External content from a URL, with line range:**

```aw wrap
# Security Audit

Follow this checklist:

{{#runtime-import https://company.com/security/api-checklist.md}}

Reference implementation (lines 100-150):
{{#runtime-import docs/engine.go:100-150}}
```

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
