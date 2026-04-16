# typeutil Package

The `typeutil` package provides general-purpose type conversion utilities for working with heterogeneous `any` values, particularly those arising from JSON and YAML parsing.

## Overview

JSON and YAML parsers produce `any` values whose concrete type varies at runtime (`int`, `float64`, `string`, etc.). This package provides safe, well-documented conversion functions that handle the common cases without requiring callers to write their own type switches.

## Functions

### Strict Conversions

#### `ParseIntValue(value any) (int, bool)`

Strictly parses numeric types (`int`, `int64`, `uint64`, `float64`) to `int`. Returns `(value, true)` on success and `(0, false)` for any unrecognized or non-numeric type.

Use this when the caller **must distinguish** a missing or invalid value from a legitimate zero (e.g. YAML config field parsing where the YAML library has already produced a typed numeric value).

```go
v, ok := typeutil.ParseIntValue(someYAMLField)
if !ok {
    return errors.New("field is missing or not an integer")
}
```

#### `ParseBool(m map[string]any, key string) bool`

Extracts a boolean value from a `map[string]any` by key. Returns `false` if the map is `nil`, the key is absent, or the value is not a `bool`.

```go
enabled := typeutil.ParseBool(config, "enabled")
```

### Safe Overflow Conversions

#### `SafeUint64ToInt(u uint64) int`

Converts `uint64` to `int`, returning `0` if the value would overflow `int`.

#### `SafeUintToInt(u uint) int`

Converts `uint` to `int`, returning `0` if the value would overflow `int`. Thin wrapper around `SafeUint64ToInt`.

### Lenient Conversions

#### `ConvertToInt(val any) int`

Leniently converts any value to `int`, returning `0` on failure. Unlike `ParseIntValue`, this function also handles string inputs via `strconv.Atoi`, making it suitable for heterogeneous sources such as JSON metrics, log-parsed data, or user-provided configuration where a zero default on failure is acceptable.

```go
// Works with int, int64, float64, and string inputs
count := typeutil.ConvertToInt(jsonData["count"])
```

#### `ConvertToFloat(val any) float64`

Safely converts any value (`float64`, `int`, `int64`, `string`) to `float64`, returning `0` on failure.

```go
ratio := typeutil.ConvertToFloat(jsonData["ratio"])
```

## Choosing the Right Function

| Situation | Function to use |
|-----------|----------------|
| YAML/Go-typed numeric field; must detect missing vs zero | `ParseIntValue` |
| JSON / log-parsed metric; zero default on failure is fine | `ConvertToInt` |
| Boolean flag in a `map[string]any` | `ParseBool` |
| Casting `uint64` counter to `int` | `SafeUint64ToInt` |
| Numeric value from any source as float | `ConvertToFloat` |

## Design Notes

- All debug output uses `logger.New("typeutil:convert")` and is only emitted when `DEBUG=typeutil:*`.
- `float64 → int` truncation is logged at debug level when the fractional part is lost.
- `uint64 → int` overflow returns `0` rather than panicking, following the defensive convention used elsewhere in the codebase.
