# syncutil Package

The `syncutil` package provides thread-safe synchronization utilities for concurrent operations.

## Overview

This package provides generic types for common concurrency patterns with zero-allocation caching. It is designed for situations where an expensive or fallible operation should be executed at most once, with subsequent callers receiving the cached result.

## Public API

### Types

| Symbol | Kind | Description |
|--------|------|-------------|
| `OnceLoader[T]` | struct | Caches the result of an expensive, fallible one-shot fetch; safe for concurrent use |

### Methods on `OnceLoader[T]`

| Method | Signature | Description |
|--------|-----------|-------------|
| `Get` | `func (o *OnceLoader[T]) Get(loader func() (T, error)) (T, error)` | Returns the cached result, invoking `loader` exactly once |
| `Reset` | `func (o *OnceLoader[T]) Reset()` | Clears the cached result and error so that the next `Get` call re-invokes `loader` |
| `Override` | `func (o *OnceLoader[T]) Override(result T, err error)` | Stores `result` and `err` as the cached value without invoking `loader`; subsequent `Get` calls return this value |

## Usage Examples

```go
import "github.com/github/gh-aw/pkg/syncutil"

var cache syncutil.OnceLoader[string]

// loader is called only once; subsequent calls return the cached value
value, err := cache.Get(func() (string, error) {
    return expensiveOperation()
})

// Reset allows re-fetching the value on the next Get call
cache.Reset()
```

**Typical usage as a package-level cache**:

```go
var currentRepoSlugCache syncutil.OnceLoader[string]

func getCurrentRepoSlug() (string, error) {
    return currentRepoSlugCache.Get(func() (string, error) {
        return fetchRepoSlugFromGitHub()
    })
}
```

## Design Notes

- The internal mutex ensures that `loader` is invoked at most once, even when multiple goroutines call `Get` concurrently.
- If `loader` returns an error, the error is cached alongside the zero value of `T`; subsequent calls return the same error without re-invoking `loader`.
- `Reset` acquires the same mutex, making it safe to call concurrently with `Get`.
- The zero value of `OnceLoader[T]` is ready to use; no constructor is needed.

## Dependencies

**Internal**:
- `github.com/github/gh-aw/pkg/logger` — package-scoped logging used by `OnceLoader[T]`.

**External**:
- None beyond the Go standard library (`sync`).

---

*This specification is automatically maintained by the [spec-extractor](../../.github/workflows/spec-extractor.md) workflow.*
