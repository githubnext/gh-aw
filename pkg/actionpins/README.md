# actionpins Package

> GitHub Action pin resolution utilities backed by embedded pin data and optional dynamic SHA resolution.

## Overview

The `actionpins` package resolves `uses:` references like `actions/checkout@v5` to pinned commit SHAs. It loads embedded pin metadata from `data/action_pins.json`, indexes pins by repository, and exposes helpers for formatting and resolving references.

Resolution supports two modes:

- Embedded-only lookup from bundled pin data
- Dynamic lookup via a caller-provided `SHAResolver`, with fallback behavior controlled by `PinContext.StrictMode`

## Public API

### Types

| Type | Kind | Description |
|------|------|-------------|
| `ActionYAMLInput` | struct | Input metadata parsed from an Action's `action.yml` |
| `ActionPin` | struct | Pinned action entry (repo, version, SHA, optional inputs) |
| `ActionPinsData` | struct | JSON container used to load embedded pin entries |
| `SHAResolver` | interface | Resolves a SHA for `repo@version` dynamically |
| `PinContext` | struct | Runtime context for resolution (resolver, strict mode, warning dedupe map) |

### Functions

| Function | Signature | Description |
|----------|-----------|-------------|
| `GetActionPins` | `func() []ActionPin` | Returns all loaded pins |
| `GetActionPinsByRepo` | `func(repo string) []ActionPin` | Returns all pins for a repository (version-descending) |
| `GetActionPinByRepo` | `func(repo string) (ActionPin, bool)` | Returns the latest pin for a repository |
| `FormatReference` | `func(repo, sha, version string) string` | Formats a pinned reference (`repo@sha # version`) |
| `FormatCacheKey` | `func(repo, version string) string` | Formats a cache key (`repo@version`) |
| `ExtractRepo` | `func(uses string) string` | Extracts the repository from a `uses` reference |
| `ExtractVersion` | `func(uses string) string` | Extracts the version from a `uses` reference |
| `ResolveActionPin` | `func(actionRepo, version string, ctx *PinContext) (string, error)` | Resolves a pinned reference with optional dynamic SHA lookup and fallback behavior |
| `GetCachedActionPin` | `func(repo string, ctx *PinContext) string` | Returns a pinned reference preferring cache/dynamic resolution when available |

## Usage Example

```go
ctx := &actionpins.PinContext{StrictMode: true}

reference, err := actionpins.ResolveActionPin("actions/checkout", "v5", ctx)
if err != nil {
	panic(err)
}

fmt.Println(reference) // actions/checkout@<sha> # v5
```

## Dependencies

**Internal**:
- `pkg/console` — warning message formatting
- `pkg/logger` — debug logging
- `pkg/semverutil` — semantic version compatibility checks

## Thread Safety

Embedded pin loading and index creation use `sync.Once`, and read access to loaded pin slices/maps is safe after initialization.

`PinContext.Warnings` is mutated in place for warning deduplication; callers should not share one `PinContext` across goroutines without external synchronization.

---

*This specification is automatically maintained by the [spec-extractor](../../.github/workflows/spec-extractor.md) workflow.*
