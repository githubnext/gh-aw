---
description: Comprehensive guide for implementing custom agentic engines in gh-aw
applyTo: "pkg/workflow/*engine*.go"
disable-model-invocation: true
---

# Custom Agentic Engine Implementation Guide

This document provides a comprehensive guide for implementing custom agentic engines in GitHub Agentic Workflows (gh-aw). It covers architecture patterns, common refactoring opportunities, and step-by-step implementation instructions.

## Table of Contents

1. [Architecture Overview](#architecture-overview)
2. [Engine Interface Design](#engine-interface-design)
3. [Common Code Analysis & Refactoring Opportunities](#common-code-analysis--refactoring-opportunities)
4. [Implementation Guide](#implementation-guide)
5. [Testing Strategy](#testing-strategy)
6. [Integration Checklist](#integration-checklist)

---

## Architecture Overview

### Interface Segregation Principle

The agentic engine architecture follows the **Interface Segregation Principle (ISP)** to avoid forcing implementations to depend on methods they don't use. The system uses **interface composition** to provide flexibility while maintaining backward compatibility.

### Interface Hierarchy

```
Engine (core identity - required by all)
├── GetID()
├── GetDisplayName()
├── GetDescription()
└── IsExperimental()

CapabilityProvider (feature detection - optional)
├── SupportsToolsAllowlist()
├── SupportsHTTPTransport()
├── SupportsMaxTurns()
├── SupportsWebFetch()
├── SupportsWebSearch()
├── SupportsFirewall()
├── SupportsPlugins()
└── SupportsLLMGateway()

WorkflowExecutor (compilation - required)
├── GetDeclaredOutputFiles()
├── GetInstallationSteps()
└── GetExecutionSteps()

MCPConfigProvider (MCP servers - optional)
└── RenderMCPConfig()

LogParser (log analysis - optional)
├── ParseLogMetrics()
├── GetLogParserScriptId()
└── GetLogFileForParsing()

SecurityProvider (security features - optional)
├── GetDefaultDetectionModel()
└── GetRequiredSecretNames()

CodingAgentEngine (composite - backward compatibility)
└── Composes all above interfaces
```

### Key Architectural Patterns

1. **BaseEngine Embedding**: All engines embed `BaseEngine` which provides default implementations
2. **Focused Interfaces**: Each interface has a single responsibility
3. **Optional Capabilities**: Engines override only the methods they need
4. **Backward Compatibility**: `CodingAgentEngine` composite interface maintains compatibility

---

## Engine Interface Design

### Core Engine Identity

Every engine must implement the `Engine` interface:

```go
type Engine interface {
    GetID() string            // Unique identifier (e.g., "copilot", "claude", "codex")
    GetDisplayName() string   // Human-readable name (e.g., "GitHub Copilot CLI")
    GetDescription() string   // Capability description
    IsExperimental() bool     // Experimental status flag
}
```

### Capability Detection

Engines can implement `CapabilityProvider` to indicate feature support:

```go
type CapabilityProvider interface {
    SupportsToolsAllowlist() bool    // MCP tool allow-listing
    SupportsHTTPTransport() bool     // HTTP transport for MCP servers
    SupportsMaxTurns() bool          // Max-turns feature
    SupportsWebFetch() bool          // Built-in web-fetch tool
    SupportsWebSearch() bool         // Built-in web-search tool
    SupportsFirewall() bool          // Network firewalling/sandboxing
    SupportsPlugins() bool           // Plugin installation
    SupportsLLMGateway() int         // LLM gateway port (or -1 if not supported)
}
```

### Workflow Compilation

All engines must implement `WorkflowExecutor`:

```go
type WorkflowExecutor interface {
    GetDeclaredOutputFiles() []string                                    // Output files to upload
    GetInstallationSteps(workflowData *WorkflowData) []GitHubActionStep  // Installation steps
    GetExecutionSteps(workflowData *WorkflowData, logFile string) []GitHubActionStep // Execution steps
}
```

### Optional Interfaces

Engines can optionally implement:

- **MCPConfigProvider**: For MCP server configuration
- **LogParser**: For custom log parsing and metrics extraction
- **SecurityProvider**: For security features and secret management

---

## Common Code Analysis & Refactoring Opportunities

### Current Engine Implementations (LOC)

```
claude_engine.go:    474 lines
codex_engine.go:     523 lines
copilot_engine.go:   170 lines
custom_engine.go:    373 lines
Total:              1540 lines
```

### Existing Shared Helpers

The codebase already has well-organized helper modules:

#### 1. **engine_helpers.go** (501 lines)
Common installation and utility functions:
- `GetBaseInstallationSteps()` - Secret validation + npm installation
- `BuildStandardNpmEngineInstallSteps()` - Standard npm engine setup
- `InjectCustomEngineSteps()` - Custom step injection
- `FormatStepWithCommandAndEnv()` - Step formatting
- `FilterEnvForSecrets()` - Security-focused env filtering
- `GetHostedToolcachePathSetup()` - Runtime path setup
- `GetNpmBinPathSetup()` - NPM binary path setup
- `ResolveAgentFilePath()` - Agent file path resolution
- `ExtractAgentIdentifier()` - Agent identifier extraction

#### 2. **awf_helpers.go** (248 lines)
AWF (firewall/sandbox) integration:
- `BuildAWFCommand()` - Complete AWF command construction
- `BuildAWFArgs()` - AWF argument generation
- `GetAWFCommandPrefix()` - AWF command determination
- `WrapCommandInShell()` - Shell wrapper for AWF

#### 3. **MCP Configuration** (Multiple files)
- `mcp_config_builtin.go` - Built-in MCP server configs
- `mcp_config_custom.go` - Custom MCP server handling
- `mcp_config_playwright_renderer.go` - Playwright MCP rendering
- `mcp_renderer.go` - Unified MCP rendering framework

### Identified Refactoring Opportunities

#### Opportunity 1: Consolidate MCP Rendering Patterns

**Current State**: Each engine has its own `RenderMCPConfig()` method with similar structure:

```go
// Claude, Codex, and Custom engines all follow this pattern
func (e *Engine) RenderMCPConfig(yaml *strings.Builder, tools map[string]any, mcpTools []string, workflowData *WorkflowData) {
    createRenderer := func(isLast bool) *MCPConfigRendererUnified {
        return NewMCPConfigRenderer(MCPRendererOptions{...})
    }
    
    RenderJSONMCPConfig(yaml, tools, mcpTools, workflowData, JSONMCPConfigOptions{
        ConfigPath: "/path/to/config",
        GatewayConfig: buildMCPGatewayConfig(workflowData),
        Renderers: MCPToolRenderers{...},
    })
}
```

**Refactoring Recommendation**:
- Extract common MCP renderer factory pattern
- Create `BuildStandardJSONMCPConfig()` helper
- Reduce duplication across engines

#### Opportunity 2: Standardize Installation Step Generation

**Current State**: Engines use different patterns for installation:
- Copilot: Uses installer script approach
- Claude/Codex: Use npm-based installation
- All validate secrets differently

**Refactoring Recommendation**:
- Already well-abstracted via `GetBaseInstallationSteps()`
- Consider adding `BuildInstallerScriptSteps()` for non-npm engines
- Standardize AWF installation integration

#### Opportunity 3: Unify Log Parsing Infrastructure

**Current State**: Each engine has dedicated log parsing files:
- `copilot_logs.go` (comprehensive)
- `claude_logs.go` (structured JSON parsing)
- `codex_logs.go` (regex-based parsing)

**Refactoring Recommendation**:
- Extract common log parsing patterns (token counting, turn tracking)
- Create `BaseLogParser` with common utilities
- Keep engine-specific parsing in separate files

#### Opportunity 4: Simplify Environment Variable Management

**Current State**: Each engine manually builds environment maps with:
- Secret references
- Safe outputs configuration
- Custom environment variables
- Model configuration

**Refactoring Recommendation**:
- Create `BuildBaseEngineEnv()` helper for common env vars
- Standardize secret filtering via `FilterEnvForSecrets()` (already exists)
- Extract model environment variable logic

---

## Implementation Guide

### Step 1: Define Engine Structure

```go
package workflow

import (
    "github.com/github/gh-aw/pkg/logger"
)

var myEngineLog = logger.New("workflow:my_engine")

// MyEngine represents the My Custom agentic engine
type MyEngine struct {
    BaseEngine
}

func NewMyEngine() *MyEngine {
    return &MyEngine{
        BaseEngine: BaseEngine{
            id:                     "my-engine",
            displayName:            "My Custom Engine",
            description:            "Custom AI engine with XYZ capabilities",
            experimental:           false, // Set to true if experimental
            supportsToolsAllowlist: true,
            supportsHTTPTransport:  true,
            supportsMaxTurns:       false,
            supportsWebFetch:       false,
            supportsWebSearch:      false,
            supportsFirewall:       true,
            supportsPlugins:        false,
            supportsLLMGateway:     false, // Override SupportsLLMGateway() if true
        },
    }
}
```

### Step 2: Implement Required Secrets

```go
func (e *MyEngine) GetRequiredSecretNames(workflowData *WorkflowData) []string {
    secrets := []string{"MY_ENGINE_API_KEY"}
    
    // Add MCP gateway API key if MCP servers are present
    if HasMCPServers(workflowData) {
        secrets = append(secrets, "MCP_GATEWAY_API_KEY")
    }
    
    // Add safe-inputs secrets if enabled
    if IsSafeInputsEnabled(workflowData.SafeInputs, workflowData) {
        safeInputsSecrets := collectSafeInputsSecrets(workflowData.SafeInputs)
        for varName := range safeInputsSecrets {
            secrets = append(secrets, varName)
        }
    }
    
    return secrets
}
```

### Step 3: Implement Installation Steps

```go
func (e *MyEngine) GetInstallationSteps(workflowData *WorkflowData) []GitHubActionStep {
    myEngineLog.Printf("Generating installation steps: workflow=%s", workflowData.Name)
    
    // Skip installation if custom command is specified
    if workflowData.EngineConfig != nil && workflowData.EngineConfig.Command != "" {
        myEngineLog.Printf("Skipping installation: custom command specified (%s)", workflowData.EngineConfig.Command)
        return []GitHubActionStep{}
    }
    
    // Use base installation steps (secret validation + npm install)
    steps := GetBaseInstallationSteps(EngineInstallConfig{
        Secrets:         []string{"MY_ENGINE_API_KEY"},
        DocsURL:         "https://example.com/docs/my-engine",
        NpmPackage:      "@company/my-engine",
        Version:         "1.0.0", // Use constant: string(constants.DefaultMyEngineVersion)
        Name:            "My Engine",
        CliName:         "my-engine",
        InstallStepName: "Install My Engine CLI",
    }, workflowData)
    
    // Add AWF installation if firewall is enabled
    if isFirewallEnabled(workflowData) {
        firewallConfig := getFirewallConfig(workflowData)
        agentConfig := getAgentConfig(workflowData)
        var awfVersion string
        if firewallConfig != nil {
            awfVersion = firewallConfig.Version
        }
        
        awfInstall := generateAWFInstallationStep(awfVersion, agentConfig)
        if len(awfInstall) > 0 {
            steps = append(steps, awfInstall)
        }
    }
    
    return steps
}
```

### Step 4: Implement Execution Steps

```go
func (e *MyEngine) GetExecutionSteps(workflowData *WorkflowData, logFile string) []GitHubActionStep {
    modelConfigured := workflowData.EngineConfig != nil && workflowData.EngineConfig.Model != ""
    firewallEnabled := isFirewallEnabled(workflowData)
    
    myEngineLog.Printf("Building execution steps: workflow=%s, model=%s, firewall=%v",
        workflowData.Name, getModel(workflowData), firewallEnabled)
    
    // Handle custom steps if they exist in engine config
    steps := InjectCustomEngineSteps(workflowData, e.convertStepToYAML)
    
    // Build engine command arguments
    var engineArgs []string
    
    // Add model if specified
    if modelConfigured {
        engineArgs = append(engineArgs, "--model", workflowData.EngineConfig.Model)
    }
    
    // Add MCP config if servers are present
    if HasMCPServers(workflowData) {
        engineArgs = append(engineArgs, "--mcp-config", "/tmp/gh-aw/mcp-config/mcp-servers.json")
    }
    
    // Build the command
    commandName := "my-engine"
    if workflowData.EngineConfig != nil && workflowData.EngineConfig.Command != "" {
        commandName = workflowData.EngineConfig.Command
    }
    
    engineCommand := fmt.Sprintf("%s %s \"$(cat /tmp/gh-aw/aw-prompts/prompt.txt)\"",
        commandName, shellJoinArgs(engineArgs))
    
    // Build the full command with AWF wrapping if enabled
    var command string
    if firewallEnabled {
        allowedDomains := GetMyEngineAllowedDomainsWithToolsAndRuntimes(
            workflowData.NetworkPermissions, 
            workflowData.Tools, 
            workflowData.Runtimes,
        )
        
        npmPathSetup := GetNpmBinPathSetup()
        engineCommandWithPath := fmt.Sprintf("%s && %s", npmPathSetup, engineCommand)
        
        command = BuildAWFCommand(AWFCommandConfig{
            EngineName:     "my-engine",
            EngineCommand:  engineCommandWithPath,
            LogFile:        logFile,
            WorkflowData:   workflowData,
            UsesTTY:        false,
            UsesAPIProxy:   false,
            AllowedDomains: allowedDomains,
        })
    } else {
        command = fmt.Sprintf(`set -o pipefail
%s 2>&1 | tee %s`, engineCommand, logFile)
    }
    
    // Build environment variables
    env := map[string]string{
        "MY_ENGINE_API_KEY":  "${{ secrets.MY_ENGINE_API_KEY }}",
        "GH_AW_PROMPT":       "/tmp/gh-aw/aw-prompts/prompt.txt",
        "GITHUB_WORKSPACE":   "${{ github.workspace }}",
    }
    
    // Add MCP config env var if needed
    if HasMCPServers(workflowData) {
        env["GH_AW_MCP_CONFIG"] = "/tmp/gh-aw/mcp-config/mcp-servers.json"
    }
    
    // Add safe outputs env
    applySafeOutputEnvToMap(env, workflowData)
    
    // Add model env var if not explicitly configured
    if !modelConfigured {
        isDetectionJob := workflowData.SafeOutputs == nil
        if isDetectionJob {
            env["GH_AW_MODEL_DETECTION_MY_ENGINE"] = "${{ vars.GH_AW_MODEL_DETECTION_MY_ENGINE || '' }}"
        } else {
            env["GH_AW_MODEL_AGENT_MY_ENGINE"] = "${{ vars.GH_AW_MODEL_AGENT_MY_ENGINE || '' }}"
        }
    }
    
    // Generate the execution step
    stepLines := []string{
        "      - name: Run My Engine",
        "        id: agentic_execution",
    }
    
    // Filter environment variables for security
    allowedSecrets := e.GetRequiredSecretNames(workflowData)
    filteredEnv := FilterEnvForSecrets(env, allowedSecrets)
    
    // Format step with command and env
    stepLines = FormatStepWithCommandAndEnv(stepLines, command, filteredEnv)
    
    steps = append(steps, GitHubActionStep(stepLines))
    return steps
}
```

### Step 5: Implement MCP Configuration (Optional)

```go
func (e *MyEngine) RenderMCPConfig(yaml *strings.Builder, tools map[string]any, mcpTools []string, workflowData *WorkflowData) {
    myEngineLog.Printf("Rendering MCP config: tool_count=%d, mcp_tool_count=%d", len(tools), len(mcpTools))
    
    // Create unified renderer with engine-specific options
    createRenderer := func(isLast bool) *MCPConfigRendererUnified {
        return NewMCPConfigRenderer(MCPRendererOptions{
            IncludeCopilotFields: false,
            InlineArgs:           false,
            Format:               "json", // or "toml" for Codex-style
            IsLast:               isLast,
            ActionMode:           GetActionModeFromWorkflowData(workflowData),
        })
    }
    
    // Use shared JSON MCP config renderer
    _ = RenderJSONMCPConfig(yaml, tools, mcpTools, workflowData, JSONMCPConfigOptions{
        ConfigPath:    "/tmp/gh-aw/mcp-config/mcp-servers.json",
        GatewayConfig: buildMCPGatewayConfig(workflowData),
        Renderers: MCPToolRenderers{
            RenderGitHub: func(yaml *strings.Builder, githubTool any, isLast bool, workflowData *WorkflowData) {
                renderer := createRenderer(isLast)
                renderer.RenderGitHubMCP(yaml, githubTool, workflowData)
            },
            RenderPlaywright: func(yaml *strings.Builder, playwrightTool any, isLast bool) {
                renderer := createRenderer(isLast)
                renderer.RenderPlaywrightMCP(yaml, playwrightTool)
            },
            RenderSerena: func(yaml *strings.Builder, serenaTool any, isLast bool) {
                renderer := createRenderer(isLast)
                renderer.RenderSerenaMCP(yaml, serenaTool)
            },
            RenderCacheMemory: e.renderCacheMemoryMCPConfig,
            RenderAgenticWorkflows: func(yaml *strings.Builder, isLast bool) {
                renderer := createRenderer(isLast)
                renderer.RenderAgenticWorkflowsMCP(yaml)
            },
            RenderSafeOutputs: func(yaml *strings.Builder, isLast bool, workflowData *WorkflowData) {
                renderer := createRenderer(isLast)
                renderer.RenderSafeOutputsMCP(yaml, workflowData)
            },
            RenderSafeInputs: func(yaml *strings.Builder, safeInputs *SafeInputsConfig, isLast bool) {
                renderer := createRenderer(isLast)
                renderer.RenderSafeInputsMCP(yaml, safeInputs, workflowData)
            },
            RenderWebFetch: func(yaml *strings.Builder, isLast bool) {
                renderMCPFetchServerConfig(yaml, "json", "              ", isLast, false)
            },
            RenderCustomMCPConfig: func(yaml *strings.Builder, toolName string, toolConfig map[string]any, isLast bool) error {
                return renderCustomMCPConfigWrapperWithContext(yaml, toolName, toolConfig, isLast, workflowData)
            },
        },
    })
}

func (e *MyEngine) renderCacheMemoryMCPConfig(yaml *strings.Builder, isLast bool, workflowData *WorkflowData) {
    // Cache-memory is a simple file share, not an MCP server
    // No MCP configuration needed
}
```

### Step 6: Implement Log Parsing (Optional)

```go
func (e *MyEngine) ParseLogMetrics(logContent string, verbose bool) LogMetrics {
    myEngineLog.Printf("Parsing log metrics: log_size=%d bytes", len(logContent))
    
    var metrics LogMetrics
    lines := strings.Split(logContent, "\n")
    
    for _, line := range lines {
        // Parse engine-specific log format
        // Extract: turns, token usage, tool calls, errors
    }
    
    return metrics
}

func (e *MyEngine) GetLogParserScriptId() string {
    return "parse_my_engine_log"
}

func (e *MyEngine) GetLogFileForParsing() string {
    return "/tmp/gh-aw/agent-stdio.log"
}
```

### Step 7: Register Engine

```go
// In agentic_engine.go, add to NewEngineRegistry():
func NewEngineRegistry() *EngineRegistry {
    registry := &EngineRegistry{
        engines: make(map[string]CodingAgentEngine),
    }
    
    registry.Register(NewClaudeEngine())
    registry.Register(NewCodexEngine())
    registry.Register(NewCopilotEngine())
    registry.Register(NewCustomEngine())
    registry.Register(NewMyEngine()) // Add your engine here
    
    return registry
}
```

---

## Testing Strategy

### Unit Tests

Create `my_engine_test.go`:

```go
package workflow

import (
    "testing"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestMyEngine(t *testing.T) {
    engine := NewMyEngine()
    
    t.Run("engine identity", func(t *testing.T) {
        assert.Equal(t, "my-engine", engine.GetID())
        assert.Equal(t, "My Custom Engine", engine.GetDisplayName())
        assert.NotEmpty(t, engine.GetDescription())
    })
    
    t.Run("capabilities", func(t *testing.T) {
        assert.True(t, engine.SupportsToolsAllowlist())
        assert.True(t, engine.SupportsHTTPTransport())
        assert.True(t, engine.SupportsFirewall())
    })
    
    t.Run("required secrets", func(t *testing.T) {
        workflowData := &WorkflowData{Name: "test"}
        secrets := engine.GetRequiredSecretNames(workflowData)
        assert.Contains(t, secrets, "MY_ENGINE_API_KEY")
    })
}

func TestMyEngineInstallation(t *testing.T) {
    engine := NewMyEngine()
    workflowData := &WorkflowData{
        Name: "test-workflow",
    }
    
    steps := engine.GetInstallationSteps(workflowData)
    require.NotEmpty(t, steps, "Should generate installation steps")
    
    // Verify secret validation step exists
    hasSecretValidation := false
    for _, step := range steps {
        for _, line := range step {
            if strings.Contains(line, "validate-secret") {
                hasSecretValidation = true
                break
            }
        }
    }
    assert.True(t, hasSecretValidation, "Should include secret validation")
}

func TestMyEngineExecution(t *testing.T) {
    engine := NewMyEngine()
    workflowData := &WorkflowData{
        Name: "test-workflow",
        EngineConfig: &EngineConfig{
            ID: "my-engine",
        },
    }
    
    steps := engine.GetExecutionSteps(workflowData, "/tmp/test.log")
    require.NotEmpty(t, steps, "Should generate execution steps")
    
    // Verify command includes engine invocation
    hasEngineCommand := false
    for _, step := range steps {
        for _, line := range step {
            if strings.Contains(line, "my-engine") {
                hasEngineCommand = true
                break
            }
        }
    }
    assert.True(t, hasEngineCommand, "Should include engine command")
}
```

### Integration Tests

Create `my_engine_integration_test.go`:

```go
//go:build integration

package workflow

import (
    "testing"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestMyEngineWorkflowCompilation(t *testing.T) {
    compiler := NewCompiler()
    workflowPath := "testdata/my-engine-workflow.md"
    
    workflow, err := compiler.Compile(workflowPath)
    require.NoError(t, err)
    assert.NotNil(t, workflow)
    
    // Verify workflow structure
    assert.Equal(t, "my-engine", workflow.EngineID)
    assert.NotEmpty(t, workflow.InstallationSteps)
    assert.NotEmpty(t, workflow.ExecutionSteps)
}
```

---

## Integration Checklist

### Code Changes

- [ ] Create `my_engine.go` with engine implementation
- [ ] Create `my_engine_test.go` with unit tests
- [ ] Create `my_engine_integration_test.go` with integration tests
- [ ] Add engine registration in `agentic_engine.go`
- [ ] Add engine constants in `pkg/constants/constants.go`
- [ ] Create `my_engine_logs.go` if custom log parsing is needed
- [ ] Create `my_engine_mcp.go` if custom MCP rendering is needed

### Documentation

- [ ] Add engine documentation in `docs/src/content/docs/reference/engines/`
- [ ] Update engine comparison table
- [ ] Add setup instructions (API keys, configuration)
- [ ] Document required secrets and environment variables
- [ ] Add example workflows using the new engine

### Testing

- [ ] Run `make test-unit` - all unit tests pass
- [ ] Run `make test` - all integration tests pass
- [ ] Run `make lint` - no linting errors
- [ ] Run `make fmt` - code is properly formatted
- [ ] Test workflow compilation with new engine
- [ ] Test workflow execution (manual or CI)

### CI/CD

- [ ] Add engine-specific CI workflow if needed
- [ ] Update CI matrix to include new engine tests
- [ ] Verify Docker image includes engine dependencies
- [ ] Test in clean environment (no cached dependencies)

### Final Validation

- [ ] Run `make agent-finish` - complete validation passes
- [ ] Create PR with comprehensive description
- [ ] Request review from maintainers
- [ ] Address review feedback
- [ ] Merge when approved

---

## Best Practices

### 1. Use Shared Helpers

Always prefer existing helpers over duplicating code:
- `GetBaseInstallationSteps()` for standard installation
- `BuildAWFCommand()` for firewall integration
- `FormatStepWithCommandAndEnv()` for step formatting
- `FilterEnvForSecrets()` for security

### 2. Follow Naming Conventions

- Engine ID: lowercase with hyphens (e.g., `my-engine`)
- Logger: `workflow:engine_name` (e.g., `workflow:my_engine`)
- Files: `engine_name_*.go` (e.g., `my_engine.go`, `my_engine_logs.go`)
- Constants: `DefaultMyEngineVersion`, `MyEngineLLMGatewayPort`

### 3. Security First

- Always filter environment variables with `FilterEnvForSecrets()`
- Validate secrets before execution
- Use AWF firewall when `isFirewallEnabled()` returns true
- Never log sensitive information

### 4. Maintain Backward Compatibility

- Use interface composition, not breaking changes
- Override BaseEngine methods, don't replace them
- Support legacy configuration formats
- Document migration paths

### 5. Test Thoroughly

- Unit tests for core functionality
- Integration tests for workflow compilation
- Test with MCP servers enabled/disabled
- Test with firewall enabled/disabled
- Test custom configuration scenarios

---

## Common Pitfalls

### 1. Forgetting to Register Engine

Always add your engine to `NewEngineRegistry()` in `agentic_engine.go`.

### 2. Not Handling Custom Commands

Support custom commands via `workflowData.EngineConfig.Command`:

```go
commandName := "my-engine"
if workflowData.EngineConfig != nil && workflowData.EngineConfig.Command != "" {
    commandName = workflowData.EngineConfig.Command
}
```

### 3. Incorrect PATH Setup

Use `GetNpmBinPathSetup()` for npm-installed CLIs inside AWF:

```go
npmPathSetup := GetNpmBinPathSetup()
engineCommandWithPath := fmt.Sprintf("%s && %s", npmPathSetup, engineCommand)
```

### 4. Missing Secret Filtering

Always filter environment variables:

```go
allowedSecrets := e.GetRequiredSecretNames(workflowData)
filteredEnv := FilterEnvForSecrets(env, allowedSecrets)
```

### 5. Hardcoding Paths

Use constants and configuration:

```go
// ❌ BAD
logFile := "/tmp/my-log.txt"

// ✅ GOOD
logFile := workflowData.LogFile // or passed as parameter
```

---

## Summary

Implementing a custom agentic engine involves:

1. **Understanding the architecture**: Interface segregation with focused responsibilities
2. **Leveraging existing helpers**: Don't reinvent the wheel
3. **Following patterns**: Learn from existing engines (Copilot, Claude, Codex)
4. **Testing thoroughly**: Unit tests, integration tests, manual validation
5. **Documenting completely**: Help users understand and use your engine

The gh-aw codebase provides excellent infrastructure for engine development. Use the shared helpers, follow the patterns, and focus on your engine's unique capabilities.

For questions or clarifications, refer to existing engine implementations:
- **Copilot** (`copilot_engine*.go`): Well-modularized, clean separation
- **Claude** (`claude_engine.go`): Comprehensive, feature-rich
- **Codex** (`codex_engine.go`): Regex-based log parsing
- **Custom** (`custom_engine.go`): Minimal, flexible implementation

Happy coding! 🚀
