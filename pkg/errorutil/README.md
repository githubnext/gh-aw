# errorutil Package

The `errorutil` package provides shared helpers for classifying and inspecting errors returned by the GitHub API and `gh` CLI.

## Overview

This package currently exposes focused helpers for identifying common error categories used across `pkg/cli` and `pkg/parser`, including "not found" (`404`), "forbidden" (`403`), "gone" (`410`), rate-limit, and authentication/authorization responses.

## Public API

### Functions

| Function | Signature | Description |
|----------|-----------|-------------|
| `IsNotFoundError` | `func(err error) bool` | Returns `true` when `err` indicates a "not found" condition by matching case-insensitive `404` or `not found` text; returns `false` for `nil` and non-matching errors |
| `IsForbiddenError` | `func(err error) bool` | Returns `true` when `err` indicates an HTTP-style `403`/"forbidden" response by matching case-insensitive patterns like `HTTP 403` or `403 Forbidden`; returns `false` for `nil` and non-matching errors |
| `IsGoneError` | `func(err error) bool` | Returns `true` when `err` indicates an HTTP-style `410`/"gone" response by matching case-insensitive patterns like `HTTP 410` or `410 Gone`; returns `false` for `nil` and non-matching errors |
| `IsRateLimitError` | `func(output string) bool` | Returns `true` when `output` indicates GitHub API rate limiting by matching case-insensitive `rate limit exceeded` (including `API rate limit exceeded`) or `secondary rate limit` text |
| `IsAuthError` | `func(output string) bool` | Returns `true` when `output` indicates authentication or authorization failures by matching case-insensitive credential-specific markers including `GH_TOKEN`, `GITHUB_TOKEN`, `authentication`, `not logged into`, `unauthorized`, `permission denied`, or `SAML enforcement` |

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

if errorutil.IsRateLimitError(output) {
    // Back off and retry
}

if errorutil.IsAuthError(output) {
    // Surface credential guidance
}
```

## Dependencies

**Internal**:
- `github.com/github/gh-aw/pkg/logger` — package-scoped logging used for error-classification diagnostics.

**External**:
- None beyond the Go standard library (`strings`).

## Design Notes

- `IsNotFoundError`, `IsForbiddenError`, and `IsGoneError` intentionally accept multiple message formats to cover errors produced by GitHub API responses, `gh` CLI output, and `go-gh` wrappers.
- `IsRateLimitError` and `IsAuthError` provide shared case-insensitive string classifiers for GitHub API and `gh` CLI output so callers avoid duplicating inline substring checks.
- `IsForbiddenError` and `IsGoneError` intentionally require HTTP-style status context so unrelated phrases like `forbidden character` or `gone away` are not misclassified.

---

*This specification is automatically maintained by the [spec-extractor](../../.github/workflows/spec-extractor.md) workflow.*
