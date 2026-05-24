# errorutil Package

The `errorutil` package provides shared helpers for classifying and inspecting errors returned by the GitHub API and `gh` CLI.

## Overview

This package currently exposes focused helpers for identifying common error categories used across `pkg/cli` and `pkg/parser`, including "not found" (`404`), "forbidden" (`403`), and "gone" (`410`) responses.

## Public API

### Functions

| Function | Signature | Description |
|----------|-----------|-------------|
| `IsNotFoundError` | `func(err error) bool` | Returns `true` when `err` indicates a "not found" condition by matching case-insensitive `404` or `not found` text; returns `false` for `nil` and non-matching errors |
| `IsForbiddenError` | `func(err error) bool` | Returns `true` when `err` indicates an HTTP-style `403`/"forbidden" response by matching case-insensitive patterns like `HTTP 403` or `403 Forbidden`; returns `false` for `nil` and non-matching errors |
| `IsGoneError` | `func(err error) bool` | Returns `true` when `err` indicates an HTTP-style `410`/"gone" response by matching case-insensitive patterns like `HTTP 410` or `410 Gone`; returns `false` for `nil` and non-matching errors |

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

**Internal**:
- `pkg/logger` — package-scoped logging used for error-classification diagnostics.

**External**:
- None beyond the Go standard library (`strings`).

## Design Notes

- `IsNotFoundError`, `IsForbiddenError`, and `IsGoneError` intentionally accept multiple message formats to cover errors produced by GitHub API responses, `gh` CLI output, and `go-gh` wrappers.
- `IsForbiddenError` and `IsGoneError` intentionally require HTTP-style status context so unrelated phrases like `forbidden character` or `gone away` are not misclassified.

---

*This specification is automatically maintained by the [spec-extractor](../../.github/workflows/spec-extractor.md) workflow.*
