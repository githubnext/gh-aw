---
name: console-rendering
description: Instructions for using the struct tag-based console rendering system in Go
---

# Console Rendering System Usage

This skill provides instructions for using the struct tag-based console rendering system.

## Struct Tag Support

Use the `console` struct tag to control rendering behavior:

- **`header:"Name"`** - Sets the display name for fields (used in both structs and tables)
- **`title:"Section Title"`** - Sets the title for nested structs, slices, or maps
- **`format:"type"`** - Sets the formatting type for the field value
  - `format:number` - Formats integers as human-readable numbers (e.g., "1k", "1.2M")
  - `format:cost` - Formats floats as currency with $ prefix (e.g., "$1.234")
- **`omitempty`** - Skips the field if it has a zero value
- **`"-"`** - Always skips the field

## Example Usage

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

// Simple rendering
fmt.Print(console.RenderStruct(data))

// Output:
//   Run ID  : 12345
//   Workflow: test-workflow
//   Status  : completed
//   Duration: 5m30s
```

## Format Tag Examples

### Number Formatting

```go
type Metrics struct {
    TokenUsage int `console:"header:Token Usage,format:number"`
    Errors     int `console:"header:Errors"`
}

data := Metrics{
    TokenUsage: 250000,
    Errors:     5,
}

// Renders as:
//   Token Usage: 250k
//   Errors     : 5
```

### Cost Formatting

```go
type Billing struct {
    Cost float64 `console:"header:Estimated Cost,format:cost"`
}

data := Billing{
    Cost: 1.234,
}

// Renders as:
//   Estimated Cost: $1.234
```

## Rendering Behavior

### Structs
Structs are rendered as key-value pairs with proper alignment.

### Slices
Slices of structs are automatically rendered as tables:

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

### Maps
Maps are rendered as markdown-style headers with key-value pairs.

### Special Type Handling

#### time.Time
`time.Time` fields are automatically formatted as `"2006-01-02 15:04:05"`. Zero time values are considered empty when used with `omitempty`.

#### Unexported Fields
The rendering system safely handles unexported struct fields by checking `CanInterface()` before attempting to access field values.
