# envutil Package

The `envutil` package provides utilities for reading and validating environment variables with bounds checking.

## Overview

This package centralizes the pattern of reading integer-valued environment variables, validating them against configured minimum and maximum bounds, and falling back to a default value when the variable is absent or out of range. It emits warning messages to stderr when an invalid value is encountered, following the console formatting conventions of the rest of the codebase.

## Usage

### GetIntFromEnv

```go
import (
    "github.com/github/gh-aw/pkg/envutil"
    "github.com/github/gh-aw/pkg/logger"
)

var log = logger.New("mypackage:config")

// Read GH_AW_MAX_CONCURRENT_DOWNLOADS, constrained to [1, 20], default 5
concurrency := envutil.GetIntFromEnv("GH_AW_MAX_CONCURRENT_DOWNLOADS", 5, 1, 20, log)
```

**Behavior**:
- Returns `defaultValue` when the environment variable is not set.
- Returns `defaultValue` and emits a warning when the value cannot be parsed as an integer.
- Returns `defaultValue` and emits a warning when the value is outside `[minValue, maxValue]`.
- Logs the accepted value at debug level when `log` is non-nil.
- Pass `nil` for `log` to suppress debug output.

### Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `envVar` | `string` | Environment variable name (e.g. `"GH_AW_TIMEOUT"`) |
| `defaultValue` | `int` | Value returned when env var is absent or invalid |
| `minValue` | `int` | Minimum allowed value (inclusive) |
| `maxValue` | `int` | Maximum allowed value (inclusive) |
| `log` | `*logger.Logger` | Optional logger for debug output; pass `nil` to disable |

## Design Notes

- Warning messages use `console.FormatWarningMessage` so they render consistently in terminals.
- All warnings go to `os.Stderr` to avoid polluting structured stdout output.
- The function only handles integers; floating-point or string env vars should be read directly via `os.Getenv`.
