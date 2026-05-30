# sliceutil Package

The `sliceutil` package provides generic utility functions for working with slices and maps.

## Overview

All functions in this package are pure: they never modify their input. They are generic and work with any element type using Go's type-parameter syntax.

## Public API

### Functions

| Function | Signature | Description |
|----------|-----------|-------------|
| `Filter` | `func[T any](slice []T, predicate func(T) bool) []T` | Returns a new slice containing only elements for which `predicate` returns `true` |
| `Map` | `func[T, U any](slice []T, transform func(T) U) []U` | Applies `transform` to every element and returns the results as a new slice |
| `MapKeys` | `func[K comparable, V any](m map[K]V) []K` | Converts the keys of a map into a slice; order is not guaranteed |
| `FilterMapKeys` | `func[K comparable, V any](m map[K]V, predicate func(K, V) bool) []K` | Returns map keys for which `predicate(key, value)` is `true`; order is not guaranteed |
| `Any` | `func[T any](slice []T, predicate func(T) bool) bool` | Returns `true` if at least one element satisfies `predicate`; returns `false` for nil or empty slices |
| `Deduplicate` | `func[T comparable](slice []T) []T` | Returns a new slice with duplicate elements removed, preserving order of first occurrence |
| `MergeUnique` | `func[T comparable](base []T, extra ...T) []T` | Returns a deduplicated slice starting with `base` and appending unseen values from `extra` |
| `Exclude` | `func[T comparable](base []T, exclude ...T) []T` | Returns a new slice with all `exclude` values removed while preserving order |

## Usage Examples

```go
import "github.com/github/gh-aw/pkg/sliceutil"

// Filter a slice
numbers := []int{1, 2, 3, 4, 5}
evens := sliceutil.Filter(numbers, func(n int) bool { return n%2 == 0 })
// evens = [2, 4]

// Map a slice
names := []string{"alice", "bob"}
upper := sliceutil.Map(names, strings.ToUpper)
// upper = ["ALICE", "BOB"]

// Deduplicate
items := []string{"a", "b", "a", "c"}
unique := sliceutil.Deduplicate(items)
// unique = ["a", "b", "c"]

// Merge unique values
merged := sliceutil.MergeUnique([]string{"a", "b"}, "b", "c")
// merged = ["a", "b", "c"]

// Exclude values
filtered := sliceutil.Exclude([]string{"a", "b", "c"}, "b")
// filtered = ["a", "c"]
```

## Dependencies

**Internal**:
- `pkg/logger` — package-scoped logging used by `Deduplicate` and `MergeUnique`.

**External**:
- None beyond the Go standard library (`slices`).

## Design Notes

- `Any` is implemented via `slices.ContainsFunc` from the standard library.
- `Deduplicate`, `MergeUnique`, and `Exclude` use hash sets (`map[T]struct{}`) for O(n) behavior.
- None of these functions sort their output; callers that require sorted results should call `slices.Sort` on the returned slice.

---

*This specification is automatically maintained by the [spec-extractor](../../.github/workflows/spec-extractor.md) workflow.*
