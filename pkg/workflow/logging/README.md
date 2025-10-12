# Logging Package

The `logging` package provides a structured logging interface for the compiler using Go's standard `log/slog` package.

## Overview

The package wraps `log/slog.Logger` with convenience methods for common logging patterns and supports both verbose and non-verbose modes.

## Features

- **Structured Logging**: Built on top of `log/slog` for modern structured logging
- **Verbose Mode**: Debug-level logging when verbose mode is enabled
- **Format Methods**: Convenient `*f` methods for formatted messages
- **Field Methods**: Structured logging with key-value pairs
- **Configurable Output**: Support for custom writers (useful for testing)

## Usage

### Creating a Logger

```go
import "github.com/githubnext/gh-aw/pkg/workflow/logging"

// Create a logger with verbose mode enabled
logger := logging.NewLogger(true)

// Create a logger with verbose mode disabled
logger := logging.NewLogger(false)

// Create a logger with custom output writer (for testing)
var buf bytes.Buffer
logger := logging.NewLoggerWithWriter(true, &buf)
```

### Logging Messages

#### Format Methods

```go
// Info level
logger.Infof("Compiling workflow: %s", workflowName)

// Debug level (only shown in verbose mode)
logger.Debugf("Processing step: %d", stepNum)

// Warning level
logger.Warnf("Validation warning: %s", warning)

// Error level
logger.Errorf("Compilation failed: %v", err)
```

#### Structured Logging with Fields

```go
// Info with fields
logger.InfoWithFields("Compilation started",
    "workflow", "example.md",
    "engine", "claude",
)

// Debug with fields
logger.DebugWithFields("Step processed",
    "step", 5,
    "duration", "1.2s",
)

// Warning with fields
logger.WarnWithFields("Resource limit approaching",
    "current", 95,
    "limit", 100,
)

// Error with fields
logger.ErrorWithFields("Failed to parse frontmatter",
    "file", "workflow.md",
    "error", err.Error(),
)
```

### Checking Verbose Mode

```go
if logger.IsVerbose() {
    // Perform expensive debug operations only when needed
}
```

## Integration with Compiler

The `Compiler` struct includes a logger field that is automatically initialized based on the verbose flag:

```go
// Create compiler with verbose logging
c := NewCompiler(true, "", "1.0.0")

// Access the logger
c.GetLogger().Infof("Starting compilation...")

// Set a custom logger (useful for testing)
var buf bytes.Buffer
customLogger := logging.NewLoggerWithWriter(true, &buf)
c.SetLogger(customLogger)
```

## Output Format

The logger uses `slog.TextHandler` which produces output in the following format:

```
time=2024-01-15T10:30:45.123Z level=INFO msg="Compiling workflow: example.md"
time=2024-01-15T10:30:45.456Z level=DEBUG msg="Processing step: 5" step=5 duration=1.2s
time=2024-01-15T10:30:45.789Z level=WARN msg="Validation warning: schema mismatch"
time=2024-01-15T10:30:46.012Z level=ERROR msg="Compilation failed" error="invalid syntax"
```

## Log Levels

- **DEBUG**: Detailed diagnostic information (only shown in verbose mode)
- **INFO**: Informational messages about normal operations
- **WARN**: Warning messages about potential issues
- **ERROR**: Error messages about failures

## Best Practices

1. **Use Debug for Detailed Information**: Reserve debug logs for detailed diagnostic information that's only needed when troubleshooting
2. **Use Structured Fields**: Prefer `*WithFields` methods over string formatting when logging structured data
3. **Check Verbose Mode**: Use `IsVerbose()` to avoid expensive operations when debug logging is disabled
4. **Consistent Field Names**: Use consistent field names across log messages (e.g., always use "workflow" not "workflow_name")
5. **Log Errors with Context**: Include relevant context when logging errors

## Examples

### Basic Compiler Logging

```go
c := NewCompiler(verbose, "", "1.0.0")

c.logger.Infof("Starting compilation of: %s", workflowPath)
c.logger.DebugWithFields("Workflow data loaded",
    "name", data.Name,
    "engine", data.EngineConfig.ID,
)

if err := validate(data); err != nil {
    c.logger.ErrorWithFields("Validation failed",
        "workflow", workflowPath,
        "error", err.Error(),
    )
    return err
}

c.logger.Infof("Compilation successful")
```

### Testing with Custom Logger

```go
func TestCompilerLogging(t *testing.T) {
    var buf bytes.Buffer
    c := NewCompiler(true, "", "1.0.0")
    c.SetLogger(logging.NewLoggerWithWriter(true, &buf))

    // Perform operations...

    output := buf.String()
    if !strings.Contains(output, "expected message") {
        t.Errorf("Expected log message not found")
    }
}
```
