# colorwriter Package

> Thin wrapper that returns a color-profile-aware `io.Writer` for terminal output, with a no-op stub for wasm builds.

## Overview

The `colorwriter` package provides a factory for `io.Writer` values that adapt ANSI color output based on the current environment. On non-wasm platforms it wraps the given writer with [`github.com/charmbracelet/colorprofile`](https://pkg.go.dev/github.com/charmbracelet/colorprofile) so that `NO_COLOR`, `COLORTERM`, and terminal capability are consulted automatically. On wasm (`js` / `wasm` build tags) the package returns the writer unchanged, since color-profile detection is not supported on that platform.

It is imported by `pkg/console` and `pkg/logger` to obtain consistent stdout/stderr writers and to degrade already-rendered ANSI strings when output helpers must return a string instead of writing directly.

## Public API

| Symbol | Signature | Description |
|--------|-----------|-------------|
| `New` | `func(w io.Writer, environ []string) io.Writer` | Returns a color-profile-aware writer wrapping `w` using `environ` (e.g. `os.Environ()`) to detect `NO_COLOR`, `COLORTERM`, and terminal capabilities. On wasm, returns `w` unchanged. |
| `Stderr` | `func() io.Writer` | Convenience wrapper that calls `New(os.Stderr, os.Environ())`. On wasm, returns `os.Stderr` directly. |
| `Stdout` | `func() io.Writer` | Convenience wrapper that calls `New(os.Stdout, os.Environ())`. On wasm, returns `os.Stdout` directly. |
| `Degrade` | `func(s string, environ []string) string` | Routes a pre-rendered ANSI string through a color-profile-aware writer backed by a string builder, downgrading or stripping ANSI according to `environ`. On wasm, returns `s` unchanged. |

### Build variants

| Build constraint | Behavior |
|-----------------|----------|
| `!js && !wasm` (`colorprofile_writer.go`) | `New` delegates to `colorprofile.NewWriter`; `Stderr` and `Stdout` wrap the corresponding standard streams with the process environment; `Degrade` transforms a rendered ANSI string through an in-memory color-profile-aware writer. |
| `js \|\| wasm` (`colorprofile_writer_wasm.go`) | `New` returns `w` unchanged; `Stderr` and `Stdout` return the corresponding standard streams directly; `Degrade` returns the original string unchanged. Color-profile detection is not supported on wasm. |

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

// Degrade a rendered ANSI string before returning it to a stdout caller.
plain := colorwriter.Degrade("\x1b[31mwarning\x1b[0m", []string{"NO_COLOR=1", "TERM=xterm-256color"})
fmt.Println(plain)
```

## Dependencies

**External**:
- `github.com/charmbracelet/colorprofile` — color-profile detection and ANSI downgrading (non-wasm builds only)

<!-- BEGIN SOURCE-VERIFIED EXPORT COVERAGE -->
## Source-verified export coverage

This appendix is generated from the current non-test Go source files in this package and records any exported top-level symbols that are not already described above.

| Category | Count |
|----------|------:|
| Types | 0 |
| Constants | 0 |
| Variables | 0 |
| Functions and methods | 4 |
| Additional symbols documented in this appendix | 0 |

The sections above already mention every exported top-level symbol in the current source tree.
<!-- END SOURCE-VERIFIED EXPORT COVERAGE -->

---

*This specification is automatically maintained by the [spec-extractor](../../.github/workflows/spec-extractor.md) workflow.*
