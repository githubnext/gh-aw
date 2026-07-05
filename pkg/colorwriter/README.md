# colorwriter Package

> Thin wrapper that returns a color-profile-aware `io.Writer` for terminal output, with a no-op stub for wasm builds.

## Overview

The `colorwriter` package provides a factory for `io.Writer` values that adapt ANSI color output based on the current environment. On non-wasm platforms it wraps the given writer with [`github.com/charmbracelet/colorprofile`](https://pkg.go.dev/github.com/charmbracelet/colorprofile) so that `NO_COLOR`, `COLORTERM`, and terminal capability are consulted automatically. On wasm (`js` / `wasm` build tags) the package returns the writer unchanged, since color-profile detection is not supported on that platform.

It is imported by `pkg/console` and `pkg/logger` to obtain a consistent stderr writer.

## Public API

| Symbol | Signature | Description |
|--------|-----------|-------------|
| `New` | `func(w io.Writer, environ []string) io.Writer` | Returns a color-profile-aware writer wrapping `w` using `environ` (e.g. `os.Environ()`) to detect `NO_COLOR`, `COLORTERM`, and terminal capabilities. On wasm, returns `w` unchanged. |
| `Stderr` | `func() io.Writer` | Convenience wrapper that calls `New(os.Stderr, os.Environ())`. On wasm, returns `os.Stderr` directly. |

### Build variants

| Build constraint | Behavior |
|-----------------|----------|
| `!js && !wasm` (`colorprofile_writer.go`) | `New` delegates to `colorprofile.NewWriter`; `Stderr` wraps `os.Stderr` with the process environment. |
| `js \|\| wasm` (`colorprofile_writer_wasm.go`) | `New` returns `w` unchanged; `Stderr` returns `os.Stderr` directly. Color-profile detection is not supported on wasm. |

## Usage Examples

```go
import (
    "os"

    "github.com/github/gh-aw/pkg/colorwriter"
)

// Wrap an arbitrary writer (e.g. for tests or piped output).
w := colorwriter.New(os.Stderr, os.Environ())
fmt.Fprintln(w, "styled output respects NO_COLOR and terminal capabilities")

// Obtain a ready-to-use stderr writer.
stderr := colorwriter.Stderr()
fmt.Fprintln(stderr, "styled output to os.Stderr")
```

## Dependencies

**External**:
- `github.com/charmbracelet/colorprofile` — color-profile detection and ANSI downgrading (non-wasm builds only)

---

*This specification is automatically maintained by the [spec-extractor](../../.github/workflows/spec-extractor.md) workflow.*
