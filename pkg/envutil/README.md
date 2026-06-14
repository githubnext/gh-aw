# envutil Package

> Reads and validates integer-valued environment variables with bounds checking.

## Overview

The `envutil` package centralizes the pattern of reading integer-valued environment variables, validating them against configured minimum and maximum bounds, and falling back to a default value when the variable is absent or out of range. It emits warning messages to stderr when an invalid value is encountered, following the console formatting conventions of the rest of the codebase.

The package exposes a single generic helper, `GetIntFromEnv`, which encapsulates the repetitive read-parse-validate-default logic required wherever configurable integer parameters are exposed via environment variables.

## Public API

### Functions

| Function | Signature | Description |
|----------|-----------|-------------|
| `GetIntFromEnv` | `func(envVar string, defaultValue, minValue, maxValue int, debugLog *logger.Logger) int` | Reads an integer-valued environment variable, validates it, and returns a default when absent or invalid |

#### `GetIntFromEnv`

```go
func GetIntFromEnv(envVar string, defaultValue, minValue, maxValue int, debugLog *logger.Logger) int
```

Reads `envVar` from the process environment, parses it as an integer, validates it against `[minValue, maxValue]`, and returns `defaultValue` when the variable is absent, unparseable, or out of range.

| Parameter | Type | Description |
|-----------|------|-------------|
| `envVar` | `string` | Environment variable name (e.g. `"GH_AW_TIMEOUT"`) |
| `defaultValue` | `int` | Value returned when env var is absent or invalid |
| `minValue` | `int` | Minimum allowed value (inclusive) |
| `maxValue` | `int` | Maximum allowed value (inclusive) |
| `debugLog` | `*logger.Logger` | Optional logger for debug output; pass `nil` to disable |

**Behavioral contract**:
- MUST return `defaultValue` when the environment variable is not set (empty string).
- MUST return `defaultValue` and emit a warning when the value cannot be parsed as an integer.
- MUST return `defaultValue` and emit a warning when the value is outside `[minValue, maxValue]` (bounds are inclusive).
- MUST log the accepted value via `debugLog` when `debugLog` is non-nil.
- SHOULD emit warnings formatted via `console.FormatWarningMessage` to `os.Stderr` when `debugLog` is `nil`.
- MAY route warnings through `debugLog.Printf` when `debugLog` is non-nil instead of writing directly to stderr.

## Usage Examples

```go
import (
    "github.com/github/gh-aw/pkg/envutil"
    "github.com/github/gh-aw/pkg/logger"
)

var log = logger.New("mypackage:config")

// Read GH_AW_MAX_CONCURRENT_DOWNLOADS, constrained to [1, 20], default 5
concurrency := envutil.GetIntFromEnv("GH_AW_MAX_CONCURRENT_DOWNLOADS", 5, 1, 20, log)

// Suppress debug output by passing nil logger
timeout := envutil.GetIntFromEnv("GH_AW_TIMEOUT", 60, 1, 3600, nil)
```

## Thread Safety

`GetIntFromEnv` is safe for concurrent use. It holds no shared mutable state; each invocation reads the process environment via `os.Getenv` and operates on function-local variables only.

## Design Decisions

- Warning messages use `console.FormatWarningMessage` so they render consistently in terminals.
- All warnings go to `os.Stderr` to avoid polluting structured stdout output.
- The function handles integers only; floating-point or string env vars should be read directly via `os.Getenv`.
- When `debugLog` is non-nil, warnings are routed through the logger rather than written directly to stderr, allowing callers to control output formatting.

## Dependencies

**Internal**:
- `github.com/github/gh-aw/pkg/console` — warning message formatting
- `github.com/github/gh-aw/pkg/logger` — debug logging

---

*This specification is automatically maintained by the [spec-extractor](../../.github/workflows/spec-extractor.md) workflow.*
