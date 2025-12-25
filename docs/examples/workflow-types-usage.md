# WorkflowStep and WorkflowJob Types Usage

This document demonstrates how to use the `WorkflowStep` and `WorkflowJob` types for type-safe workflow processing.

## WorkflowStep Type

The `WorkflowStep` type provides compile-time type safety for GitHub Actions workflow steps.

### Creating a WorkflowStep

```go
step := &WorkflowStep{
    Name:  "Checkout code",
    Uses:  "actions/checkout@v4",
    With:  map[string]any{"fetch-depth": "0"},
}
```

### Converting Between Types

```go
// Convert WorkflowStep to map for YAML generation
stepMap := step.ToMap()

// Convert map back to WorkflowStep
step, err := MapToStep(stepMap)
if err != nil {
    log.Fatal(err)
}

// Convert slice of WorkflowSteps to []any for compatibility
stepsAny := StepsToAny(steps)

// Convert []any back to []WorkflowStep
steps, err := StepsFromAny(stepsAny)
if err != nil {
    log.Fatal(err)
}
```

### Type-Safe Action Pinning

```go
steps := []WorkflowStep{
    {Name: "Checkout", Uses: "actions/checkout@v4"},
    {Name: "Setup Node", Uses: "actions/setup-node@v4"},
}

// Apply action pins with type safety
pinnedSteps := ApplyActionPinsToWorkflowSteps(steps, workflowData)
```

## WorkflowJob Type

The `WorkflowJob` type provides type safety for job configurations.

### Creating a WorkflowJob

```go
job := &WorkflowJob{
    Name:   "Build",
    RunsOn: "ubuntu-latest",
    Needs:  []string{"test"},
    Steps: []WorkflowStep{
        {Name: "Checkout", Uses: "actions/checkout@v4"},
        {Name: "Build", Run: "npm run build"},
    },
    Permissions: map[string]string{
        "contents": "read",
    },
}
```

### Converting Between Types

```go
// Convert WorkflowJob to map
jobMap := job.ToMap()

// Convert map back to WorkflowJob
job, err := MapToJob(jobMap)
if err != nil {
    log.Fatal(err)
}
```

## Runtime Detection with Type Safety

```go
steps := []WorkflowStep{
    {Name: "Install", Run: "npm install"},
    {Name: "Test", Run: "npm test"},
}

requirements := make(map[string]*RuntimeRequirement)
detectFromWorkflowSteps(steps, requirements)

// requirements now contains detected runtimes (e.g., "node")
```

## Benefits

1. **Compile-time Type Safety**: Catch errors at compile time instead of runtime
2. **Better IDE Support**: Autocomplete and type hints in modern IDEs
3. **Clearer APIs**: Function signatures are self-documenting
4. **Easier Refactoring**: Type-safe refactoring reduces errors
5. **Backward Compatibility**: Old functions still work but are marked deprecated

## Migration Guide

### Before (using []any)

```go
func processSteps(steps []any) []any {
    result := make([]any, len(steps))
    for i, step := range steps {
        if stepMap, ok := step.(map[string]any); ok {
            // Process stepMap
            result[i] = processStepMap(stepMap)
        }
    }
    return result
}
```

### After (using []WorkflowStep)

```go
func processSteps(steps []WorkflowStep) []WorkflowStep {
    result := make([]WorkflowStep, len(steps))
    for i, step := range steps {
        // Process step with type safety
        result[i] = processStep(step)
    }
    return result
}
```

## Related Types

- `WorkflowStep`: Represents a single workflow step
- `WorkflowJob`: Represents a complete job with steps
- `Job`: Internal compiler type for job management (different from WorkflowJob)
- `WorkflowData`: Contains parsed workflow configuration

## See Also

- [pkg/workflow/step_types.go](../../pkg/workflow/step_types.go) - Type definitions
- [pkg/workflow/step_types_test.go](../../pkg/workflow/step_types_test.go) - Test examples
- [pkg/workflow/action_pins.go](../../pkg/workflow/action_pins.go) - Action pinning functions
- [pkg/workflow/runtime_setup.go](../../pkg/workflow/runtime_setup.go) - Runtime detection
