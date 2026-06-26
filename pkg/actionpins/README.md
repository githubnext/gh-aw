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
| `ContainerPin` | struct | Pinned container image entry (image, digest, pinned image reference) |
| `ActionPinsData` | struct | JSON container used to load embedded pin entries |
| `SHAResolver` | interface | Resolves a SHA for `repo@version` dynamically |
| `ResolutionErrorType` | string | Classifies unresolved action-ref pinning outcomes for auditing |
| `ResolutionFailure` | struct | Captures an unresolved action-ref pinning event (repo, ref, error type) |
| `PinContext` | struct | Runtime context for resolution (resolver, strict mode, warning dedupe map, action-pin mappings) |

### Functions

| Function | Signature | Description |
|----------|-----------|-------------|
| `GetActionPinsByRepo` | `func(repo string) []ActionPin` | Returns all pins for a repository (version-descending) |
| `GetLatestActionPinByRepo` | `func(repo string) (ActionPin, bool)` | Returns the latest pin for a repository |
| `GetContainerPin` | `func(image string) (ContainerPin, bool)` | Returns a pinned container image by its original image reference |
| `FormatPinnedActionReference` | `func(repo, sha, version string) string` | Formats a pinned action reference string (`repo@sha # version`) |
| `FormatCacheKey` | `func(repo, version string) string` | Formats a cache key (`repo@version`) |
| `ExtractRepo` | `func(uses string) string` | Extracts the repository from a `uses` reference |
| `ExtractVersion` | `func(uses string) string` | Extracts the version from a `uses` reference |
| `ResolveActionPin` | `func(actionRepo, version string, ctx *PinContext) (string, error)` | Resolves a pinned reference with optional dynamic SHA lookup and fallback behavior |
| `ResolveLatestActionPin` | `func(repo string, ctx *PinContext) string` | Resolves a pinned reference for the latest known version, preferring cache/dynamic resolution when available |

## Usage Examples

```go
ctx := &actionpins.PinContext{StrictMode: true}

reference, err := actionpins.ResolveActionPin("actions/checkout", "v5", ctx)
if err != nil {
	panic(err)
}

fmt.Println(reference) // actions/checkout@<sha> # v5
```

### Auditing Resolution Failures

`ResolutionErrorType` classifies why a pin could not be resolved. The two defined values are:

| Constant | Value | Description |
|----------|-------|-------------|
| `ResolutionErrorTypeDynamicResolutionFailed` | `"dynamic_resolution_failed"` | Dynamic tag/ref → SHA resolution failed |
| `ResolutionErrorTypePinNotFound` | `"pin_not_found"` | No usable hardcoded pin was found for the ref |

Use `PinContext.RecordResolutionFailure` to collect `ResolutionFailure` events for auditing:

```go
var failures []actionpins.ResolutionFailure
ctx := &actionpins.PinContext{
    RecordResolutionFailure: func(f actionpins.ResolutionFailure) {
        failures = append(failures, f)
    },
}
actionpins.ResolveActionPin("unknown/action", "v1", ctx)
// failures[0].ErrorType == actionpins.ResolutionErrorTypePinNotFound
```

### Action Pin Mappings

`PinContext.Mappings` redirects `owner/repo@ref` references to replacement references before pin resolution. This is used to substitute private or mirror repositories for well-known public actions (set from `aw.json` `action_pins`).

Keys and values use the format `"owner/repo@ref"`. When a key matches the incoming `actionRepo@version`, resolution proceeds against the mapped value instead. An informational message is emitted once per mapping via `PinContext.Warnings`.

```go
ctx := &actionpins.PinContext{
    Warnings: make(map[string]bool),
    Mappings: map[string]string{
        "actions/checkout@v4": "acme-corp/checkout@v4",
    },
}
reference, err := actionpins.ResolveActionPin("actions/checkout", "v4", ctx)
// reference resolves against acme-corp/checkout@v4 pins
```

### Container Pins

`ContainerPin` provides a pinned image reference for container images:

```go
pin, ok := actionpins.GetContainerPin("ghcr.io/some/image:v1")
if ok {
    fmt.Println(pin.PinnedImage) // ghcr.io/some/image@sha256:abc123
}
```

## Dependencies

**Internal**:
- `github.com/github/gh-aw/pkg/console` — warning message formatting
- `github.com/github/gh-aw/pkg/gitutil` — dynamic SHA resolution via GitHub API/CLI helpers
- `github.com/github/gh-aw/pkg/logger` — debug logging
- `github.com/github/gh-aw/pkg/semverutil` — semantic version compatibility checks

**Test-only**:
- `github.com/github/gh-aw/pkg/constants` — engine names, job names, and version constants used by package tests

## Thread Safety

Embedded pin loading and index creation use `sync.Once`, and read access to loaded pin slices/maps is safe after initialization.

`PinContext.Warnings` is mutated in place for warning deduplication; callers should not share one `PinContext` across goroutines without external synchronization.

---

*This specification is automatically maintained by the [spec-extractor](../../.github/workflows/spec-extractor.md) workflow.*
