# validationerror Package

> Shared payload and formatting helpers for structured validation errors.

## Overview

The `validationerror` package centralizes field, value, reason, and suggestion details for validation errors. Packages such as `parser` and `workflow` embed `Payload` in their concrete error types, allowing callers to inspect them uniformly with `errors.As` and `ValidationError`.

## Public API

### Types

| Type | Description |
|------|-------------|
| `Payload` | Validation details containing the exported `Field`, `Value`, `Reason`, and `Suggestion` string fields |
| `ValidationError` | Interface for errors exposing `ValidationField`, `ValidationValue`, `ValidationReason`, and `ValidationSuggestion` |

### Constants

| Constant | Type | Value | Description |
|----------|------|-------|-------------|
| `MaxValueLength` | `int` | `100` | Maximum value length used by `Format` when truncation is enabled |

### Functions and methods

| Function | Signature | Description |
|----------|-----------|-------------|
| `Format` | `func Format(header string, p Payload, truncateValue bool) string` | Formats a header and payload as a deterministic multi-line message |
| `Payload.ValidationField` | `func (p Payload) ValidationField() string` | Returns the field that failed validation |
| `Payload.ValidationValue` | `func (p Payload) ValidationValue() string` | Returns the offending value, if present |
| `Payload.ValidationReason` | `func (p Payload) ValidationReason() string` | Returns the reason validation failed |
| `Payload.ValidationSuggestion` | `func (p Payload) ValidationSuggestion() string` | Returns remediation guidance, if present |

`Format` always includes the header and reason. It includes the value and suggestion only when they are non-empty. When `truncateValue` is true, the value is truncated to `MaxValueLength`.

## Usage Example

```go
var validationErr validationerror.ValidationError
if errors.As(err, &validationErr) {
    fmt.Printf("%s: %s\n", validationErr.ValidationField(), validationErr.ValidationReason())
}
```

## Dependencies

**Internal**:
- `pkg/stringutil` — value truncation

**External**:
- None beyond the Go standard library.

---

*This specification is automatically maintained by the [spec-extractor](../../.github/workflows/spec-extractor.md) workflow.*
