# sliceutil Package

The `sliceutil` package provides generic utility functions for working with slices and maps.

## Overview

All functions in this package are pure: they never modify their input. They are generic and work with any element type using Go's type-parameter syntax.

## Functions

### `Filter[T any](slice []T, predicate func(T) bool) []T`

Returns a new slice containing only elements for which `predicate` returns `true`.

```go
import "github.com/github/gh-aw/pkg/sliceutil"

numbers := []int{1, 2, 3, 4, 5}
evens := sliceutil.Filter(numbers, func(n int) bool { return n%2 == 0 })
// evens = [2, 4]
```

### `Map[T, U any](slice []T, transform func(T) U) []U`

Applies `transform` to every element and returns the results as a new slice.

```go
names := []string{"alice", "bob"}
upper := sliceutil.Map(names, strings.ToUpper)
// upper = ["ALICE", "BOB"]
```

### `MapToSlice[K comparable, V any](m map[K]V) []K`

Converts the keys of a map into a slice. **Order is not guaranteed.**

```go
m := map[string]int{"a": 1, "b": 2}
keys := sliceutil.MapToSlice(m)
// keys = ["a", "b"] (in some order)
```

### `FilterMapKeys[K comparable, V any](m map[K]V, predicate func(K, V) bool) []K`

Returns the map keys for which `predicate(key, value)` is `true`. **Order is not guaranteed.**

```go
scores := map[string]int{"alice": 90, "bob": 50, "carol": 80}
passed := sliceutil.FilterMapKeys(scores, func(name string, score int) bool {
    return score >= 75
})
// passed = ["alice", "carol"] (in some order)
```

### `Any[T any](slice []T, predicate func(T) bool) bool`

Returns `true` if at least one element in `slice` satisfies `predicate`. Returns `false` for nil or empty slices.

```go
words := []string{"hello", "world"}
hasWorld := sliceutil.Any(words, func(w string) bool { return w == "world" })
// hasWorld = true
```

### `Deduplicate[T comparable](slice []T) []T`

Returns a new slice with duplicate elements removed, preserving the order of first occurrence.

```go
items := []string{"a", "b", "a", "c", "b"}
unique := sliceutil.Deduplicate(items)
// unique = ["a", "b", "c"]
```

## Design Notes

- `Any` is implemented via `slices.ContainsFunc` from the standard library.
- `Deduplicate` uses a `map[T]bool` for O(n) time complexity.
- None of these functions sort their output; callers that require sorted results should call `slices.Sort` on the returned slice.

---

*This specification is automatically maintained by the [spec-extractor](../../.github/workflows/spec-extractor.md) workflow.*
