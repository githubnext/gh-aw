# Trigger Structure Implementation

This document describes the strongly typed trigger structure implementation for workflow frontmatter parsing in gh-aw.

## Overview

The trigger structure provides a strongly typed representation of the "on" section in workflow frontmatter, improving type safety and maintainability compared to the previous map-based approach.

## Files

- `pkg/workflow/trigger.go` - Core trigger structure and parsing logic
- `pkg/workflow/trigger_test.go` - Unit tests for trigger parsing
- `pkg/workflow/trigger_integration_test.go` - Integration tests with real workflows

## TriggerConfig Structure

### Fields

```go
type TriggerConfig struct {
    // Raw holds the original trigger data from frontmatter for backward compatibility
    Raw map[string]any

    // Simple is set when the trigger is a simple string like "push" or "workflow_dispatch"
    Simple string

    // Events maps event names to their configurations
    Events map[string]EventConfig

    // Command holds command trigger configuration if present
    Command *CommandTriggerConfig

    // Reaction holds reaction configuration if present in the on section
    Reaction string

    // StopAfter holds the stop-after deadline if present in the on section
    StopAfter string

    // ManualApproval holds the environment name for manual approval if present
    ManualApproval string
}
```

### EventConfig Structure

```go
type EventConfig struct {
    // Raw holds the raw configuration for this event
    Raw any

    // Types specifies the activity types that trigger the event (e.g., [opened, closed])
    Types []string

    // Branches specifies branch filters
    Branches []string

    // Tags specifies tag filters
    Tags []string

    // Paths specifies path filters
    Paths []string

    // WorkflowRuns specifies workflow run filters for workflow_run events
    WorkflowRuns []string
}
```

### CommandTriggerConfig Structure

```go
type CommandTriggerConfig struct {
    // Name is the command name (e.g., "bot-name" for /bot-name)
    Name string

    // Events lists the events where the command should be active
    // nil or empty means all comment-related events
    Events []string
}
```

## Usage

### Parsing Triggers

The `ParseTrigger` function converts frontmatter "on" values into a `TriggerConfig`:

```go
onValue := frontmatter["on"]
trigger, err := ParseTrigger(onValue)
if err != nil {
    return fmt.Errorf("failed to parse trigger: %w", err)
}
```

### Supported Formats

#### Simple String Trigger

```yaml
on: push
```

Parsed as:
```go
TriggerConfig{
    Simple: "push",
    Events: map[string]EventConfig{"push": {Raw: nil}},
}
```

#### Complex Event Configuration

```yaml
on:
  pull_request:
    types: [opened, synchronize]
    branches: [main, develop]
```

Parsed as:
```go
TriggerConfig{
    Events: map[string]EventConfig{
        "pull_request": {
            Types: ["opened", "synchronize"],
            Branches: ["main", "develop"],
        },
    },
}
```

#### Multiple Events

```yaml
on:
  push:
    branches: [main]
  pull_request:
    types: [opened]
  workflow_dispatch:
```

#### Command Triggers

```yaml
on:
  command:
    name: bot
    events: [issues, issue_comment]
```

#### Special Fields

```yaml
on:
  issues:
    types: [opened]
  reaction: rocket
  stop-after: "2024-12-31 23:59:59"
```

### Helper Methods

```go
// Check if trigger includes a specific event
if trigger.HasEvent("pull_request") {
    // Handle pull_request event
}

// Check if trigger includes a command
if trigger.HasCommand() {
    commandName := trigger.GetCommandName()
    events := trigger.GetCommandEvents()
}

// Get event configuration
if event, exists := trigger.Events["pull_request"]; exists {
    types := event.Types
    branches := event.Branches
}
```

### Converting Back to YAML

```go
yamlStr, err := trigger.ToYAML()
if err != nil {
    return err
}
// yamlStr contains: '"on":\n  pull_request:\n    types: [opened]'
```

## Integration with WorkflowData

The `WorkflowData` struct now includes a `ParsedTrigger` field:

```go
type WorkflowData struct {
    // ... other fields
    On            string          // Original YAML string (backward compatibility)
    ParsedTrigger *TriggerConfig  // Structured trigger configuration
    // ... other fields
}
```

### Parsing Integration

The trigger is parsed in the `parseOnSection` function:

```go
func (c *Compiler) parseOnSection(frontmatter map[string]any, workflowData *WorkflowData, markdownPath string) error {
    if onValue, exists := frontmatter["on"]; exists {
        parsedTrigger, err := ParseTrigger(onValue)
        if err != nil {
            return fmt.Errorf("failed to parse trigger configuration: %w", err)
        }
        workflowData.ParsedTrigger = parsedTrigger
        // ... rest of the function
    }
    return nil
}
```

## Backward Compatibility

The implementation maintains full backward compatibility:

1. **On Field Preserved**: The original `On` string field remains populated with YAML-formatted trigger data
2. **Raw Data Stored**: The `TriggerConfig.Raw` field stores the original map for reference
3. **ToYAML Support**: The `ToYAML()` method can reconstruct the YAML representation

Existing code that uses `workflowData.On` continues to work unchanged. New code can use `workflowData.ParsedTrigger` for type-safe access.

## Testing

### Unit Tests

Located in `pkg/workflow/trigger_test.go`:

- `TestParseTrigger_SimpleString` - Simple string triggers
- `TestParseTrigger_ComplexEvents` - Complex event configurations
- `TestParseTrigger_CommandTrigger` - Command trigger parsing
- `TestParseTrigger_ReactionAndStopAfter` - Special fields
- `TestParseTrigger_Errors` - Error handling
- `TestTriggerConfig_HasEvent` - Helper methods
- `TestTriggerConfig_ToYAML` - YAML conversion
- `TestParseEventConfig` - Event configuration parsing
- `TestParseStringArray` - String array parsing

### Integration Tests

Located in `pkg/workflow/trigger_integration_test.go`:

- `TestTriggerParsingIntegration` - Real workflow scenarios
- `TestTriggerParsingWithRealWorkflows` - Complex workflow parsing
- `TestTriggerBackwardCompatibility` - Backward compatibility validation
- `TestTriggerToYAMLRoundTrip` - Round-trip conversion

All tests pass with 100% coverage of core functionality.

## Future Enhancements

Potential future improvements:

1. **Trigger Validation**: Add validation rules for event types and combinations
2. **Builder Pattern**: Add fluent builder for constructing triggers programmatically
3. **Event Type Constants**: Define constants for standard GitHub event types
4. **Migration Helpers**: Tools to help migrate code from On string to ParsedTrigger

## Migration Guide

To migrate existing code from using `workflowData.On` to `workflowData.ParsedTrigger`:

### Before
```go
// Parsing YAML manually
var onMap map[string]any
yaml.Unmarshal([]byte(workflowData.On), &onMap)
if _, hasPR := onMap["pull_request"]; hasPR {
    // Handle pull_request
}
```

### After
```go
// Using parsed trigger
if workflowData.ParsedTrigger.HasEvent("pull_request") {
    prEvent := workflowData.ParsedTrigger.Events["pull_request"]
    types := prEvent.Types
    branches := prEvent.Branches
}
```

## Best Practices

1. **Use ParsedTrigger for New Code**: Always prefer the typed structure over string parsing
2. **Check for Nil**: Always check if `ParsedTrigger` is nil before accessing
3. **Use Helper Methods**: Use `HasEvent()`, `HasCommand()`, etc. for cleaner code
4. **Preserve Raw Data**: When modifying triggers, preserve the `Raw` field for debugging

## Error Handling

The parser returns descriptive errors for invalid configurations:

```go
trigger, err := ParseTrigger(invalidValue)
if err != nil {
    // Error examples:
    // - "on field is required but not provided"
    // - "on field must be a string or object, got int"
    // - "invalid configuration for event 'pull_request': ..."
}
```
