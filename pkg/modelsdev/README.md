# modelsdev Package

The `modelsdev` package provides provider and model identifier normalization helpers.

## Overview

This package normalizes provider/model identifiers so callers can perform consistent model matching.

## Public API

### Functions

| Function | Signature | Description |
|----------|-----------|-------------|
| `NormalizeProvider` | `func(provider string) string` | Normalizes provider aliases such as `github`, `copilot`, and `github_models` to `github-copilot`, and lower-cases other provider identifiers |
| `NormalizeComparableModelID` | `func(value string) string` | Lower-cases a model identifier, trims surrounding whitespace, and replaces `.` and `_` with `-` so equivalent model IDs compare consistently |

## Usage Examples

```go
import "github.com/github/gh-aw/pkg/modelsdev"

provider := modelsdev.NormalizeProvider(" github_models ")
comparableModel := modelsdev.NormalizeComparableModelID(" GPT_4.1-mini ")
_, _ = provider, comparableModel
```

## Dependencies

**Internal**:
- None

## Design Notes

- Provider aliases such as `github`, `copilot`, and `github_models` are normalized to `github-copilot`.
- Comparable model matching normalizes separators (`.` and `_` to `-`) to improve lookup robustness.

## Source Synchronization

Reviewed against recent source updates on 2026-07-17; no additional public-contract deltas were identified beyond the sections above.

---

*This specification is automatically maintained by the [spec-extractor](../../.github/workflows/spec-extractor.md) workflow.*
