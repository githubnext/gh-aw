# tty Package

The `tty` package provides TTY (terminal) detection utilities.

## Overview

This package exposes two simple functions for checking whether the standard output or error streams are connected to a real terminal. The detection uses `golang.org/x/term`, which is the same library used by the spinner and progress-bar components in this codebase.

On WebAssembly targets (`js/wasm`) the package provides stub implementations that always return `false`, since WASM environments do not have real TTY file descriptors.

## Functions

### `IsStdoutTerminal() bool`

Returns `true` if `stdout` (`os.Stdout`) is connected to a terminal.

```go
import "github.com/github/gh-aw/pkg/tty"

if tty.IsStdoutTerminal() {
    // Safe to emit colored or animated output to stdout
}
```

### `IsStderrTerminal() bool`

Returns `true` if `stderr` (`os.Stderr`) is connected to a terminal.

```go
if tty.IsStderrTerminal() {
    // Safe to emit colored or animated output to stderr
}
```

## Design Notes

- Terminal detection is evaluated at call time, not cached. This is intentional: the streams could be redirected between calls in some testing scenarios.
- The WASM stub (`tty_wasm.go`) always returns `false` so that components built for the browser never attempt to use ANSI escape codes.
- Prefer this package over calling `term.IsTerminal` directly to keep the TTY detection logic centralized and easily testable.
- Components that need to adapt output for terminals (spinners, progress bars, colored messages) should call `IsStderrTerminal()` rather than checking `os.Stderr` directly.
