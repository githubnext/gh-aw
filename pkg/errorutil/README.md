# errorutil Package

The `errorutil` package provides shared helpers for classifying and inspecting errors returned by the GitHub API and `gh` CLI.

## Overview

This package currently exposes focused helpers for identifying common error categories used across `pkg/cli` and `pkg/parser`, including "not found" (`404`), "forbidden" (`403`), and "gone" (`410`) responses.

## Public API

### Functions

| Function | Signature | Description |
|----------|-----------|-------------|
| `IsNotFoundError` | `func(err error) bool` | Returns `true` when `err` indicates a "not found" condition by matching case-insensitive `404` or `not found` text; returns `false` for `nil` and non-matching errors |
| `IsForbiddenError` | `func(err error) bool` | Returns `true` when `err` indicates a "forbidden" condition by matching case-insensitive `403` or `forbidden` text; returns `false` for `nil` and non-matching errors |
| `IsGoneError` | `func(err error) bool` | Returns `true` when `err` indicates a "gone" condition by matching case-insensitive `410` or `gone` text; returns `false` for `nil` and non-matching errors |

## Usage Examples

```go
import "github.com/github/gh-aw/pkg/errorutil"

if errorutil.IsNotFoundError(err) {
    // Handle missing resource path
}

if errorutil.IsForbiddenError(err) {
    // Handle insufficient permissions
}

if errorutil.IsGoneError(err) {
    // Handle expired or deleted resource
}
```

## Dependencies

None.

## Design Notes

- `IsNotFoundError`, `IsForbiddenError`, and `IsGoneError` intentionally accept multiple message formats to cover errors produced by GitHub API responses, `gh` CLI output, and `go-gh` wrappers.

---

*This specification is automatically maintained by the [spec-extractor](../../.github/workflows/spec-extractor.md) workflow.*
