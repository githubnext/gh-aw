# logger Package

> Namespace-based debug logging with zero overhead when disabled, following the pattern of the [debug npm package](https://www.npmjs.com/package/debug).

## Overview

A simple, debug-style logging framework for Go that follows the pattern matching syntax of the [debug npm package](https://www.npmjs.com/package/debug).

## Features

- **Namespace-based logging**: Each logger has a namespace (e.g., `workflow:compiler`, `cli:audit`)
- **Pattern matching**: Enable/disable loggers using wildcards and exclusions via the `DEBUG` environment variable
- **Printf interface**: Standard printf-style formatting
- **Time diff display**: Shows time elapsed since last log call (like debug npm package)
- **Automatic color coding**: Each namespace gets a unique color, determined by `DEBUG_COLORS` and rendered by lipgloss
- **Zero overhead**: Logger enabled state is computed once at construction time
- **Thread-safe**: Safe for concurrent use

## Public API

### Types

| Type | Kind | Description |
|------|------|-------------|
| `Logger` | struct | Namespace-based debug logger; enabled state and namespace label are computed once at construction time |
| `SlogHandler` | struct | Implements `slog.Handler` by delegating to a `Logger`; allows libraries that expect a standard `slog.Logger` to use the gh-aw logger |

### `Logger`

The `Logger` type provides namespace-based debug logging with pattern matching, printf interface, and time-diff display.

| Method / Function | Signature | Description |
|------------------|-----------|-------------|
| `New` | `func(namespace string) *Logger` | Creates a new logger for the given namespace |
| `(*Logger).Printf` | `func(format string, args ...any)` | Formatted output (always adds newline) |
| `(*Logger).Print` | `func(args ...any)` | Simple concatenation (always adds newline) |
| `(*Logger).Enabled` | `func() bool` | Returns `true` if the logger matches the active `DEBUG` pattern |
| `NewSlogHandler` | `func(logger *Logger) *SlogHandler` | Creates a `slog.Handler` wrapping the given `Logger` |
| `NewSlogLoggerWithHandler` | `func(logger *Logger) *slog.Logger` | Creates a `slog.Logger` backed by the given `Logger` |

**Behavioral contracts**:

- `New` MUST compute the enabled state exactly once at construction time from the `DEBUG` environment variable (or `ACTIONS_RUNNER_DEBUG=true` as a fallback); subsequent changes to those variables MUST NOT affect already-constructed `Logger` instances.
- `Logger.Printf` and `Logger.Print` MUST be no-ops (return immediately before any string formatting or I/O) when `Enabled()` returns `false`.
- `Logger.Printf` and `Logger.Print` MUST write all output to `os.Stderr` and MUST append a trailing newline to every message.
- `Logger.Printf` and `Logger.Print` MUST include a `+<duration>` suffix showing elapsed time since the previous call on the same instance.
- `Logger.Printf` and `Logger.Print` MUST acquire an internal mutex before updating the `lastLog` timestamp; concurrent callers MUST receive independent, accurate time-diff values without data races.
- `SlogHandler.Enabled` MUST return `false` when the underlying `Logger.Enabled()` is `false`, preventing attribute collection overhead for disabled loggers.
- `SlogHandler.WithAttrs` and `SlogHandler.WithGroup` MUST return the receiver unchanged; attributes and groups are not accumulated across calls.
- Color styling MUST be disabled when `DEBUG_COLORS=0`; when enabled, actual color rendering is delegated to lipgloss which adapts to the output terminal's capabilities.

## Usage Examples

### Basic Usage

```go
package main

import "github.com/github/gh-aw/pkg/logger"

var log = logger.New("myapp:feature")

func main() {
    log.Printf("Starting application with config: %s", config)
    log.Print("Multiple", " ", "arguments")
}
```

Output shows namespace, message, and time diff:
```
myapp:feature Starting application with config: production +0ns
myapp:feature Multiple arguments +125ms
```

### Avoiding Expensive Operations

Check if a logger is enabled before performing expensive operations:

```go
if log.Enabled() {
    // Do expensive work only if logging is enabled
    result := expensiveOperation()
    log.Printf("Result: %v", result)
}
```

### Time Diff Display

Like the debug npm package, each log shows the time elapsed since the last log call:

```go
log.Printf("Starting task")
// ... do some work ...
log.Printf("Task completed")  // Shows +2.5s (or +500ms, +100µs, etc.)
```

## DEBUG Environment Variable

Control which loggers are enabled using the `DEBUG` environment variable with patterns. When `ACTIONS_RUNNER_DEBUG=true` is set (as it is in GitHub Actions debug runs) and `DEBUG` is not explicitly set, all loggers are enabled automatically — equivalent to `DEBUG=*`.

### Examples

```bash
# Enable all loggers
DEBUG=*

# Enable all loggers in the 'workflow' namespace
DEBUG=workflow:*

# Enable specific loggers
DEBUG=workflow:compiler,cli:audit

# Enable all except specific loggers
DEBUG=*,-workflow:compiler

# Enable namespace but exclude specific patterns
DEBUG=workflow:*,-workflow:compiler:cache

# Multiple patterns with exclusions
DEBUG=workflow:*,cli:*,-workflow:test
```

## Color Support

Color styling is enabled when `DEBUG_COLORS` is not set to `0`. The [lipgloss](https://github.com/charmbracelet/lipgloss) library handles actual color rendering and adapts to the output terminal's capabilities; colors are typically suppressed when piping output to a non-TTY stream.

Each namespace gets a consistent color based on a FNV-1a hash of its name. This makes it easy to visually distinguish between different loggers.

### Disabling Colors

```bash
# Disable colors
DEBUG_COLORS=0 DEBUG=* gh aw compile workflow.md

# Colors are automatically disabled when piping output
DEBUG=* gh aw compile workflow.md 2>&1 | tee output.log
```

### Pattern Syntax

- `*` - Matches all loggers
- `namespace:*` - Matches all loggers with the given prefix
- `*:suffix` - Matches all loggers with the given suffix
- `prefix:*:suffix` - Matches loggers with both prefix and suffix
- `-pattern` - Excludes loggers matching the pattern (takes precedence)
- `pattern1,pattern2` - Multiple patterns separated by commas

## Design Decisions

### Logger Enabled State

The enabled state is computed **once at logger construction time** based on the `DEBUG` environment variable. This means:

- Zero overhead for disabled loggers (simple boolean check)
- `DEBUG` changes after the process starts won't affect existing loggers

### Time Diff Tracking

Each logger tracks the time of its last log call to display elapsed time, similar to the debug npm package. This helps identify performance bottlenecks and understand timing relationships between log messages.

### Output Destination

All log output goes to **stderr** to avoid interfering with stdout data (JSON, command output, etc.).

### Printf Interface

The logger provides a familiar printf-style interface that Go developers expect:

- `Printf(format, args...)` - Formatted output (always adds newline)
- `Print(args...)` - Simple concatenation (always adds newline)

## Example Patterns

### File-based Namespaces

```go
// In pkg/workflow/compiler.go
var log = logger.New("workflow:compiler")

// In pkg/cli/audit.go  
var log = logger.New("cli:audit")

// In pkg/parser/frontmatter.go
var log = logger.New("parser:frontmatter")
```

Enable with:
```bash
DEBUG=workflow:* go run main.go      # Only workflow package
DEBUG=cli:*,parser:* go run main.go  # CLI and parser packages
DEBUG=* go run main.go                # Everything
```

### Feature-based Namespaces

```go
var compileLog = logger.New("compile")
var parseLog = logger.New("parse")
var validateLog = logger.New("validate")
```

## slog Integration

The package includes a bridge to Go's standard `log/slog` library for libraries that expect a `slog.Logger` instead of the custom `Logger` type.

### `SlogHandler`

`SlogHandler` implements `slog.Handler` by delegating to an existing `Logger`. It respects the logger's enabled state, formats attributes as `key=value` pairs, and prefixes each message with compact terminal-friendly glyphs (`·` for debug/info, `⚠` for warning, `✗` for error).

### `NewSlogHandler(logger *Logger) *SlogHandler`

Creates a new `slog.Handler` wrapping the provided `Logger`.

```go
import "github.com/github/gh-aw/pkg/logger"

var log = logger.New("myapp:feature")
handler := logger.NewSlogHandler(log)
slogLogger := slog.New(handler)
slogLogger.Info("using slog interface", "key", "value")
```

### `NewSlogLoggerWithHandler(logger *Logger) *slog.Logger`

Convenience constructor that creates both the `SlogHandler` and the `slog.Logger` in one call.

```go
var log = logger.New("myapp:feature")
slogLogger := logger.NewSlogLoggerWithHandler(log)
slogLogger.Warn("something unusual happened", "count", 42)
```

### Behavior

- **Enabled check**: `SlogHandler.Enabled` returns `false` when the underlying `Logger` is disabled (i.e. the namespace does not match the `DEBUG` pattern). This prevents expensive attribute collection for disabled loggers.
- **Attribute formatting**: All record attributes are appended as `key=value` pairs after the message.
- **Groups and persistent attributes**: `WithAttrs` and `WithGroup` return the handler unchanged — attributes are not persisted across calls. This keeps the adapter lightweight.
- **Output destination**: All output goes to `stderr` via the underlying `Logger`.

## Thread Safety

`Logger` instances are safe for concurrent use from multiple goroutines. The `Printf` and `Print` methods acquire a `sync.Mutex` before updating the `lastLog` timestamp, so concurrent callers receive accurate time-diff values without data races. The `Enabled` method is read-only and requires no lock.

`SlogHandler` instances are safe for concurrent use when the underlying `Logger` is safe for concurrent use.

## Dependencies

**Internal**:
- `github.com/github/gh-aw/pkg/colorwriter` — color-profile-aware stderr writer selection
- `github.com/github/gh-aw/pkg/timeutil` — time-diff formatting for log output
- `github.com/github/gh-aw/pkg/styles` — shared terminal style constants and color helpers

## Implementation Notes

- The `DEBUG` environment variable is read once when the package is initialized
- Thread-safe using `sync.Mutex` for time tracking
- Simple pattern matching without regex (prefix, suffix, and middle wildcards only)
- Exclusion patterns (prefixed with `-`) take precedence over inclusion patterns
- Time diff formatted like debug npm package (ns, µs, ms, s, m, h)
- Colors assigned using FNV-1a hash for consistent namespace-to-color mapping
- Color palette chosen for readability on both light and dark terminals
- Uses ANSI 256-color codes for better compatibility

---

*This specification is automatically maintained by the [spec-extractor](../../.github/workflows/spec-extractor.md) workflow.*
