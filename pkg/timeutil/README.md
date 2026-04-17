# timeutil Package

The `timeutil` package provides human-readable duration formatting utilities.

## Overview

This package contains helpers for converting `time.Duration` values and raw numeric durations (milliseconds, nanoseconds) into compact, readable strings. The primary formatting style follows the [debug npm package](https://www.npmjs.com/package/debug) conventions used by the `logger` package.

## Functions

### `FormatDuration(d time.Duration) string`

Formats a `time.Duration` for display. Provides granular output from nanoseconds to hours.

| Range | Example output |
|-------|---------------|
| `< 1µs` | `"500ns"` |
| `1µs – < 1ms` | `"250µs"` |
| `1ms – < 1s` | `"750ms"` |
| `1s – < 1min` | `"2.5s"` |
| `1min – < 1h` | `"1.3m"` |
| `≥ 1h` | `"2.0h"` |

```go
import "github.com/github/gh-aw/pkg/timeutil"

timeutil.FormatDuration(500 * time.Millisecond)  // "500ms"
timeutil.FormatDuration(2500 * time.Millisecond) // "2.5s"
timeutil.FormatDuration(90 * time.Second)        // "1.5m"
```

### `FormatDurationMs(ms int) string`

Formats a duration given in **milliseconds** as a human-readable string.

| Range | Example |
|-------|---------|
| `< 1000ms` | `"500ms"` |
| `1000ms – < 60s` | `"1.5s"` |
| `≥ 60s` | `"1m30s"` |

```go
timeutil.FormatDurationMs(500)   // "500ms"
timeutil.FormatDurationMs(1500)  // "1.5s"
timeutil.FormatDurationMs(90000) // "1m30s"
```

### `FormatDurationNs(ns int64) string`

Formats a duration given in **nanoseconds** as a human-readable string. Returns `"—"` for zero or negative values. Uses Go's standard `time.Duration.Round(time.Second)` for output.

```go
timeutil.FormatDurationNs(0)              // "—"
timeutil.FormatDurationNs(2_500_000_000)  // "2s"
timeutil.FormatDurationNs(90_000_000_000) // "1m30s"
```

## Design Notes

- `FormatDuration` is used by the `logger` package to display time-diff between consecutive log calls (the `+500ms` suffix in debug output).
- `FormatDurationMs` is used for workflow run duration display, where GitHub Actions reports durations in milliseconds.
- `FormatDurationNs` is used for job duration display, where GitHub Actions reports billing durations in nanoseconds.

---

*This specification is automatically maintained by the [spec-extractor](../../.github/workflows/spec-extractor.md) workflow.*
