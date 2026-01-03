# Console Rendering Package

The `console` package provides utilities for rendering Go structs and data structures to formatted console output, as well as progress bar components for long-running operations.

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

```go
import "github.com/githubnext/gh-aw/pkg/console"

// Create a progress bar for 1GB total
totalBytes := int64(1024 * 1024 * 1024)
bar := console.NewProgressBar(totalBytes)

// Update progress (returns formatted string)
output := bar.Update(currentBytes)
fmt.Fprintf(os.Stderr, "\r%s", output)
```

### Output Examples

**TTY Mode** (with color support):
```
████████████████████░░░░░░░░░░░░░░░░░  50%
```
*(Displays with gradient from purple to cyan)*

**Non-TTY Mode** (text fallback):
```
50% (512.0MB/1.00GB)
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
    "Repository": "githubnext/gh-aw",
    "Author":     "test-user",
}

fmt.Print(console.RenderStruct(data))
```

Renders as:

```
  Repository: githubnext/gh-aw
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
