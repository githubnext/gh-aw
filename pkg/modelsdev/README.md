# modelsdev Package

The `modelsdev` package provides model pricing lookup backed by the public `models.dev` catalog.

## Overview

This package downloads and parses `https://models.dev/catalog.json`, normalizes provider/model identifiers, and exposes per-token pricing for callers that need cost-aware behavior.

The catalog is loaded once per process via a singleton cache (`syncutil.OnceLoader`) to avoid repeated network fetches.

## Public API

### Functions

| Function | Signature | Description |
|----------|-----------|-------------|
| `FindPricing` | `func(ctx context.Context, provider, model string) (map[string]float64, bool)` | Returns normalized per-token pricing for a provider/model pair. Falls back to cross-provider matching when provider lookup fails. Returns `(nil, false)` when no pricing is available |
| `NormalizeProvider` | `func(provider string) string` | Normalizes provider aliases such as `github`, `copilot`, and `github_models` to `github-copilot`, and lower-cases other provider identifiers |
| `NormalizeComparableModelID` | `func(value string) string` | Lower-cases a model identifier, trims surrounding whitespace, and replaces `.` and `_` with `-` so equivalent model IDs compare consistently |

## Usage Examples

```go
import "github.com/github/gh-aw/pkg/modelsdev"

pricing, ok := modelsdev.FindPricing(ctx, "github", "gpt-4.1")
if !ok {
    // pricing unavailable
    return
}

inputUSD := pricing["input"]  // per token
outputUSD := pricing["output"] // per token
_ = inputUSD
_ = outputUSD
```

## Dependencies

**Internal**:
- `github.com/github/gh-aw/pkg/logger` — debug logging
- `github.com/github/gh-aw/pkg/syncutil` — one-time catalog load/cache primitive

## Design Notes

- Provider aliases such as `github`, `copilot`, and `github_models` are normalized to `github-copilot`.
- Comparable model matching normalizes separators (`.` and `_` to `-`) to improve lookup robustness.
- Numeric catalog costs are interpreted as per-million-token values and converted to per-token units.
- String catalog costs are treated as already normalized per-token values.
- Network or parsing failures degrade gracefully to an empty cache so callers can continue without pricing data.

---

*This specification is automatically maintained by the [spec-extractor](../../.github/workflows/spec-extractor.md) workflow.*
