# ctxutil Package

> Small helpers for consistent `context.Context` handling.

## Overview

The `ctxutil` package centralizes context fallback conventions used across the repository. It currently exposes a single helper, `OrBackground`, for call sites that accept an optional context and need a non-nil `context.Context` before invoking context-aware APIs.

## Public API

### Functions

| Function | Signature | Description |
|----------|-----------|-------------|
| `OrBackground` | `func(ctx context.Context) context.Context` | Returns `context.Background()` when `ctx` is nil; otherwise returns `ctx` unchanged |

## Usage Examples

```go
ctx := ctxutil.OrBackground(ctx)
```

## Dependencies

**External**:
- None beyond the Go standard library (`context`).

---

*This specification is automatically maintained by the [spec-extractor](../../.github/workflows/spec-extractor.md) workflow.*
