# CodingAgentEngine Interface Decomposition

## Overview

The `CodingAgentEngine` interface has been decomposed from a single large interface with 16 methods into 5 focused interfaces following the Interface Segregation Principle (ISP) and Single Responsibility Principle (SRP).

## New Interface Structure

### 1. EngineMetadata (3 methods)
Core metadata about an agentic engine:
- `GetID() string` - Unique engine identifier
- `GetDisplayName() string` - Human-readable engine name
- `GetDescription() string` - Engine capabilities description

**When to use**: When you only need to identify or describe an engine

### 2. EngineCapabilities (7 methods)
Feature capabilities and support flags:
- `IsExperimental() bool`
- `SupportsToolsAllowlist() bool`
- `SupportsHTTPTransport() bool`
- `SupportsMaxTurns() bool`
- `SupportsWebFetch() bool`
- `SupportsWebSearch() bool`
- `SupportsFirewall() bool`

**When to use**: When you need to check if an engine supports specific features

### 3. EngineInstaller (2 methods)
Installation workflow generation:
- `GetInstallationSteps(workflowData *WorkflowData) []GitHubActionStep`
- `GetRequiredSecretNames(workflowData *WorkflowData) []string`

**When to use**: When generating GitHub Actions installation steps

### 4. EngineExecutor (4 methods)
Runtime execution and configuration:
- `GetExecutionSteps(workflowData *WorkflowData, logFile string) []GitHubActionStep`
- `GetDeclaredOutputFiles() []string`
- `RenderMCPConfig(yaml *strings.Builder, tools map[string]any, mcpTools []string, workflowData *WorkflowData)`
- `GetDefaultDetectionModel() string`

**When to use**: When generating execution steps or configuring the engine at runtime

### 5. EngineLogParser (3 methods)
Log parsing and metrics extraction:
- `ParseLogMetrics(logContent string, verbose bool) LogMetrics`
- `GetLogParserScriptId() string`
- `GetLogFileForParsing() string`

**When to use**: When parsing logs or extracting metrics from engine output

### 6. CodingAgentEngine (Composite)
Combines all focused interfaces for backward compatibility:
```go
type CodingAgentEngine interface {
    EngineMetadata
    EngineCapabilities
    EngineInstaller
    EngineExecutor
    EngineLogParser
}
```

**When to use**: When you need multiple interfaces or for backward compatibility with existing code

## Benefits

1. **Reduced Implementation Burden**: New engines can implement interfaces incrementally
2. **Easier Testing**: Mock only the interfaces needed for specific tests
3. **Better Separation of Concerns**: Code depends only on what it needs
4. **Clearer Intent**: Interface names clearly describe their purpose
5. **Type Safety**: Compile-time assertions ensure all implementations are complete

## Implementation Examples

### Example 1: Using EngineCapabilities for Feature Detection

```go
// Before: Required full CodingAgentEngine interface
func validateHTTPTransport(engine CodingAgentEngine) error {
    if !engine.SupportsHTTPTransport() {
        return fmt.Errorf("engine does not support HTTP transport")
    }
    return nil
}

// After: Only needs EngineCapabilities
func validateHTTPTransport(engine EngineCapabilities) error {
    if !engine.SupportsHTTPTransport() {
        return fmt.Errorf("engine does not support HTTP transport")
    }
    return nil
}
```

### Example 2: Using EngineExecutor for Output Collection

```go
// Before: Required full CodingAgentEngine interface
func collectOutputs(engine CodingAgentEngine) []string {
    return engine.GetDeclaredOutputFiles()
}

// After: Only needs EngineExecutor
func collectOutputs(engine EngineExecutor) []string {
    return engine.GetDeclaredOutputFiles()
}
```

### Example 3: Using Composite When Multiple Interfaces Needed

```go
// Still uses composite interface when multiple concerns are needed
func generateMCPSetup(engine CodingAgentEngine, workflowData *WorkflowData) {
    // Needs both EngineMetadata (GetID) and EngineExecutor (RenderMCPConfig)
    engineID := engine.GetID()
    engine.RenderMCPConfig(yaml, tools, mcpTools, workflowData)
}
```

## Type Assertions for Multiple Interfaces

When a function uses a focused interface but needs to access methods from another interface, use type assertions:

```go
func checkFirewallSupport(engine EngineCapabilities) error {
    // Get metadata for error messages
    engineMeta := engine.(EngineMetadata)
    
    if !engine.SupportsFirewall() {
        return fmt.Errorf("engine '%s' does not support firewall", engineMeta.GetID())
    }
    return nil
}
```

## Compile-Time Assertions

All engine implementations have compile-time assertions to ensure they implement all interfaces:

```go
// In agentic_engine_interface_test.go
var _ EngineMetadata = (*CopilotEngine)(nil)
var _ EngineCapabilities = (*CopilotEngine)(nil)
var _ EngineInstaller = (*CopilotEngine)(nil)
var _ EngineExecutor = (*CopilotEngine)(nil)
var _ EngineLogParser = (*CopilotEngine)(nil)
var _ CodingAgentEngine = (*CopilotEngine)(nil)
```

## Migration Guide

When adding new code:

1. **Identify the concern**: Metadata, capabilities, installation, execution, or logging?
2. **Use the focused interface**: Only depend on what you need
3. **Fallback to composite**: Use `CodingAgentEngine` if you need multiple concerns
4. **Add tests**: Mock only the focused interface needed

## Backward Compatibility

- All existing engine implementations continue to work without changes
- The composite `CodingAgentEngine` interface provides full backward compatibility
- Functions using the composite interface do not need to be updated immediately
- New code should prefer focused interfaces for better separation of concerns

## Files Modified

- `pkg/workflow/agentic_engine.go` - Interface definitions
- `pkg/workflow/agentic_engine_interface_test.go` - Compile-time assertions
- `pkg/workflow/engine_firewall_support.go` - Uses EngineCapabilities
- `pkg/workflow/agent_validation.go` - Uses EngineCapabilities
- `pkg/workflow/fetch.go` - Uses EngineCapabilities
- `pkg/workflow/engine_output.go` - Uses EngineExecutor
- `pkg/workflow/compiler_yaml_ai_execution.go` - Uses EngineExecutor and EngineLogParser
- `pkg/workflow/compiler_yaml_helpers.go` - Uses EngineMetadata

## Related Issues

- Addresses issue #10806: Interface Bloat & Function Signature Complexity
- Reduces implementation burden from 16 methods to focused subsets
- Improves testability by allowing focused mocking
