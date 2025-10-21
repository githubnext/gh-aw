# Logger Package

A simple, debug-style logging framework for Go that follows the pattern matching syntax of the [debug npm package](https://www.npmjs.com/package/debug).

## Features

- **Namespace-based logging**: Each logger has a namespace (e.g., `workflow:compiler`, `cli:audit`)
- **Pattern matching**: Enable/disable loggers using wildcards and exclusions via the `DEBUG` environment variable
- **Printf interface**: Standard printf-style formatting
- **Lazy evaluation**: Compute expensive strings only when the logger is enabled
- **Zero overhead**: Logger enabled state is computed once at construction time
- **Thread-safe**: Safe for concurrent use with internal caching

## Usage

### Basic Usage

```go
package main

import "github.com/githubnext/gh-aw/pkg/logger"

var log = logger.New("myapp:feature")

func main() {
    log.Printf("Starting application with config: %s", config)
    log.Print("Multiple", " ", "arguments")
    log.Println("Single line message")
}
```

### Lazy Evaluation

For expensive string operations, use `LazyPrintf` to avoid computation when the logger is disabled:

```go
log.LazyPrintf(func() string {
    // This expensive computation only runs if the logger is enabled
    data := fetchLargeData()
    return fmt.Sprintf("Large data: %+v", data)
})
```

### Checking if Logger is Enabled

You can check if a logger is enabled before performing expensive operations:

```go
if log.Enabled() {
    // Do expensive work only if logging is enabled
    result := expensiveOperation()
    log.Printf("Result: %v", result)
}
```

## DEBUG Environment Variable

Control which loggers are enabled using the `DEBUG` environment variable with patterns:

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
- Pattern matching results are cached for performance

### Output Destination

All log output goes to **stderr** to avoid interfering with stdout data (JSON, command output, etc.).

### Printf Interface

The logger provides a familiar printf-style interface that Go developers expect:

- `Printf(format, args...)` - Formatted output
- `Print(args...)` - Simple concatenation
- `Println(args...)` - Line-based output (alias of Print)
- `LazyPrintf(func() string)` - Deferred computation

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

## Implementation Notes

- The `DEBUG` environment variable is read once when the package is initialized
- Pattern matching results are cached in memory to avoid repeated computation
- Thread-safe using `sync.RWMutex` for cache access
- Simple pattern matching without regex (prefix, suffix, and middle wildcards only)
- Exclusion patterns (prefixed with `-`) take precedence over inclusion patterns
