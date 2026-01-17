# Runtime Import Syntax

This document describes the runtime import syntax feature for GitHub Agentic Workflows.

## Overview

The runtime import syntax allows you to include content from files and URLs directly within your workflow prompts at runtime. This provides a convenient way to reference external content using the `{{#runtime-import}}` macro.

**Important:** File paths are resolved within the `.github` folder. Paths are validated to ensure they stay within the git repository root for security.

## Security

**Path Validation**: All file paths are validated to ensure they stay within the `.github` folder:
- Paths are normalized to resolve `.` and `..` components
- After normalization, the resolved path must be within `.github` folder
- Attempts to escape the folder (e.g., `../../../etc/passwd`) are rejected with a security error
- Example: `.github/a/b/../../c/file.txt` is allowed if it resolves to `.github/c/file.txt`

## Syntax

### File Import

**Full File**: `{{#runtime-import filepath}}`
- Includes the entire content of the file from `.github` folder
- Path can be specified with or without `.github/` prefix
- Example: `{{#runtime-import docs/README.md}}` or `{{#runtime-import .github/docs/README.md}}`

**Line Range**: `{{#runtime-import filepath:start-end}}`
- Includes specific lines from the file (1-indexed, inclusive)
- Start and end are line numbers
- Example: `{{#runtime-import src/main.go:10-20}}` includes lines 10 through 20

### URL Import

**HTTP/HTTPS URLs**: `{{#runtime-import https://example.com/file.txt}}`
- Fetches content from the URL
- Content is cached for 1 hour to reduce network requests
- Cache is stored in `/tmp/gh-aw/url-cache/`
- Example: `{{#runtime-import https://raw.githubusercontent.com/owner/repo/main/README.md}}`

## Features

### Content Sanitization

All imported content is automatically sanitized:
- **Front matter removal**: YAML front matter (between `---` delimiters) is stripped
- **XML comment removal**: HTML/XML comments (`<!-- ... -->`) are removed
- **GitHub Actions macro detection**: Content containing `${{ ... }}` expressions is rejected with an error

## Examples

### Example 1: Include Documentation

```markdown
---
description: Code review workflow
on: pull_request
engine: copilot
---

# Code Review Agent

Please review the following code changes.

## Coding Guidelines

{{#runtime-import docs/coding-guidelines.md}}

## Changes Summary

Review the changes and provide feedback.
```

### Example 2: Include Specific Lines

```markdown
---
description: Bug fix validator
on: pull_request
engine: copilot
---

# Bug Fix Validator

The original buggy code was:

{{#runtime-import src/auth.go:45-52}}

Verify that the fix addresses the issue.
```

### Example 3: External Checklist

```markdown
---
description: Security review
on: pull_request
engine: copilot
---

# Security Review

Follow this security checklist:

{{#runtime-import https://raw.githubusercontent.com/org/security/main/checklist.md}}

Review the changes for security vulnerabilities.
```

Verify the fix addresses the issue.
```

### Example 3: Include Remote Content

```markdown
---
description: Security check
on: pull_request
engine: copilot
---

# Security Review

Follow these security guidelines:

@https://raw.githubusercontent.com/organization/security-guidelines/main/checklist.md

Review all code changes for security vulnerabilities.
```

## Processing Order

File and URL inlining occurs as part of the runtime import system:

1. `@./path` and `@url` references are converted to `{{#runtime-import}}` macros
2. All `{{#runtime-import}}` macros are processed (files and URLs together)
3. `${GH_AW_EXPR_*}` variable interpolation occurs
4. `{{#if}}` template conditionals are rendered

The `@` syntax is pure syntactic sugar - it simply converts to `{{#runtime-import}}` before processing.

## Error Handling

### File Not Found
If a referenced file doesn't exist, the workflow will fail with an error:
```
Failed to process runtime import for ./missing.txt: Runtime import file not found: ./missing.txt
```

### Invalid Line Range
If line numbers are out of bounds, the workflow will fail:
```
Invalid start line 100 for file ./src/main.go (total lines: 50)
```

### Invalid Path Format
If a file path doesn't start with `./` or `../`, it will be ignored:
```
@docs/file.md  # NOT processed - stays as plain text
@./docs/file.md  # Processed correctly
```

### Path Security Violation
If a path tries to escape the git root, the workflow will fail:
```
Security: Path ../../../etc/passwd resolves outside git root (/workspace)
```

### URL Fetch Failure
If a URL cannot be fetched, the workflow will fail:
```
Failed to process runtime import for https://example.com/file.txt: Failed to fetch URL https://example.com/file.txt: HTTP 404
```

### GitHub Actions Macros
If inlined content contains GitHub Actions expressions, the workflow will fail:
```
File ./docs/template.md contains GitHub Actions macros (${{ ... }}) which are not allowed in runtime imports
```

## Limitations

- File paths MUST start with `./` or `../` - paths without these prefixes are ignored
- Resolved paths must stay within the git repository root (enforced via security checks)
- Path normalization is performed to resolve `.` and `..` components before validation
- Line ranges are applied to the raw file content (before front matter removal)
- URLs are cached for 1 hour; longer caching requires manual workflow re-run
- Large files or URLs may impact workflow performance
- Network errors for URL references will fail the workflow

## Implementation Details

The feature is implemented using a unified runtime import system with security validation:

1. **`convertInlinesToMacros()`**: Converts `@./path` and `@url` to `{{#runtime-import}}` macros
2. **`processRuntimeImport()`**: Handles both files and URLs with sanitization and security checks
   - For files: Resolves and normalizes path, validates it stays within git root
   - For URLs: Fetches content with caching
3. **`processRuntimeImports()`**: Processes all runtime-import macros (async)
2. **`processRuntimeImport()`**: Handles both files and URLs with sanitization
3. **`processRuntimeImports()`**: Processes all runtime-import macros (async)

The `@` syntax is pure syntactic sugar that converts to `{{#runtime-import}}` macros.

## Testing

The feature includes comprehensive test coverage:
- 75+ unit tests in `runtime_import.test.cjs`
- Tests for full file inlining with `./` and `../` prefixes
- Tests for line range extraction
- Tests for URL fetching and caching
- Tests for error conditions
- Tests for email address filtering
- Tests for content sanitization

## Related Documentation

- Runtime Import Macros: `{{#runtime-import filepath}}`
- Variable Interpolation: `${GH_AW_EXPR_*}`
- Template Conditionals: `{{#if condition}}`
