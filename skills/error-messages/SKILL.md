---
name: error-messages
description: Error Message Style Guide for Validation Errors
---


# Error Message Style Guide

This guide establishes the standard format for validation error messages in the gh-aw codebase. All validation errors should be clear, actionable, and include examples.

## Error Message Template

```
[what's wrong]. [what's expected]. [example of correct usage]
```

Each error message should answer three questions:
1. **What's wrong?** - Clearly state the validation error
2. **What's expected?** - Explain the valid format or values
3. **How to fix it?** - Provide a concrete example of correct usage

## Good Examples

These examples follow the template and provide actionable guidance:

### Time Delta Validation (from time_delta.go)
```go
return nil, fmt.Errorf("invalid time delta format: +%s. Expected format like +25h, +3d, +1w, +1mo, +1d12h30m", deltaStr)
```
✅ **Why it's good:**
- Clearly identifies the invalid input
- Lists multiple valid format examples
- Shows combined formats (+1d12h30m)

### Type Validation with Example
```go
return "", fmt.Errorf("manual-approval value must be a string, got %T. Example: manual-approval: \"production\"", val)
```
✅ **Why it's good:**
- Shows actual type received (%T)
- Provides concrete YAML example
- Uses proper YAML syntax with quotes

### Enum Validation with Options
```go
return fmt.Errorf("invalid engine: %s. Valid engines are: copilot, claude, codex, custom. Example: engine: copilot", engineID)
```
✅ **Why it's good:**
- Lists all valid options
- Provides simplest example
- Uses consistent formatting

### MCP Configuration
```go
return fmt.Errorf("tool '%s' mcp configuration must specify either 'command' or 'container'. Example:\ntools:\n  %s:\n    command: \"npx @my/tool\"", toolName, toolName)
```
✅ **Why it's good:**
- Explains mutual exclusivity
- Shows realistic tool name
- Formats multi-line YAML example

## Bad Examples

These examples lack clarity or actionable guidance:

### Too Vague
```go
return fmt.Errorf("invalid format")
```
❌ **Problems:**
- Doesn't specify what format is invalid
- Doesn't explain expected format
- No example provided

### Missing Example
```go
return fmt.Errorf("manual-approval value must be a string")
```
❌ **Problems:**
- States requirement but no example
- User doesn't know proper YAML syntax
- Could be clearer about type received

### Incomplete Information
```go
return fmt.Errorf("invalid engine: %s", engineID)
```
❌ **Problems:**
- Doesn't list valid options
- No guidance on fixing the error
- User must search documentation

## When to Include Examples

Always include examples for:

1. **Format/Syntax Errors** - Show the correct syntax
   ```go
   fmt.Errorf("invalid date format. Expected: YYYY-MM-DD HH:MM:SS. Example: 2024-01-15 14:30:00")
   ```

2. **Enum/Choice Fields** - List all valid options
   ```go
   fmt.Errorf("invalid permission level: %s. Valid levels: read, write, none. Example: permissions:\n  contents: read", level)
   ```

3. **Type Mismatches** - Show expected type and example
   ```go
   fmt.Errorf("timeout-minutes must be an integer, got %T. Example: timeout-minutes: 10", value)
   ```

4. **Complex Configurations** - Provide complete valid example
   ```go
   fmt.Errorf("invalid MCP server config. Example:\nmcp-servers:\n  my-server:\n    command: \"node\"\n    args: [\"server.js\"]")
   ```

## When Examples May Be Optional

Examples can be omitted when:

1. **Error is from wrapped error** - When wrapping another error with context
   ```go
   return fmt.Errorf("failed to parse configuration: %w", err)
   ```

2. **Error is self-explanatory with clear context**
   ```go
   return fmt.Errorf("duplicate unit '%s' in time delta: +%s", unit, deltaStr)
   ```

3. **Error points to specific documentation**
   ```go
   return fmt.Errorf("unsupported feature. See https://docs.example.com/features")
   ```

## Formatting Guidelines

### Use Type Verbs for Dynamic Content
- `%s` - strings
- `%d` - integers  
- `%T` - type of value
- `%v` - general value
- `%w` - wrapped errors

### Multi-line Examples
For YAML configuration examples spanning multiple lines:
```go
fmt.Errorf("invalid config. Example:\ntools:\n  github:\n    mode: \"remote\"")
```

### Quoting in Examples
Use proper YAML syntax in examples:
```go
// Good - shows quotes when needed
fmt.Errorf("Example: name: \"my-workflow\"")

// Good - shows no quotes for simple values
fmt.Errorf("Example: timeout-minutes: 10")
```

### Consistent Terminology
Use the same field names as in YAML:
```go
// Good - matches YAML field name
fmt.Errorf("timeout-minutes must be positive")

// Bad - uses different name
fmt.Errorf("timeout must be positive")
```

## Error Message Testing

All improved error messages should have corresponding tests:

```go
func TestErrorMessageQuality(t *testing.T) {
    err := validateSomething(invalidInput)
    require.Error(t, err)
    
    // Error should explain what's wrong
    assert.Contains(t, err.Error(), "invalid")
    
    // Error should include expected format or values
    assert.Contains(t, err.Error(), "Expected")
    
    // Error should include example
    assert.Contains(t, err.Error(), "Example:")
}
```

## Migration Strategy

When improving existing error messages:

1. **Identify the error** - Find validation error that lacks clarity
2. **Analyze context** - Understand what's being validated
3. **Apply template** - Add what's wrong + expected + example
4. **Add tests** - Verify error message content
5. **Update comments** - Document the validation logic

## Examples by Category

### Format Validation
```go
// Time deltas
fmt.Errorf("invalid time delta format: +%s. Expected format like +25h, +3d, +1w, +1mo, +1d12h30m", input)

// Dates
fmt.Errorf("invalid date format: %s. Expected: YYYY-MM-DD or relative like -1w. Example: 2024-01-15 or -7d", input)

// URLs
fmt.Errorf("invalid URL format: %s. Expected: https:// URL. Example: https://api.example.com", input)
```

### Type Validation
```go
// Boolean expected
fmt.Errorf("read-only must be a boolean, got %T. Example: read-only: true", value)

// String expected
fmt.Errorf("workflow name must be a string, got %T. Example: name: \"my-workflow\"", value)

// Object expected
fmt.Errorf("permissions must be an object, got %T. Example: permissions:\n  contents: read", value)
```

### Choice/Enum Validation
```go
// Engine selection
fmt.Errorf("invalid engine: %s. Valid engines: copilot, claude, codex, custom. Example: engine: copilot", id)

// Permission levels
fmt.Errorf("invalid permission level: %s. Valid levels: read, write, none. Example: contents: read", level)

// Tool modes
fmt.Errorf("invalid mode: %s. Valid modes: local, remote. Example: mode: \"remote\"", mode)
```

### Configuration Validation
```go
// Missing required field
fmt.Errorf("tool '%s' missing required 'command' field. Example:\ntools:\n  %s:\n    command: \"node server.js\"", name, name)

// Mutually exclusive fields
fmt.Errorf("cannot specify both 'command' and 'container'. Choose one. Example: command: \"node server.js\"")

// Invalid combination
fmt.Errorf("http MCP servers cannot use 'container' field. Example:\ntools:\n  my-http:\n    type: http\n    url: \"https://api.example.com\"")
```

## References

- **Excellent example to follow**: `pkg/workflow/time_delta.go`
- **Pattern inspiration**: Go standard library error messages
- **Testing examples**: `pkg/workflow/*_test.go`

## Tools

When writing error messages, consider:
- The user's perspective (what do they need to fix it?)
- The context (where in the workflow is the error?)
- The documentation (should we reference specific docs?)
- The complexity (is multi-line example needed?)
