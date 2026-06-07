# importinpututil Package

The `importinpututil` package provides input-value resolution and formatting utilities for `@import` directives in `gh-aw` workflows.

## Overview

When a workflow uses an `@import` directive, it may supply input values that override defaults in the imported workflow. This package resolves those input values by path (supporting dotted sub-key notation) and formats the resolved values for textual substitution in the workflow rendering pipeline.

## Public API

### Exported Functions

| Function | Signature | Description |
|----------|-----------|-------------|
| `ResolvePathValue` | `func(inputs map[string]any, inputPath string) (any, bool)` | Resolves a top-level or dotted sub-key from import inputs |
| `FormatResolvedValue` | `func(value any) (string, bool)` | Formats a resolved import input value for textual substitution |

### `ResolvePathValue(inputs map[string]any, inputPath string) (any, bool)`

Resolves either a top-level input key (e.g. `"count"`) or a one-level dotted object sub-key (e.g. `"config.apiKey"`) from a map of import inputs. Returns `(value, true)` on success, or `(nil, false)` when the key is absent or the dotted path cannot be traversed.

```go
inputs := map[string]any{
    "count":  42,
    "config": map[string]any{"apiKey": "secret"},
}

// Top-level key
value, ok := importinpututil.ResolvePathValue(inputs, "count")
// value == 42, ok == true

// Dotted sub-key
value, ok = importinpututil.ResolvePathValue(inputs, "config.apiKey")
// value == "secret", ok == true

// Missing key
_, ok = importinpututil.ResolvePathValue(inputs, "missing")
// ok == false
```

### `FormatResolvedValue(value any) (string, bool)`

Formats a resolved import input value for textual substitution. The formatting rules are:

- `nil` returns `("", false)`
- Scalar values (`int`, `bool`, `string`, etc.) are formatted with `fmt.Sprintf("%v", v)`
- `[]any` and `map[string]any` values are JSON-marshalled
- Typed slices and maps are normalized to `[]any` / `map[string]any` and JSON-marshalled
- Returns `("", false)` if JSON marshalling fails

```go
// Scalar
s, ok := importinpututil.FormatResolvedValue(42)
// s == "42", ok == true

// Slice
s, ok = importinpututil.FormatResolvedValue([]any{"a", "b"})
// s == `["a","b"]`, ok == true

// Map
s, ok = importinpututil.FormatResolvedValue(map[string]any{"x": 1})
// s == `{"x":1}`, ok == true

// nil
s, ok = importinpututil.FormatResolvedValue(nil)
// s == "", ok == false
```

## Usage Examples

```go
import "github.com/github/gh-aw/pkg/importinpututil"

// Resolve an import input by path
value, ok := importinpututil.ResolvePathValue(importInputs, "config.endpoint")
if !ok {
    return errors.New("required import input 'config.endpoint' not found")
}

// Format the resolved value for substitution
formatted, ok := importinpututil.FormatResolvedValue(value)
if !ok {
    return errors.New("import input value could not be formatted")
}
```

## Dependencies

This package has no external or internal `gh-aw` dependencies — it uses only the Go standard library.

## Design Notes

- Only one level of dot notation is supported for sub-key resolution: `"a.b"` is valid; `"a.b.c"` is treated as a lookup for key `"b.c"` inside the top-level object `"a"`, which will typically fail.
- Map keys in JSON output are sorted lexicographically for deterministic output.
- Typed slices and maps (e.g. `[]string`, `map[string]int`) are normalized to `[]any` / `map[string]any` via reflection before marshalling.

---

*This specification is automatically maintained by the [spec-extractor](../../.github/workflows/spec-extractor.md) workflow.*
