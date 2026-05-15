# linters Package

The `linters` package namespace contains custom static analysis linters used by `gh-aw` quality checks.

## Overview

This package currently provides one custom Go analyzer in the `largefunc` subpackage:

- `largefunc` — reports function bodies that exceed a configurable line-count threshold.

## Public API

### Subpackages

| Subpackage | Description |
|------------|-------------|
| `largefunc` | Custom `go/analysis` analyzer that flags large functions with actionable diagnostics |

## Usage Examples

```go
import "github.com/github/gh-aw/pkg/linters/largefunc"

// Use with multichecker, singlechecker, or custom go/analysis driver.
_ = largefunc.Analyzer
```

## Dependencies

**Internal**:
- None at the `pkg/linters` namespace level. `pkg/linters/largefunc` is documented above as a subpackage API, not as an internal dependency.

**External**:
- `golang.org/x/tools/go/analysis` — analyzer framework
- `golang.org/x/tools/go/analysis/passes/inspect` — AST inspection support
- `golang.org/x/tools/go/ast/inspector` — efficient AST traversal

## Design Notes

- The package is intentionally organized as a namespace (`pkg/linters/*`) so individual analyzers remain isolated and independently testable.
- `largefunc` exposes a `-max-lines` analyzer flag and defaults to `60` lines (`DefaultMaxLines`).

---

*This specification is automatically maintained by the [spec-extractor](../../.github/workflows/spec-extractor.md) workflow.*
