# File/URL Inlining Syntax

This document describes the file and URL inlining syntax feature for GitHub Agentic Workflows.

## Overview

The file/URL inlining syntax allows you to include content from files and URLs directly within your workflow prompts at runtime. This provides a convenient way to reference external content without using the `{{#runtime-import}}` macro.

## Syntax

### File Inlining

**Full File**: `@path/to/file.ext`
- Includes the entire content of the file
- Path is relative to `GITHUB_WORKSPACE`
- Example: `@docs/README.md`

**Line Range**: `@path/to/file.ext:start-end`
- Includes specific lines from the file (1-indexed, inclusive)
- Start and end are line numbers
- Example: `@src/main.go:10-20` includes lines 10 through 20

### URL Inlining

**HTTP/HTTPS URLs**: `@https://example.com/file.txt`
- Fetches content from the URL
- Content is cached for 1 hour to reduce network requests
- Cache is stored in `/tmp/gh-aw/url-cache/`
- Example: `@https://raw.githubusercontent.com/owner/repo/main/README.md`

## Features

### Content Sanitization

All inlined content is automatically sanitized:
- **Front matter removal**: YAML front matter (between `---` delimiters) is stripped
- **XML comment removal**: HTML/XML comments (`<!-- ... -->`) are removed
- **GitHub Actions macro detection**: Content containing `${{ ... }}` expressions is rejected with an error

### Email Address Handling

The parser is smart about email addresses:
- `user@example.com` is NOT treated as a file reference
- Only `@path` patterns that look like file paths or URLs are processed

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

@docs/coding-guidelines.md

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

@src/auth.go:45-52

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

File and URL inlining occurs after runtime imports but before variable interpolation:

1. `{{#runtime-import}}` macros are processed
2. `@path` and `@path:line-line` file references are inlined
3. `@https://...` and `@http://...` URL references are fetched and inlined
4. Variable interpolation (`${GH_AW_EXPR_*}`) is performed
5. Template conditionals (`{{#if}}`) are rendered

## Error Handling

### File Not Found
If a referenced file doesn't exist, the workflow will fail with an error:
```
Failed to process inline for @missing.txt: File not found for inline: missing.txt
```

### Invalid Line Range
If line numbers are out of bounds, the workflow will fail:
```
Invalid start line 100 for file src/main.go (total lines: 50)
```

### URL Fetch Failure
If a URL cannot be fetched, the workflow will fail:
```
Failed to process URL inline for @https://example.com/file.txt: Failed to fetch URL https://example.com/file.txt: HTTP 404
```

### GitHub Actions Macros
If inlined content contains GitHub Actions expressions, the workflow will fail:
```
File docs/template.md contains GitHub Actions macros (${{ ... }}) which are not allowed in inline content
```

## Limitations

- Line ranges are applied to the raw file content (before front matter removal)
- URLs are cached for 1 hour; longer caching requires manual workflow re-run
- Large files or URLs may impact workflow performance
- Network errors for URL references will fail the workflow

## Implementation Details

The feature is implemented in two JavaScript modules:

1. **`runtime_import.cjs`**: Core file and URL processing functions
   - `processFileInline()`: Reads and sanitizes file content
   - `processFileInlines()`: Processes all `@path` references
   - `processUrlInline()`: Fetches and sanitizes URL content
   - `processUrlInlines()`: Processes all `@url` references

2. **`interpolate_prompt.cjs`**: Integration with prompt processing
   - Calls file inlining before variable interpolation
   - Calls URL inlining after file inlining
   - Handles async URL fetching

## Testing

The feature includes comprehensive test coverage:
- 82+ unit tests in `runtime_import.test.cjs`
- Tests for full file inlining
- Tests for line range extraction
- Tests for URL fetching and caching
- Tests for error conditions
- Tests for email address filtering
- Tests for content sanitization

## Related Documentation

- Runtime Import Macros: `{{#runtime-import filepath}}`
- Variable Interpolation: `${GH_AW_EXPR_*}`
- Template Conditionals: `{{#if condition}}`
