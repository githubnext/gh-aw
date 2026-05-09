# envutil Package

The `envutil` package provides utilities for reading and validating environment variables with bounds checking.

## Overview

This package centralizes the pattern of reading integer-valued environment variables, validating them against configured minimum and maximum bounds, and falling back to a default value when the variable is absent or out of range. It emits warning messages to stderr when an invalid value is encountered, following the console formatting conventions of the rest of the codebase.

## Public API

### `GetIntFromEnv(envVar string, defaultValue, minValue, maxValue int, log *logger.Logger) int`

Reads an integer-valued environment variable, validates it against `[minValue, maxValue]`, and returns `defaultValue` when the variable is absent, unparseable, or out of range. A warning is emitted to `os.Stderr` when the value is invalid.

| Parameter | Type | Description |
|-----------|------|-------------|
| `envVar` | `string` | Environment variable name (e.g. `"GH_AW_TIMEOUT"`) |
| `defaultValue` | `int` | Value returned when env var is absent or invalid |
| `minValue` | `int` | Minimum allowed value (inclusive) |
| `maxValue` | `int` | Maximum allowed value (inclusive) |
| `log` | `*logger.Logger` | Optional logger for debug output; pass `nil` to disable |

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

**Behavior**:
- Returns `defaultValue` when the environment variable is not set.
- Returns `defaultValue` and emits a warning when the value cannot be parsed as an integer.
- Returns `defaultValue` and emits a warning when the value is outside `[minValue, maxValue]`.
- Logs the accepted value at debug level when `log` is non-nil.

## Dependencies

**Internal**:
- `github.com/github/gh-aw/pkg/console` — warning message formatting
- `github.com/github/gh-aw/pkg/logger` — debug logging

## Design Notes

- Warning messages use `console.FormatWarningMessage` so they render consistently in terminals.
- All warnings go to `os.Stderr` to avoid polluting structured stdout output.
- The function only handles integers; floating-point or string env vars should be read directly via `os.Getenv`.

---

*This specification is automatically maintained by the [spec-extractor](../../.github/workflows/spec-extractor.md) workflow.*
