# Console Formatting Guide

This guide provides comprehensive documentation for console output formatting in GitHub Agentic Workflows. The console package provides a unified, styled, TTY-aware output system using Lipgloss for rich terminal formatting.

## Table of Contents

- [Console Formatters](#console-formatters)
- [Lipgloss Usage](#lipgloss-usage)
- [Table Rendering](#table-rendering)
- [Layout Helpers](#layout-helpers)
- [Struct Tag-Based Rendering](#struct-tag-based-rendering)
- [Huh Forms](#huh-forms)
- [TTY Detection](#tty-detection)
- [Best Practices](#best-practices)
- [Anti-Patterns](#anti-patterns)
- [Testing Console Output](#testing-console-output)

## Console Formatters

The console package provides semantic formatting functions for different message types. All formatters automatically adapt to TTY vs non-TTY environments.

### Message Type Formatters

```go
import (
    "fmt"
    "os"
    "github.com/githubnext/gh-aw/pkg/console"
)

// Success messages - Use for completed operations
fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Operation completed successfully"))
// Output: ✓ Operation completed successfully

// Error messages - Use for failures and critical issues
fmt.Fprintln(os.Stderr, console.FormatErrorMessage("Failed to process workflow"))
// Output: ✗ Failed to process workflow

// Warning messages - Use for non-critical issues
fmt.Fprintln(os.Stderr, console.FormatWarningMessage("File has uncommitted changes"))
// Output: ⚠ File has uncommitted changes

// Info messages - Use for general information
fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Processing 5 workflows..."))
// Output: ℹ Processing 5 workflows...
```

### Additional Formatters

```go
// Location/directory messages
fmt.Fprintln(os.Stderr, console.FormatLocationMessage("Output saved to .github/aw/logs"))
// Output: 📁 Output saved to .github/aw/logs

// Command execution messages
fmt.Fprintln(os.Stderr, console.FormatCommandMessage("Running gh aw compile"))
// Output: ⚡ Running gh aw compile

// Progress/activity messages
fmt.Fprintln(os.Stderr, console.FormatProgressMessage("Compiling workflows..."))
// Output: 🔨 Compiling workflows...

// User prompt messages
fmt.Fprintln(os.Stderr, console.FormatPromptMessage("Enter workflow name:"))
// Output: ❓ Enter workflow name:

// Count/numeric status messages
fmt.Fprintln(os.Stderr, console.FormatCountMessage("Found 12 workflows"))
// Output: 📊 Found 12 workflows

// Verbose debugging output
fmt.Fprintln(os.Stderr, console.FormatVerboseMessage("Debug: Token count = 1234"))
// Output: 🔍 Debug: Token count = 1234
```

### List Formatting

```go
// Section headers
fmt.Fprintln(os.Stderr, console.FormatListHeader("Available Workflows:"))

// List items
fmt.Fprintln(os.Stderr, console.FormatListItem("issue-triage.md"))
fmt.Fprintln(os.Stderr, console.FormatListItem("pr-review.md"))
// Output:
//   • issue-triage.md
//   • pr-review.md
```

### Error Messages with Suggestions

```go
suggestions := []string{
    "Run `gh aw compile` to validate the workflow",
    "Check the workflow frontmatter for syntax errors",
    "Ensure all required fields are present",
}

fmt.Fprintln(os.Stderr, console.FormatErrorWithSuggestions(
    "Workflow validation failed",
    suggestions,
))
```

### **Critical Rule: Always Use stderr for Console Output**

```go
// ✅ CORRECT - All console output to stderr
fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Success"))
fmt.Fprintln(os.Stderr, console.FormatErrorMessage(err.Error()))

// ❌ INCORRECT - Never use stdout for console messages
fmt.Println(console.FormatSuccessMessage("Success"))
fmt.Printf("Error: %v\n", err)
```

**Exception:** JSON output goes to stdout; all other output to stderr.

## Lipgloss Usage

Lipgloss provides styled terminal output with automatic color adaptation for light and dark themes.

### Adaptive Colors

The styles package defines adaptive colors that work in both light and dark terminals:

```go
import (
    "github.com/charmbracelet/lipgloss"
    "github.com/githubnext/gh-aw/pkg/styles"
)

// Semantic colors with light/dark variants
// ColorError - Red for errors (#D73737 light / #FF5555 dark)
// ColorWarning - Orange for warnings (#E67E22 light / #FFB86C dark)
// ColorSuccess - Green for success (#27AE60 light / #50FA7B dark)
// ColorInfo - Cyan for info (#2980B9 light / #8BE9FD dark)
// ColorPurple - Purple for highlights (#8E44AD light / #BD93F9 dark)
// ColorYellow - Yellow for progress (#B7950B light / #F1FA8C dark)
```

### Pre-configured Styles

```go
// Using pre-configured styles
errorText := styles.Error.Render("Something went wrong")
successText := styles.Success.Render("Operation completed")
infoText := styles.Info.Render("Processing...")

// Custom styles with adaptive colors
customStyle := lipgloss.NewStyle().
    Foreground(styles.ColorInfo).
    Bold(true).
    Padding(0, 1)
output := customStyle.Render("Custom styled text")
```

### Border Styles

```go
// Available border styles
// RoundedBorder - Soft, polished appearance
// NormalBorder - Clean, simple borders for tables
// ThickBorder - High-emphasis, critical information

boxStyle := lipgloss.NewStyle().
    Border(styles.RoundedBorder).
    BorderForeground(styles.ColorInfo).
    Padding(1, 2)
```

## Table Rendering

The `RenderTable` function provides consistent, styled table output with automatic zebra striping and TTY adaptation.

### Basic Table

```go
config := console.TableConfig{
    Title: "Workflow Status",
    Headers: []string{"Name", "Status", "Duration"},
    Rows: [][]string{
        {"issue-triage", "success", "2m 30s"},
        {"pr-review", "running", "1m 15s"},
        {"code-scan", "failed", "45s"},
    },
}

fmt.Fprint(os.Stderr, console.RenderTable(config))
```

### Table with Total Row

```go
config := console.TableConfig{
    Headers: []string{"Workflow", "Runs", "Cost"},
    Rows: [][]string{
        {"workflow-1", "25", "$0.50"},
        {"workflow-2", "15", "$0.30"},
    },
    ShowTotal: true,
    TotalRow: []string{"Total", "40", "$0.80"},
}

fmt.Fprint(os.Stderr, console.RenderTable(config))
```

### JSON Table Output

For machine-readable output:

```go
jsonOutput, err := console.RenderTableAsJSON(config)
if err != nil {
    return fmt.Errorf("failed to render table as JSON: %w", err)
}
fmt.Println(jsonOutput) // JSON goes to stdout
```

### Table Features

- **Automatic zebra striping** - Alternating row colors for readability
- **TTY detection** - Styled output in terminals, plain text when piped
- **Header styling** - Bold, muted headers
- **Total row styling** - Bold, green totals
- **Border styling** - Adaptive border colors

## Layout Helpers

Layout helpers provide reusable patterns for composing styled CLI output.

### LayoutTitleBox

Creates a centered title with double border:

```go
title := console.LayoutTitleBox("Trial Execution Plan", 60)
fmt.Fprintln(os.Stderr, title)
```

**TTY Output:**
```
═════════════════════════════════════════════════════════════
  Trial Execution Plan
═════════════════════════════════════════════════════════════
```

**Non-TTY Output:**
```
============================================================
  Trial Execution Plan
============================================================
```

### LayoutInfoSection

Creates an info section with left border emphasis:

```go
info := console.LayoutInfoSection("Workflow", "test-workflow")
fmt.Fprintln(os.Stderr, info)
```

**TTY Output:**
```
│  Workflow: test-workflow
```

**Non-TTY Output:**
```
  Workflow: test-workflow
```

### LayoutEmphasisBox

Creates a thick-bordered box with custom color:

```go
warning := console.LayoutEmphasisBox(
    "⚠️ WARNING: Large workflow file",
    styles.ColorWarning,
)
fmt.Fprintln(os.Stderr, warning)
```

### LayoutJoinVertical

Composes multiple sections vertically:

```go
title := console.LayoutTitleBox("Plan", 60)
info1 := console.LayoutInfoSection("Workflow", "test")
info2 := console.LayoutInfoSection("Status", "Ready")

output := console.LayoutJoinVertical(title, "", info1, info2)
fmt.Fprintln(os.Stderr, output)
```

### Legacy Rendering Functions

For backward compatibility, these functions return `[]string`:

```go
// RenderTitleBox - Returns []string for manual composition
sections := []string{}
sections = append(sections, console.RenderTitleBox("Title", 60)...)
sections = append(sections, console.RenderInfoSection("Content")...)

// RenderComposedSections - Outputs to stderr directly
console.RenderComposedSections(sections)
```

## Struct Tag-Based Rendering

The `RenderStruct` function provides automatic rendering of Go structs using struct tags.

### Basic Struct Rendering

```go
type Overview struct {
    RunID    int64  `console:"header:Run ID"`
    Workflow string `console:"header:Workflow"`
    Status   string `console:"header:Status"`
    Duration string `console:"header:Duration,omitempty"`
}

data := Overview{
    RunID:    12345,
    Workflow: "test-workflow",
    Status:   "completed",
    Duration: "5m30s",
}

fmt.Print(console.RenderStruct(data))
```

**Output:**
```
Run ID  : 12345
Workflow: test-workflow
Status  : completed
Duration: 5m30s
```

### Struct Tag Options

| Tag | Description | Example |
|-----|-------------|---------|
| `header:"Name"` | Display name for field | `header:"Run ID"` |
| `title:"Title"` | Title for nested structs/slices | `title:"Jobs"` |
| `format:"type"` | Format type for value | `format:number` |
| `omitempty` | Skip zero values | `omitempty` |
| `"-"` | Always skip field | `console:"-"` |
| `default:"value"` | Default value for zero fields | `default:N/A` |
| `maxlen:N` | Truncate to N characters | `maxlen:50` |

### Format Types

```go
// Number formatting - "1k", "1.2M", "1.12B"
type Metrics struct {
    TokenUsage int `console:"header:Token Usage,format:number"`
}

// Cost formatting - "$1.234"
type Billing struct {
    Cost float64 `console:"header:Cost,format:cost"`
}

// File size formatting - "1.2 MB", "3.4 KB"
type Storage struct {
    Size int64 `console:"header:Size,format:filesize"`
}
```

### Table Rendering from Slices

Slices of structs are automatically rendered as tables:

```go
type Job struct {
    Name       string `console:"header:Name"`
    Status     string `console:"header:Status"`
    Conclusion string `console:"header:Conclusion,omitempty"`
}

jobs := []Job{
    {Name: "build", Status: "completed", Conclusion: "success"},
    {Name: "test", Status: "completed", Conclusion: "success"},
    {Name: "deploy", Status: "running", Conclusion: ""},
}

fmt.Print(console.RenderStruct(jobs))
```

**Output:**
```
Name   | Status    | Conclusion
-------|-----------|------------
build  | completed | success
test   | completed | success
deploy | running   | 
```

## Huh Forms

Huh provides accessible, interactive forms for CLI input with full keyboard navigation.

### Basic Form

```go
import (
    "github.com/charmbracelet/huh"
    "os"
)

var workflowName string

form := huh.NewForm(
    huh.NewGroup(
        huh.NewInput().
            Title("What should we call this workflow?").
            Description("Enter a descriptive name").
            Value(&workflowName).
            Validate(ValidateWorkflowName),
    ),
).WithAccessible(isAccessibleMode())

if err := form.Run(); err != nil {
    return err
}
```

### Multi-Page Form with Groups

```go
var (
    engine      string
    tools       []string
    networkAccess string
)

form := huh.NewForm(
    // Page 1: Basic Configuration
    huh.NewGroup(
        huh.NewSelect[string]().
            Title("Which AI engine should process this workflow?").
            Options(
                huh.NewOption("copilot", "copilot"),
                huh.NewOption("claude", "claude"),
            ).
            Value(&engine),
    ).
        Title("Basic Configuration").
        Description("Choose your AI engine"),

    // Page 2: Capabilities
    huh.NewGroup(
        huh.NewMultiSelect[string]().
            Title("Which tools should the AI have access to?").
            Options(
                huh.NewOption("github - GitHub API tools", "github"),
                huh.NewOption("bash - Shell commands", "bash"),
            ).
            Height(8).
            Value(&tools),
    ).
        Title("Capabilities").
        Description("Select available tools"),
).WithAccessible(isAccessibleMode())

if err := form.Run(); err != nil {
    return err
}
```

### Form Input Types

| Type | Use Case | Example |
|------|----------|---------|
| `Input` | Single-line text | Workflow name, file path |
| `Text` | Multi-line text | Workflow instructions |
| `Select` | Single choice | AI engine, trigger type |
| `MultiSelect` | Multiple choices | Tools, safe outputs |
| `Confirm` | Yes/No question | Overwrite confirmation |

### Accessibility Support

```go
// isAccessibleMode detects if accessibility mode should be enabled
func isAccessibleMode() bool {
    return os.Getenv("ACCESSIBLE") != "" ||
        os.Getenv("TERM") == "dumb" ||
        os.Getenv("NO_COLOR") != ""
}

// Always enable accessibility for better user experience
form := huh.NewForm(...).WithAccessible(isAccessibleMode())
```

**Accessibility Features:**
- Screen reader support
- High contrast mode
- Keyboard-only navigation
- Clear labels and descriptions

### Form Validation

```go
// Custom validation function
func ValidateWorkflowName(name string) error {
    if len(name) == 0 {
        return fmt.Errorf("workflow name cannot be empty")
    }
    if len(name) > 100 {
        return fmt.Errorf("workflow name too long (max 100 characters)")
    }
    if !regexp.MustCompile(`^[a-z0-9-]+$`).MatchString(name) {
        return fmt.Errorf("workflow name must be lowercase alphanumeric with hyphens")
    }
    return nil
}

huh.NewInput().
    Value(&workflowName).
    Validate(ValidateWorkflowName)
```

### Confirmation Dialogs

```go
var overwrite bool

confirmForm := huh.NewForm(
    huh.NewGroup(
        huh.NewConfirm().
            Title("Workflow file already exists. Overwrite?").
            Affirmative("Yes, overwrite").
            Negative("No, cancel").
            Value(&overwrite),
    ),
).WithAccessible(isAccessibleMode())

if err := confirmForm.Run(); err != nil {
    return err
}

if !overwrite {
    return fmt.Errorf("operation cancelled")
}
```

## TTY Detection

The console package automatically detects TTY environments and adapts output accordingly.

### Automatic TTY Detection

```go
import "github.com/githubnext/gh-aw/pkg/tty"

// Check if stderr is a terminal
if tty.IsStderrTerminal() {
    // Show styled output with colors and borders
    styled := lipgloss.NewStyle().Foreground(styles.ColorInfo).Render("Info")
    fmt.Fprintln(os.Stderr, styled)
} else {
    // Plain text for pipes/redirects
    fmt.Fprintln(os.Stderr, "Info")
}

// Check if stdout is a terminal
if tty.IsStdoutTerminal() {
    // Interactive table
    console.RenderTable(config)
} else {
    // JSON output
    jsonOutput, _ := console.RenderTableAsJSON(config)
    fmt.Println(jsonOutput)
}
```

### TTY vs Non-TTY Output

| Feature | TTY Output | Non-TTY Output |
|---------|------------|----------------|
| **Colors** | Full ANSI colors | No colors |
| **Borders** | Unicode box drawing | ASCII characters |
| **Icons** | Emoji (✓, ✗, ⚠) | ASCII (*, !, ?) |
| **Tables** | Styled with zebra striping | Plain with pipes |
| **Layout** | Lipgloss composition | Simple line breaks |

### Manual TTY Control

```go
import "github.com/githubnext/gh-aw/pkg/console"

// applyStyle conditionally applies styling
func displayMessage(message string) {
    if tty.IsStderrTerminal() {
        styled := styles.Info.Render(message)
        fmt.Fprintln(os.Stderr, styled)
    } else {
        fmt.Fprintln(os.Stderr, message)
    }
}
```

## Best Practices

### 1. Use Semantic Formatters

Always use the appropriate formatter for the message type:

```go
// ✅ CORRECT - Semantic formatters
fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Compilation succeeded"))
fmt.Fprintln(os.Stderr, console.FormatErrorMessage(err.Error()))
fmt.Fprintln(os.Stderr, console.FormatWarningMessage("File modified"))

// ❌ INCORRECT - Manual formatting
fmt.Fprintln(os.Stderr, "✓ Compilation succeeded")
fmt.Fprintf(os.Stderr, "Error: %v\n", err)
fmt.Fprintln(os.Stderr, "Warning: File modified")
```

### 2. Always Use stderr for Console Output

```go
// ✅ CORRECT - All user-facing messages to stderr
fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Processing..."))
fmt.Fprint(os.Stderr, console.RenderTable(config))
fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Done"))

// ❌ INCORRECT - stdout is for data output only
fmt.Println(console.FormatInfoMessage("Processing..."))
console.RenderTable(config) // Goes to stdout by default if using Print
```

### 3. Use Struct Tags for Complex Data

```go
// ✅ CORRECT - Struct tags for automatic rendering
type WorkflowStatus struct {
    Name     string `console:"header:Workflow"`
    Status   string `console:"header:Status"`
    Duration string `console:"header:Duration,omitempty"`
}

fmt.Print(console.RenderStruct(status))

// ❌ INCORRECT - Manual formatting
fmt.Printf("Workflow: %s\n", status.Name)
fmt.Printf("Status: %s\n", status.Status)
fmt.Printf("Duration: %s\n", status.Duration)
```

### 4. Compose Layouts with Helpers

```go
// ✅ CORRECT - Layout helpers for composition
title := console.LayoutTitleBox("Plan", 60)
info := console.LayoutInfoSection("Status", "Ready")
output := console.LayoutJoinVertical(title, "", info)
fmt.Fprintln(os.Stderr, output)

// ❌ INCORRECT - Manual composition
fmt.Fprintln(os.Stderr, "=======================================")
fmt.Fprintln(os.Stderr, "  Plan")
fmt.Fprintln(os.Stderr, "=======================================")
fmt.Fprintln(os.Stderr, "")
fmt.Fprintln(os.Stderr, "  Status: Ready")
```

### 5. Enable Accessibility in Forms

```go
// ✅ CORRECT - Always check accessibility mode
form := huh.NewForm(...).WithAccessible(isAccessibleMode())

// ❌ INCORRECT - Hardcoded accessibility
form := huh.NewForm(...).WithAccessible(false)
```

### 6. Provide Helpful Error Messages

```go
// ✅ CORRECT - Actionable error with suggestions
suggestions := []string{
    "Run `gh aw compile` to validate the workflow",
    "Check the YAML syntax in the frontmatter",
}
fmt.Fprintln(os.Stderr, console.FormatErrorWithSuggestions(
    "Workflow validation failed",
    suggestions,
))

// ❌ INCORRECT - Generic error
fmt.Fprintln(os.Stderr, console.FormatErrorMessage("Error"))
```

## Anti-Patterns

### 1. Direct stdout Usage

```go
// ❌ INCORRECT - Using stdout for console messages
fmt.Println("Processing workflows...")
fmt.Printf("Status: %s\n", status)

// ✅ CORRECT - Use stderr with formatters
fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Processing workflows..."))
fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Status: %s", status)))
```

### 2. Manual Formatting

```go
// ❌ INCORRECT - Manual ANSI codes and formatting
fmt.Fprintln(os.Stderr, "\033[32m✓\033[0m Operation completed")
fmt.Fprintln(os.Stderr, "\033[31m✗\033[0m Operation failed")

// ✅ CORRECT - Use console formatters
fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Operation completed"))
fmt.Fprintln(os.Stderr, console.FormatErrorMessage("Operation failed"))
```

### 3. Missing TTY Checks

```go
// ❌ INCORRECT - Always styled output
styled := styles.Error.Render("Error message")
fmt.Fprintln(os.Stderr, styled)

// ✅ CORRECT - Console formatters handle TTY automatically
fmt.Fprintln(os.Stderr, console.FormatErrorMessage("Error message"))

// Or manual TTY check when needed
if tty.IsStderrTerminal() {
    styled := styles.Error.Render("Error message")
    fmt.Fprintln(os.Stderr, styled)
} else {
    fmt.Fprintln(os.Stderr, "Error message")
}
```

### 4. Inconsistent Message Formatting

```go
// ❌ INCORRECT - Mix of formatting styles
fmt.Fprintln(os.Stderr, "✓ Success")
fmt.Fprintln(os.Stderr, console.FormatErrorMessage("Error"))
fmt.Fprintln(os.Stderr, "[INFO] Processing")

// ✅ CORRECT - Consistent formatters
fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Success"))
fmt.Fprintln(os.Stderr, console.FormatErrorMessage("Error"))
fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Processing"))
```

### 5. Ignoring Accessibility

```go
// ❌ INCORRECT - No accessibility support
form := huh.NewForm(
    huh.NewGroup(
        huh.NewInput().
            Title("Name").
            Value(&name),
    ),
)

// ✅ CORRECT - Enable accessibility
form := huh.NewForm(
    huh.NewGroup(
        huh.NewInput().
            Title("What is your name?").
            Description("Enter a descriptive name").
            Value(&name),
    ),
).WithAccessible(isAccessibleMode())
```

### 6. Poor Table Structure

```go
// ❌ INCORRECT - Manual table formatting
fmt.Fprintln(os.Stderr, "Name\t\tStatus")
fmt.Fprintln(os.Stderr, "----\t\t------")
fmt.Fprintf(os.Stderr, "%s\t\t%s\n", name, status)

// ✅ CORRECT - Use RenderTable
config := console.TableConfig{
    Headers: []string{"Name", "Status"},
    Rows: [][]string{{name, status}},
}
fmt.Fprint(os.Stderr, console.RenderTable(config))
```

## Testing Console Output

### Testing TTY Detection

```go
func TestConsoleOutput(t *testing.T) {
    // Test with both TTY and non-TTY environments
    tests := []struct {
        name    string
        isTTY   bool
        message string
        want    string
    }{
        {
            name:    "TTY output includes styling",
            isTTY:   true,
            message: "Success",
            want:    "✓", // Check for emoji/icon
        },
        {
            name:    "Non-TTY output is plain text",
            isTTY:   false,
            message: "Success",
            want:    "Success", // No styling
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            output := console.FormatSuccessMessage(tt.message)
            if tt.isTTY {
                assert.Contains(t, output, tt.want)
            } else {
                assert.NotContains(t, output, "\033[") // No ANSI codes
            }
        })
    }
}
```

### Testing Formatters

```go
func TestFormatters(t *testing.T) {
    tests := []struct {
        name      string
        formatter func(string) string
        message   string
        want      string
    }{
        {
            name:      "success message",
            formatter: console.FormatSuccessMessage,
            message:   "Done",
            want:      "Done",
        },
        {
            name:      "error message",
            formatter: console.FormatErrorMessage,
            message:   "Failed",
            want:      "Failed",
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := tt.formatter(tt.message)
            assert.Contains(t, result, tt.want)
        })
    }
}
```

### Testing Table Rendering

```go
func TestTableRendering(t *testing.T) {
    config := console.TableConfig{
        Headers: []string{"Name", "Status"},
        Rows: [][]string{
            {"test-1", "success"},
            {"test-2", "failed"},
        },
    }

    // Test styled output
    styled := console.RenderTable(config)
    assert.Contains(t, styled, "Name")
    assert.Contains(t, styled, "test-1")

    // Test JSON output
    json, err := console.RenderTableAsJSON(config)
    assert.NoError(t, err)
    assert.Contains(t, json, `"name":"test-1"`)
}
```

### Testing Struct Rendering

```go
func TestStructRendering(t *testing.T) {
    type TestStruct struct {
        Name  string `console:"header:Name"`
        Value int    `console:"header:Value,format:number"`
    }

    data := TestStruct{Name: "test", Value: 1000}
    output := console.RenderStruct(data)

    assert.Contains(t, output, "Name")
    assert.Contains(t, output, "test")
    assert.Contains(t, output, "1k") // Formatted number
}
```

## Summary

The console formatting system in GitHub Agentic Workflows provides:

1. **Semantic formatters** - Clear, consistent message formatting
2. **TTY adaptation** - Automatic styling based on output destination
3. **Lipgloss integration** - Adaptive colors for light/dark themes
4. **Table rendering** - Professional table output with styling
5. **Layout helpers** - Reusable composition patterns
6. **Struct tag rendering** - Automatic formatting from Go structs
7. **Huh forms** - Accessible, interactive CLI forms
8. **Testing support** - Comprehensive testing utilities

By following these patterns and avoiding anti-patterns, you'll create consistent, accessible, and professional CLI output that works seamlessly in all environments.
