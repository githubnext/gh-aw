---
name: error-messages
description: Error Message Style Guide for Validation Errors
---


# Error Message Style Guide

This guide establishes the standard format for validation error messages in the gh-aw codebase. All validation errors should be clear, actionable, and include examples.

## Error Message Template

```
[empathetic opening with emoji] [what's wrong and why it matters]. 

[context: why this rule exists]

[what's expected with specific examples]

[learning resource link if applicable]
```

Each error message should answer these questions:
1. **Acknowledge the situation** - Use an emoji and empathetic opening (🤔, 💡, ⚠️, 🔒)
2. **What's wrong?** - Clearly state the validation error
3. **Why does this matter?** - Explain the reason (security, best practices, compatibility)
4. **What's expected?** - Explain the valid format or values
5. **How to fix it?** - Provide concrete examples of correct usage
6. **Where to learn more?** - Link to relevant documentation when helpful

## Good Examples

These examples follow the empathetic template and provide educational guidance:

### Engine Validation (Empathetic)
```go
return fmt.Errorf("🤔 Hmm, we don't recognize the engine '%s'.\n\nValid options are:\n  • copilot (GitHub Copilot)\n  • claude (Anthropic Claude)\n  • codex (OpenAI Codex)\n  • custom (your own engine)\n\nExample:\n  engine: copilot\n\nNeed help choosing? See: https://githubnext.github.io/gh-aw/reference/engines/", engineID)
```
✅ **Why it's good:**
- Empathetic opening with emoji
- Lists all options with descriptions
- Provides learning resource link
- Conversational, non-blaming tone

### Security Validation (Empathetic)
```go
return fmt.Errorf("🔒 Write permissions detected.\n\nFor security, workflows use read-only permissions by default. Write permissions require the 'dangerous-permissions-write' feature flag.\n\nWhy? Write permissions can modify repository contents and settings, which needs explicit opt-in.\n\nFound write permissions:\n%s\n\nTo fix, either:\n  1. Change to read permissions:\n     permissions:\n       contents: read\n\n  2. Or enable the feature flag:\n     features:\n       dangerous-permissions-write: true\n\nLearn more: https://githubnext.github.io/gh-aw/reference/permissions/", details)
```
✅ **Why it's good:**
- Lock emoji for security context
- Explains *why* the rule exists
- Provides two fix options
- Links to documentation
- Educational and empowering

### MCP Configuration (Empathetic)
```go
return fmt.Errorf("💡 The MCP server '%s' needs a way to start.\n\nMCP servers using 'stdio' type need either a 'command' or 'container', but not both.\n\nWhy? The command tells us how to launch your MCP server.\n\nExample with command:\n  tools:\n    %s:\n      command: \"node server.js\"\n      args: [\"--port\", \"3000\"]\n\nOr with container:\n  tools:\n    %s:\n      container: \"my-registry/my-tool\"\n      version: \"latest\"\n\nLearn more: https://githubnext.github.io/gh-aw/guides/mcp-servers/", toolName, toolName, toolName)
```
✅ **Why it's good:**
- Light bulb emoji for helpful suggestion
- Explains the requirement clearly
- Shows both valid options
- Includes why the rule exists
- Links to detailed guide

## Emoji Guidelines

Choose emojis that match the context and tone:

- **🤔** - Confusion or something not recognized (invalid values, unknown options)
- **💡** - Helpful suggestion or tip (configuration needs, how to fix)
- **⚠️** - Warning or caution (potential issues, best practices)
- **🔒** - Security-related (permissions, access control, sensitive data)
- **📝** - Documentation or format issues (syntax, structure)
- **🏗️** - Build or configuration setup (missing dependencies, setup requirements)
- **🔍** - Not found or missing (files, resources, references)

Use emojis sparingly - one per error message is enough.

## Conversational Tone Guidelines

Write error messages as if you're a helpful colleague:

**DO:**
- Use "we" and "you" to be inclusive
- Acknowledge the situation empathetically
- Explain the "why" behind rules
- Offer choices when applicable
- End with a learning opportunity

**DON'T:**
- Blame the user ("you did this wrong")
- Use overly technical jargon without explanation
- Be condescending or patronizing
- Use multiple emojis in one message
- Make jokes that minimize the issue

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
