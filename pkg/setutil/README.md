# setutil Package

The `setutil` package provides utility functions for working with sets implemented as `map[K]struct{}`.

## Overview

All functions in this package are pure: they never modify their input. They are generic and work with any comparable key type using Go's type-parameter syntax.

## Public API

### Functions

| Function | Signature | Description |
|----------|-----------|-------------|
| `Contains` | `func Contains[K comparable](set map[K]struct{}, key K) bool` | Reports whether `key` is present in `set` |

## Usage Examples

```go
import "github.com/github/gh-aw/pkg/setutil"

// Check membership in a string set
seen := map[string]struct{}{"foo": {}, "bar": {}}
if setutil.Contains(seen, "foo") {
    // ...
}
```

## Dependencies

**Internal**:
- None

**External**:
- None beyond the Go standard library.

---

*This specification is automatically maintained by the [spec-extractor](../../.github/workflows/spec-extractor.md) workflow.*
