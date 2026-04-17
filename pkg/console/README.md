# console Package

The `console` package provides utilities for formatting and rendering terminal output in GitHub Agentic Workflows. It covers message formatting, table and section rendering, interactive prompts, progress bars, spinners, struct rendering, and accessibility support.

## Design Philosophy

This package follows Charmbracelet best practices for terminal UI:

- **Adaptive Colors**: All styling uses `lipgloss.AdaptiveColor` for light/dark theme support
- **Rounded Borders**: Tables and boxes use rounded corners (╭╮╰╯) for a polished appearance
- **Consistent Padding**: All rendered elements include proper spacing (horizontal and vertical)
- **TTY Detection**: Automatically adapts output for terminals vs pipes/redirects
- **Visual Hierarchy**: Clear separation between sections using borders and spacing
- **Zebra Striping**: Tables use alternating row colors for improved readability

### Border Usage Guidelines

- **RoundedBorder** (primary): Use for tables, boxes, and panels
  - Creates a polished, modern appearance
  - Consistent with Charmbracelet design language
- **NormalBorder** (subtle): Use for left-side emphasis on info sections
  - Provides gentle visual guidance without overwhelming
- **ThickBorder** (reserved): Available for special cases requiring extra emphasis
  - Use sparingly - rounded borders with bold text usually suffice

### Padding Guidelines

- **Table cells**: 1 character horizontal padding (left/right)
- **Boxes**: 2 character horizontal padding, 0-1 vertical padding
- **Info sections**: 2 character left padding for consistent indentation

## Spinner Component

The `Spinner` component provides animated visual feedback during long-running operations with automatic TTY detection and accessibility support.

### Features

- **MiniDot animation**: Minimal dot spinner (⣾ ⣽ ⣻ ⢿ ⡿ ⣟ ⣯ ⣷)
- **TTY detection**: Automatically disabled in pipes/redirects
- **Accessibility support**: Respects `ACCESSIBLE` environment variable
- **Color adaptation**: Uses adaptive colors for light/dark themes
- **Idiomatic Bubble Tea**: Uses `tea.NewProgram()` with proper message passing
- **Thread-safe**: Safe for concurrent use via Bubble Tea's message handling

### Usage

```go
import "github.com/github/gh-aw/pkg/console"

// Create and use a spinner
spinner := console.NewSpinner("Loading...")
spinner.Start()
// Long-running operation
spinner.Stop()

// Stop with a message
spinner := console.NewSpinner("Processing...")
spinner.Start()
// Long-running operation
spinner.StopWithMessage("✓ Done!")

// Update message while running
spinner := console.NewSpinner("Starting...")
spinner.Start()
spinner.UpdateMessage("Still working...")
// Long-running operation
spinner.Stop()
```

### Accessibility

The spinner respects the `ACCESSIBLE` environment variable. When set to any value, spinner animations are disabled to support screen readers and accessibility tools:

```bash
export ACCESSIBLE=1
gh aw compile workflow.md  # Spinners will be disabled
```

### TTY Detection

Spinners only animate in terminal environments. When output is piped or redirected, the spinner is automatically disabled:

```bash
gh aw compile workflow.md           # Spinner animates
gh aw compile workflow.md > log.txt # Spinner disabled
```

## ProgressBar Component

The `ProgressBar` component provides a reusable progress bar with TTY detection and graceful fallback for non-TTY environments.

### Features

- **Scaled gradient effect**: Smooth color transition from purple to cyan as progress advances
- **TTY detection**: Automatically adapts to terminal environment
- **Byte formatting**: Converts byte counts to human-readable sizes (KB, MB, GB)
- **Thread-safe updates**: Safe for concurrent use with atomic operations

### Visual Styling

The progress bar uses bubbles v0.21.0+ gradient capabilities for enhanced visual appeal:
- **Start (0%)**: #BD93F9 (purple) - vibrant, attention-grabbing
- **End (100%)**: #8BE9FD (cyan) - cool, completion feeling
- **Empty portion**: #6272A4 (muted purple-gray)
- **Gradient scaling**: WithScaledGradient ensures gradient scales with filled portion

### Usage

#### Determinate Mode (known total)
Use when the total size or count is known:

```go
import "github.com/github/gh-aw/pkg/console"

// Create a progress bar for 1GB total
totalBytes := int64(1024 * 1024 * 1024)
bar := console.NewProgressBar(totalBytes)

// Update progress (returns formatted string)
output := bar.Update(currentBytes)
fmt.Fprintf(os.Stderr, "\r%s", output)
```

#### Indeterminate Mode (unknown total)
Use when the total size or count is unknown:

```go
import "github.com/github/gh-aw/pkg/console"

// Create an indeterminate progress bar
bar := console.NewIndeterminateProgressBar()

// Update with current progress (shows activity without percentage)
output := bar.Update(currentBytes)
fmt.Fprintf(os.Stderr, "\r%s", output)
```

### Output Examples

**Determinate Mode - TTY**:
```
████████████████████░░░░░░░░░░░░░░░░░  50%
```
*(Displays with gradient from purple to cyan)*

**Determinate Mode - Non-TTY**:
```
50% (512.0MB/1.00GB)
```

**Indeterminate Mode - TTY**:
```
████████████████░░░░░░░░░░░░░░░░░░░░  (pulsing animation)
```
*(Shows pulsing progress indicator)*

**Indeterminate Mode - Non-TTY**:
```
Processing... (512.0MB)
```


## RenderStruct Function

The `RenderStruct` function uses reflection to automatically render Go structs based on struct tags.

### Struct Tags

Use the `console` struct tag to control rendering behavior:

#### Available Tags

- **`header:"Column Name"`** - Sets the display name for the field (used in both structs and tables)
- **`title:"Section Title"`** - Sets the title for nested structs, slices, or maps
- **`omitempty`** - Skips the field if it has a zero value
- **`"-"`** - Always skips the field

#### Tag Examples

```go
type Overview struct {
    RunID      int64  `console:"header:Run ID"`
    Workflow   string `console:"header:Workflow"`
    Status     string `console:"header:Status"`
    Duration   string `console:"header:Duration,omitempty"`
    Internal   string `console:"-"` // Never displayed
}
```

### Rendering Behavior

#### Structs
Structs are rendered as key-value pairs with proper alignment:

```
  Run ID    : 12345
  Workflow  : my-workflow
  Status    : completed
  Duration  : 5m30s
```

#### Slices
Slices of structs are automatically rendered as tables using the console table renderer:

```go
type Job struct {
    Name       string `console:"header:Name"`
    Status     string `console:"header:Status"`
    Conclusion string `console:"header:Conclusion,omitempty"`
}

jobs := []Job{
    {Name: "build", Status: "completed", Conclusion: "success"},
    {Name: "test", Status: "in_progress", Conclusion: ""},
}

fmt.Print(console.RenderStruct(jobs))
```

Renders as:

```
Name  | Status      | Conclusion
----- | ----------- | ----------
build | completed   | success
test  | in_progress | -
```

#### Maps
Maps are rendered as markdown-style headers with key-value pairs:

```go
data := map[string]string{
    "Repository": "github/gh-aw",
    "Author":     "test-user",
}

fmt.Print(console.RenderStruct(data))
```

Renders as:

```
  Repository: github/gh-aw
  Author    : test-user
```

### Special Type Handling

#### time.Time
`time.Time` fields are automatically formatted as `"2006-01-02 15:04:05"`. Zero time values are considered empty when used with `omitempty`.

#### Unexported Fields
The rendering system safely handles unexported struct fields by checking `CanInterface()` before attempting to access field values.

### Usage in Audit Command

The audit command uses the new rendering system for structured output:

```go
// Render overview section
renderOverview(data.Overview)

// Render metrics with custom formatting
renderMetrics(data.Metrics)

// Render jobs as a table
renderJobsTable(data.Jobs)
```

This provides:
- Consistent formatting across all audit sections
- Automatic table generation for slice data
- Proper handling of optional/empty fields
- Type-safe reflection-based rendering

### Migration Guide

To migrate existing rendering code to use the new system:

1. **Add struct tags** to your data types:
   ```go
   type MyData struct {
       Field1 string `console:"header:Field 1"`
       Field2 int    `console:"header:Field 2,omitempty"`
   }
   ```

2. **Use RenderStruct** for simple structs:
   ```go
   fmt.Print(console.RenderStruct(myData))
   ```

3. **Use custom rendering** for special formatting needs:
   ```go
   func renderMyData(data MyData) {
       fmt.Printf("  %-15s %s\n", "Field 1:", formatCustom(data.Field1))
       // ... custom formatting logic
   }
   ```

4. **Use console.RenderTable** for tables with custom formatting:
   ```go
   config := console.TableConfig{
       Headers: []string{"Name", "Value"},
       Rows: [][]string{
           {truncateString(item.Name, 40), formatNumber(item.Value)},
       },
   }
   fmt.Print(console.RenderTable(config))
   ```

## Message Formatting Functions

All `Format*` functions return a styled string ready to be printed to `os.Stderr`. Colors adapt automatically to the terminal background.

| Function | Style | Typical use |
|----------|-------|-------------|
| `FormatSuccessMessage(message string) string` | Green, bold | Operation completed successfully |
| `FormatInfoMessage(message string) string` | Cyan, bold | General informational output |
| `FormatWarningMessage(message string) string` | Orange, bold | Non-fatal warnings |
| `FormatErrorMessage(message string) string` | Red, bold | Recoverable error messages |
| `FormatCommandMessage(command string) string` | Purple | CLI commands and code snippets |
| `FormatProgressMessage(message string) string` | Yellow | In-progress status updates |
| `FormatPromptMessage(message string) string` | Cyan | Interactive prompt labels |
| `FormatVerboseMessage(message string) string` | Muted/comment | Verbose/debug detail |
| `FormatListItem(item string) string` | Foreground | Individual list entries |
| `FormatListHeader(header string) string` | Plain (WASM only) | Section headers inside lists |
| `FormatSectionHeader(header string) string` | Bold, bordered | Section titles in output |
| `FormatLocationMessage(message string) string` | Foreground (WASM only) | File and location paths |
| `FormatCountMessage(message string) string` | Foreground (WASM only) | Counts and metrics |

### Usage Pattern

```go
import (
    "fmt"
    "os"
    "github.com/github/gh-aw/pkg/console"
)

// Always write formatted messages to stderr
fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Workflow compiled successfully"))
fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Processing 3 files..."))
fmt.Fprintln(os.Stderr, console.FormatWarningMessage("Network access is unrestricted"))
fmt.Fprintln(os.Stderr, console.FormatErrorMessage("File not found: workflow.md"))
fmt.Fprintln(os.Stderr, console.FormatCommandMessage("gh aw compile workflow.md"))
fmt.Fprintln(os.Stderr, console.FormatProgressMessage("Downloading release..."))
```

## Error Formatting

### `FormatError(err CompilerError) string`

Formats a structured `CompilerError` with position information, source context lines, and an optional fix hint. Used by the compiler to display actionable error messages.

```go
err := console.CompilerError{
    Position: console.ErrorPosition{File: "workflow.md", Line: 12, Column: 5},
    Type:     "error",
    Message:  "unknown engine: 'myengine'",
    Context:  []string{"engine: myengine"},
    Hint:     "Valid engines are: copilot, claude, codex, gemini, opencode",
}
fmt.Fprint(os.Stderr, console.FormatError(err))
```

### `FormatErrorChain(err error) string`

Formats an error together with its entire `%w`-wrapped cause chain. Each level of the chain is shown on a new indented line for easy debugging.

```go
fmt.Fprintln(os.Stderr, console.FormatErrorChain(err))
```

## Section Rendering Functions

These functions return `[]string` slices (lines of output) that can be composed using `RenderComposedSections`.

### `RenderTitleBox(title string, width int) []string`

Returns a rounded-border box containing `title`, padded to at least `width` characters.

### `RenderErrorBox(title string) []string`

Returns a red-bordered error box displaying `title`.

### `RenderInfoSection(content string) []string`

Returns `content` wrapped in a left-bordered info section with muted styling.

### `RenderComposedSections(sections []string)`

Prints multiple rendered sections to `os.Stderr`, separated by blank lines.

```go
lines := append(
    console.RenderTitleBox("Audit Report", 60),
    console.RenderInfoSection("3 jobs completed")...,
)
console.RenderComposedSections(lines)
```

### `RenderTable(config TableConfig) string`

Renders a formatted table with optional title and total row. See the `TableConfig` type for configuration options.

```go
table := console.RenderTable(console.TableConfig{
    Headers: []string{"Name", "Status", "Duration"},
    Rows: [][]string{
        {"build", "success", "1m30s"},
        {"test",  "failure", "45s"},
    },
    Title: "Job Results",
})
fmt.Print(table)
```

### `RenderTree(root TreeNode) string`

Renders a `TreeNode` hierarchy as an indented tree string.

```go
tree := console.RenderTree(console.TreeNode{
    Value: "root",
    Children: []console.TreeNode{
        {Value: "child1"},
        {Value: "child2", Children: []console.TreeNode{{Value: "grandchild"}}},
    },
})
fmt.Print(tree)
```

## Types

### `CompilerError`

Structured error with source position, type, message, context lines, and a fix hint.

```go
type CompilerError struct {
    Position ErrorPosition // Source file position
    Type     string        // "error", "warning", "info"
    Message  string
    Context  []string      // Source lines shown around the error
    Hint     string        // Optional actionable fix suggestion
}
```

### `ErrorPosition`

```go
type ErrorPosition struct {
    File   string
    Line   int
    Column int
}
```

### `TableConfig`

```go
type TableConfig struct {
    Headers   []string
    Rows      [][]string
    Title     string   // Optional table title
    ShowTotal bool     // Display a total row
    TotalRow  []string // Content for the total row
}
```

### `TreeNode`

Represents a node in a hierarchical tree for tree-style rendering.

```go
type TreeNode struct {
    Value    string
    Children []TreeNode
}
```

### `SelectOption`

A labeled option for interactive select fields.

```go
type SelectOption struct {
    Label string
    Value string
}
```

### `FormField`

Configuration for a single field in an interactive form.

```go
type FormField struct {
    Type        string             // "input", "password", "confirm", "select"
    Title       string
    Description string
    Placeholder string
    Value       any                // Pointer to the field's result value
    Options     []SelectOption     // For "select" type
    Validate    func(string) error // For "input" and "password" types
}
```

### `ListItem`

An item in an interactive list with title, description, and an internal value. Create with `NewListItem(title, description, value string)`.

## Interactive Prompts

### `ConfirmAction(title, affirmative, negative string) (bool, error)`

Displays an interactive yes/no confirmation dialog using the `huh` library. Returns `true` when the user selects `affirmative`.

```go
confirmed, err := console.ConfirmAction(
    "Delete all compiled workflows?",
    "Yes, delete",
    "Cancel",
)
if err != nil || !confirmed {
    return
}
```

> **Note**: `ConfirmAction` is only available in non-WASM builds. In WASM environments the function is unavailable.

### `ShowInteractiveList(title string, items []ListItem) (string, error)`

Displays an interactive single-selection list using arrow key navigation. Returns the value of the selected `ListItem`. Returns an error if the list is empty or the prompt is cancelled. Falls back to a numbered text list in non-TTY environments.

```go
items := []console.ListItem{
    console.NewListItem("Option A", "First choice", "a"),
    console.NewListItem("Option B", "Second choice", "b"),
}
selected, err := console.ShowInteractiveList("Pick one", items)
```

### `PromptInput(title, description, placeholder string) (string, error)`

Displays a single-line text input prompt. Returns the entered string or an error.

```go
value, err := console.PromptInput("Repository name", "Enter the full owner/repo", "owner/repo")
```

> **Note**: `PromptInput` is only available in WASM builds. Use `huh` forms directly in non-WASM builds.

### `PromptSecretInput(title, description string) (string, error)`

Displays a masked password/secret input prompt. The entered value is hidden as the user types. Returns the entered secret or an error.

```go
token, err := console.PromptSecretInput("GitHub Token", "Enter your personal access token")
```

> **Note**: In non-WASM builds this requires a TTY on stderr; in WASM builds it always returns an error.

### `PromptInputWithValidation(title, description, placeholder string, validate func(string) error) (string, error)`

Like `PromptInput` but accepts a validation function that is called on the entered value before the form is submitted.

```go
value, err := console.PromptInputWithValidation(
    "Workflow name", "Must be kebab-case", "my-workflow",
    func(s string) error {
        if s == "" {
            return errors.New("name cannot be empty")
        }
        return nil
    },
)
```

> **Note**: `PromptInputWithValidation` is only available in WASM builds.

### `PromptSelect(title, description string, options []SelectOption) (string, error)`

Displays a single-select dropdown prompt. Returns the `Value` of the chosen `SelectOption` or an error.

```go
opt, err := console.PromptSelect(
    "Engine", "Choose an AI engine",
    []console.SelectOption{
        {Label: "Copilot", Value: "copilot"},
        {Label: "Claude",  Value: "claude"},
    },
)
```

> **Note**: `PromptSelect` is only available in WASM builds.

### `PromptMultiSelect(title, description string, options []SelectOption, limit int) ([]string, error)`

Displays a multi-select prompt. Returns a slice of selected `Value` strings or an error. Pass `limit ≤ 0` for no selection limit.

```go
selected, err := console.PromptMultiSelect(
    "Toolsets", "Choose toolsets to enable",
    []console.SelectOption{
        {Label: "Default", Value: "default"},
        {Label: "Issues",  Value: "issues"},
        {Label: "PRs",     Value: "pull_requests"},
    },
    0,
)
```

> **Note**: `PromptMultiSelect` is only available in WASM builds.

### `RunForm(fields []FormField) error`

Runs a multi-field interactive form defined by a slice of `FormField` values. Populates the `Value` pointer of each field and returns an error if the form is cancelled or fails.

```go
var name, token string
err := console.RunForm([]console.FormField{
    {Type: "input",    Title: "Name",  Value: &name},
    {Type: "password", Title: "Token", Value: &token},
})
```

> **Note**: `RunForm` is only available in WASM builds. Use `huh` forms directly in non-WASM builds.

## Layout Functions

These functions compose styled sections for terminal output. They are available only in WASM builds; non-WASM code should use the `RenderTitleBox`, `RenderInfoSection`, and `RenderComposedSections` functions instead.

### `LayoutTitleBox(title string, width int) string`

Returns a title box string of at least `width` characters containing `title`.

```go
header := console.LayoutTitleBox("Audit Report", 60)
fmt.Fprintln(os.Stderr, header)
```

### `LayoutInfoSection(label, value string) string`

Returns a formatted `label: value` line with consistent indentation.

```go
line := console.LayoutInfoSection("Repository", "github/gh-aw")
fmt.Fprintln(os.Stderr, line)
```

### `LayoutEmphasisBox(content string, color any) string`

Returns `content` wrapped in an emphasis box. The `color` parameter is reserved for future use (for example, to pass a `lipgloss.Color` value) and is currently ignored by the WASM implementation.

```go
box := console.LayoutEmphasisBox("Important notice", nil)
fmt.Fprintln(os.Stderr, box)
```

### `LayoutJoinVertical(sections ...string) string`

Joins multiple rendered section strings with newlines.

```go
output := console.LayoutJoinVertical(header, body, footer)
fmt.Fprintln(os.Stderr, output)
```

## Terminal Control Functions

These functions emit ANSI control sequences to manage the terminal display. They are no-ops when stderr is not a TTY.

### `ClearScreen()`

Clears the terminal screen and moves the cursor to the home position.

```go
console.ClearScreen()
```

### `ClearLine()`

Clears the current terminal line (moves to start of line and erases to end).

```go
console.ClearLine()
fmt.Fprint(os.Stderr, "Updated status...")
```

### `ShowWelcomeBanner(description string)`

Clears the screen and displays the `gh aw` welcome banner followed by `description`. Use at the start of interactive commands for a consistent onboarding experience.

```go
console.ShowWelcomeBanner("Let's create your first agentic workflow.")
```

## Verbose Logging

### `LogVerbose(verbose bool, message string)`

Writes `message` as a `FormatVerboseMessage` to `os.Stderr` when `verbose` is `true`. This is a convenience helper that avoids repetitive `if verbose` guards throughout the codebase.

```go
console.LogVerbose(cfg.Verbose, "Loaded 12 workflow files")
```

## Utility Functions

### `FormatFileSize(size int64) string`

Formats a byte count as a human-readable string with appropriate unit suffix.

```go
console.FormatFileSize(0)              // "0 B"
console.FormatFileSize(1500)           // "1.5 KB"
console.FormatFileSize(2_100_000)      // "2.0 MB"
```

### `FormatNumber(n int) string`

Formats a large integer as a compact human-readable string using SI suffixes (`k`, `M`, `B`).

```go
console.FormatNumber(0)          // "0"
console.FormatNumber(999)        // "999"
console.FormatNumber(1500)       // "1.50k"
console.FormatNumber(1_200_000)  // "1.20M"
```

### `ToRelativePath(path string) string`

Converts an absolute path to a path relative to the current working directory. If the relative path would require traversing parent directories (`..`), the original absolute path is returned unchanged.

```go
// When cwd is /home/user/projects:
console.ToRelativePath("/home/user/projects/workflow.md")  // "workflow.md"
console.ToRelativePath("/etc/hosts")                       // "/etc/hosts"
```

### `FormatErrorWithSuggestions(message string, suggestions []string) string`

Formats an error message followed by a bulleted list of actionable suggestions. Returns an empty suggestions block when `suggestions` is nil or empty.

```go
msg := console.FormatErrorWithSuggestions(
    "Unknown engine 'myengine'",
    []string{
        "Valid engines are: copilot, claude, codex",
        "Check your workflow frontmatter",
    },
)
fmt.Fprint(os.Stderr, msg)
```

### `FormatBanner() string`

Returns the `gh aw` ASCII art banner as a styled string.

### `PrintBanner()`

Prints the banner to `os.Stderr`.

## Accessibility

### `IsAccessibleMode() bool`

Returns `true` when the terminal is in accessibility mode based on environment variables:
- `ACCESSIBLE` is set (any value)
- `TERM` is `"dumb"`
- `NO_COLOR` is set (any value)

When accessibility mode is active:
- Spinner animations are disabled.
- The `huh` confirmation dialog uses accessible (plain-text) mode.
- All `Format*` functions still work normally but rendered output may differ if called with lipgloss styles.

```go
if console.IsAccessibleMode() {
    // Use simpler, non-animated output
}
```

---

*This specification is automatically maintained by the [spec-extractor](../../.github/workflows/spec-extractor.md) workflow.*
