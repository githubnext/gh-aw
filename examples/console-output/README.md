# Console Output Examples

This directory contains examples demonstrating the console formatting system in GitHub Agentic Workflows.

## Examples

### 1. Simple Output (`simple-output.go`)

Demonstrates basic console formatters for different message types:
- Success, error, warning, and info messages
- Location, command, and progress messages
- List formatting
- Error messages with suggestions

**Run:**
```bash
go run examples/console-output/simple-output.go
```

**Key concepts:**
- Using semantic formatters (`FormatSuccessMessage`, `FormatErrorMessage`, etc.)
- All output goes to stderr (`os.Stderr`)
- Automatic TTY detection and styling

### 2. Table Example (`table-example.go`)

Demonstrates table rendering with various configurations:
- Basic tables with headers and rows
- Tables with total rows
- JSON table output
- Zebra striping for large datasets

**Run:**
```bash
go run examples/console-output/table-example.go
```

**Key concepts:**
- `TableConfig` structure
- `RenderTable` function
- `RenderTableAsJSON` for machine-readable output
- Automatic styling and adaptation

### 3. Huh Form Example (`huh-form-example.go`)

Demonstrates interactive forms with Huh:
- Simple input forms with validation
- Multi-page forms with different input types
- Confirmation dialogs
- Accessibility support

**Run:**
```bash
go run examples/console-output/huh-form-example.go
```

**Key concepts:**
- Form groups and pages
- Input types (Input, Select, MultiSelect, Text, Confirm)
- Validation functions
- Accessibility mode detection

## Running Examples

### Prerequisites

Make sure you're in the repository root directory:

```bash
cd /path/to/gh-aw
```

### Run with Default Settings

```bash
# Simple output
go run examples/console-output/simple-output.go

# Table example
go run examples/console-output/table-example.go

# Huh form (interactive)
go run examples/console-output/huh-form-example.go
```

### Test TTY Detection

**With TTY (terminal):**
```bash
go run examples/console-output/simple-output.go
# Output: Styled with colors and icons
```

**Without TTY (redirected):**
```bash
go run examples/console-output/simple-output.go 2>&1 | cat
# Output: Plain text without styling
```

### Test Accessibility Mode

Enable accessibility mode for better screen reader support:

```bash
# Set accessibility environment variable
ACCESSIBLE=1 go run examples/console-output/huh-form-example.go

# Or use NO_COLOR
NO_COLOR=1 go run examples/console-output/huh-form-example.go
```

## Key Takeaways

1. **Always use stderr** for console messages (`fmt.Fprintln(os.Stderr, ...)`)
2. **Use semantic formatters** for consistent styling
3. **TTY detection is automatic** - no manual checks needed for formatters
4. **Enable accessibility** in forms with `WithAccessible(isAccessibleMode())`
5. **JSON output goes to stdout** - everything else goes to stderr

## Documentation

For complete documentation on console formatting, see:
- `specs/console-formatting.md` - Comprehensive guide
- `pkg/console/README.md` - Package documentation
- `skills/console-rendering/SKILL.md` - Struct tag rendering system

## Related Examples

- `pkg/cli/interactive.go` - Real-world interactive workflow builder
- `pkg/cli/audit_report_render.go` - Complex table rendering
- `pkg/cli/mcp_inspect.go` - Layout composition examples
