# workflow Package

> Workflow compilation, validation, engine integration, safe-outputs, and GitHub Actions YAML generation for agentic workflow files.

## Overview

The `workflow` package is the compilation core of `gh-aw`. It transforms parsed markdown frontmatter (from `pkg/parser`) and markdown body text into complete GitHub Actions `.lock.yml` files. Compilation covers the full lifecycle: frontmatter parsing into strongly-typed configuration structs, multi-pass validation (schema, permissions, security, strict mode), engine-specific step generation (Copilot, Claude, Codex, Gemini, custom), safe-output job construction, and final YAML serialization.

The package is organized around three major subsystems:

1. **Compiler** (`compiler*.go`, `compiler_types.go`): The `Compiler` struct drives the main compilation pipeline. It accepts a markdown file path (or pre-parsed `WorkflowData`), builds the full GitHub Actions workflow YAML, and writes the `.lock.yml` file only when the content has changed.

2. **Engine registry** (`agentic_engine.go`, `*_engine.go`): A pluggable engine architecture where each AI engine (`copilot`, `claude`, `codex`, `gemini`, `crush`, `custom`) implements a set of focused interfaces (`Engine`, `CapabilityProvider`, `WorkflowExecutor`, `MCPConfigProvider`, etc.). Engines are registered in a global `EngineRegistry` and looked up by name at compile time.

3. **Validation** (`validation.go`, `strict_mode_*.go`, `*_validation.go`): A layered validation system organized by domain. Each validator is a focused file under 300 lines. Validation runs both at compile time and optionally in strict mode for production deployments.

The package is intentionally large (~320 source files) because it encodes all GitHub Actions generation logic, including per-action job builders for every supported safe-output type (add comment, add labels, assign to user, close issue, update PR, etc.).

## Public API

### Core Compiler Types

| Type | Kind | Description |
|------|------|-------------|
| `Compiler` | struct | Main compilation engine; use `NewCompiler(opts...)` |
| `CompilerOption` | func type | Functional option for configuring a `Compiler` |
| `WorkflowData` | struct | Complete in-memory representation of a compiled workflow |
| `FileTracker` | interface | Abstraction for tracking written files |

#### `Compiler` Methods

| Method | Signature | Description |
|--------|-----------|-------------|
| `CompileWorkflow` | `func(markdownPath string) error` | Compiles a markdown file and writes the `.lock.yml` |
| `CompileWorkflowData` | `func(workflowData *WorkflowData, markdownPath string) error` | Compiles pre-parsed `WorkflowData` |

#### Compiler Options

| Function | Description |
|----------|-------------|
| `WithVerbose(bool)` | Enable verbose diagnostic output |
| `WithEngineOverride(string)` | Override the AI engine |
| `WithSkipValidation(bool)` | Skip schema validation |
| `WithNoEmit(bool)` | Validate without writing lock files |
| `WithFailFast(bool)` | Stop at first validation error |
| `WithWorkflowIdentifier(string)` | Set the workflow identifier |
| `NewCompiler(opts ...CompilerOption)` | Creates a new `Compiler` |
| `NewCompilerWithVersion(string)` | Creates a `Compiler` with a specific version |

### Engine Architecture

| Type | Kind | Description |
|------|------|-------------|
| `Engine` | interface | Core identity: `GetID()`, `GetDisplayName()`, `GetDescription()`, `IsExperimental()` |
| `CapabilityProvider` | interface | Optional feature detection (`SupportsToolsAllowlist`, `SupportsMaxTurns`, etc.) |
| `WorkflowExecutor` | interface | Compilation: `GetDeclaredOutputFiles`, `GetInstallationSteps`, `GetExecutionSteps` |
| `MCPConfigProvider` | interface | MCP configuration generation |
| `LogParser` | interface | Log parsing for audit/metrics |
| `SecurityProvider` | interface | Security-related configuration |
| `ModelEnvVarProvider` | interface | Model environment variable mapping |
| `AgentFileProvider` | interface | Custom agent file support |
| `ConfigRenderer` | interface | Configuration file rendering |
| `DriverProvider` | interface | Driver-level execution configuration |
| `CodingAgentEngine` | interface | Composite interface combining all engine capabilities |
| `BaseEngine` | struct | Base implementation shared by all engines |
| `EngineRegistry` | struct | Global registry mapping engine names to implementations |
| `CopilotEngine` | struct | Copilot coding agent engine |
| `ClaudeEngine` | struct | Claude coding agent engine |
| `CodexEngine` | struct | OpenAI Codex coding agent engine |
| `GeminiEngine` | struct | Google Gemini CLI coding agent engine |

#### Engine Registry Functions

| Function | Signature | Description |
|----------|-----------|-------------|
| `NewEngineRegistry` | `func() *EngineRegistry` | Creates a new engine registry |
| `GetGlobalEngineRegistry` | `func() *EngineRegistry` | Returns the singleton global engine registry |
| `NewCopilotEngine` | `func() *CopilotEngine` | Creates the Copilot engine |
| `NewClaudeEngine` | `func() *ClaudeEngine` | Creates the Claude engine |
| `NewCodexEngine` | `func() *CodexEngine` | Creates the Codex engine |
| `NewGeminiEngine` | `func() *GeminiEngine` | Creates the Gemini engine |

### Frontmatter Configuration Types

| Type | Kind | Description |
|------|------|-------------|
| `FrontmatterConfig` | struct | Full parsed frontmatter with typed and legacy fields |
| `RuntimeConfig` | struct | Single runtime version configuration (version string) |
| `RuntimesConfig` | struct | All runtime versions (node, python, go, uv, bun, deno) |
| `PermissionsConfig` | struct | GitHub Actions permissions (shorthand + detailed fields) |
| `GitHubActionsPermissionsConfig` | struct | Detailed permissions with all scope fields |
| `GitHubAppPermissionsConfig` | struct | GitHub App permission scopes |
| `ObservabilityConfig` | struct | OTLP/observability configuration |
| `RateLimitConfig` | struct | Rate limit settings |
| `OTLPConfig` | struct | OpenTelemetry protocol configuration |

### Permissions System

| Type | Kind | Description |
|------|------|-------------|
| `Permissions` | struct | Runtime permissions representation for the compiled workflow |
| `PermissionLevel` | string alias | Permission level: `read`, `write`, `none` |
| `PermissionScope` | string alias | Permission scope (e.g., `contents`, `issues`, `pull-requests`) |
| `PermissionsParser` | struct | Parses YAML permissions blocks into `Permissions` |
| `PermissionsValidationResult` | struct | Result of `ValidatePermissions` |

#### Permissions Functions

| Function | Signature | Description |
|----------|-----------|-------------|
| `NewPermissionsParser` | `func(permissionsYAML string) *PermissionsParser` | Creates a parser from YAML text |
| `NewPermissionsParserFromValue` | `func(permissionsValue any) *PermissionsParser` | Creates a parser from parsed YAML value |
| `ValidatePermissions` | `func(*Permissions, ValidatableTool) *PermissionsValidationResult` | Validates permissions for a given tool |
| `FormatValidationMessage` | `func(*PermissionsValidationResult, bool) string` | Formats a validation result as a human-readable message |
| `ComputePermissionsForSafeOutputs` | `func(*SafeOutputsConfig) *Permissions` | Computes required permissions for safe-output types |
| `SortPermissionScopes` | `func([]PermissionScope)` | Sorts permission scopes alphabetically |

#### Permissions Factory (common combinations)

| Function | Description |
|----------|-------------|
| `NewPermissionsContentsWritePRWrite()` | contents:write + pull-requests:write |
| `NewPermissionsContentsWriteIssuesWritePRWrite()` | contents:write + issues:write + pull-requests:write |
| `NewPermissionsContentsReadDiscussionsWrite()` | contents:read + discussions:write |
| `NewPermissionsContentsReadIssuesWriteDiscussionsWrite()` | contents:read + issues:write + discussions:write |
| `NewPermissionsContentsReadPRWrite()` | contents:read + pull-requests:write |
| `NewPermissionsContentsReadSecurityEventsWrite()` | contents:read + security-events:write |
| `NewPermissionsContentsReadProjectsWrite()` | contents:read + projects:write |

### Tools Configuration

| Type | Kind | Description |
|------|------|-------------|
| `ToolsConfig` | struct | Parsed `tools:` block with all tool configurations |
| `Tools` | type alias | Alias for `ToolsConfig` |
| `GitHubToolConfig` | struct | GitHub MCP tool configuration (toolsets, allowed tools, integrity) |
| `PlaywrightToolConfig` | struct | Playwright browser automation tool config |
| `BashToolConfig` | struct | Bash execution tool config |
| `WebFetchToolConfig` | struct | Web fetch tool config |
| `WebSearchToolConfig` | struct | Web search tool config |
| `EditToolConfig` | struct | File edit tool config |
| `AgenticWorkflowsToolConfig` | struct | Nested agentic workflows tool config |
| `CacheMemoryToolConfig` | struct | Cache-memory persistence tool config |
| `MCPServerConfig` | struct | Generic MCP server configuration |
| `MCPGatewayRuntimeConfig` | struct | MCP Gateway runtime configuration |
| `GitHubToolName` | string alias | Named GitHub MCP tool (e.g., `"issue_read"`) |
| `GitHubAllowedTools` | `[]GitHubToolName` | Typed slice with conversion helpers |
| `GitHubToolset` | string alias | Named GitHub toolset (e.g., `"default"`, `"repos"`) |
| `GitHubToolsets` | `[]GitHubToolset` | Typed slice with conversion helpers |
| `GitHubIntegrityLevel` | string alias | Integrity level (`"low"`, `"medium"`, `"high"`) |

#### Tools Functions

| Function | Signature | Description |
|----------|-----------|-------------|
| `NewTools` | `func(map[string]any) *Tools` | Creates a `Tools` from a raw map |
| `ParseToolsConfig` | `func(map[string]any) (*ToolsConfig, error)` | Parses the `tools:` frontmatter section |
| `ValidateGitHubToolsAgainstToolsets` | `func([]string, []string) error` | Validates tool names against enabled toolsets |
| `GetPlaywrightTools` | `func() []any` | Returns the standard Playwright tool definitions |
| `GetSafeOutputToolOptions` | `func() []SafeOutputToolOption` | Returns valid safe-output tool option definitions |
| `GetValidationConfigJSON` | `func(enabledTypes []string) (string, error)` | Returns JSON validation config for given safe-output types |

### Safe Outputs

| Type | Kind | Description |
|------|------|-------------|
| `SafeOutputsConfig` | struct | Parsed `safe-outputs:` configuration |
| `SafeOutputTargetConfig` | struct | Target configuration for a safe-output job |
| `SafeOutputFilterConfig` | struct | Filter configuration for a safe-output job |
| `SafeOutputToolOption` | struct | A valid safe-output tool option |

#### Safe Output Functions

| Function | Signature | Description |
|----------|-----------|-------------|
| `HasSafeOutputsEnabled` | `func(*SafeOutputsConfig) bool` | Returns whether any safe-output type is enabled |
| `ParseTargetConfig` | `func(map[string]any) (SafeOutputTargetConfig, bool)` | Parses a target configuration block |
| `ParseFilterConfig` | `func(map[string]any) SafeOutputFilterConfig` | Parses a filter configuration block |
| `SafeOutputsConfigFromKeys` | `func([]string) *SafeOutputsConfig` | Creates a config from a list of type keys |

### Network Permissions

| Type | Kind | Description |
|------|------|-------------|
| `NetworkPermissions` | struct | Parsed `network:` block with `allowed` and `blocked` domain lists |

#### Network Functions

| Function | Signature | Description |
|----------|-----------|-------------|
| `GetAllowedDomains` | `func(*NetworkPermissions) []string` | Returns the full list of allowed domains |
| `GetDomainEcosystem` | `func(domain string) string` | Returns the ecosystem name for a domain |
| `GetAllowedDomainsForEngine` | `func(EngineName, *NetworkPermissions, ...) string` | Returns allowed domains for a specific engine |
| `GetCopilotAllowedDomainsWithToolsAndRuntimes` | `func(*NetworkPermissions, ...) string` | Copilot-specific allowed domains |
| `GetCodexAllowedDomainsWithToolsAndRuntimes` | `func(*NetworkPermissions, ...) string` | Codex-specific allowed domains |
| `GetClaudeAllowedDomainsWithToolsAndRuntimes` | `func(*NetworkPermissions, ...) string` | Claude-specific allowed domains |
| `GetGeminiAllowedDomainsWithToolsAndRuntimes` | `func(*NetworkPermissions, ...) string` | Gemini-specific allowed domains |
| `GetThreatDetectionAllowedDomains` | `func(*NetworkPermissions) string` | Allowed domains for threat detection jobs |

### Error Types

| Type | Kind | Description |
|------|------|-------------|
| `WorkflowValidationError` | struct | Validation error with field, value, reason, and suggestion |
| `OperationError` | struct | Error from a workflow operation with entity context |
| `ConfigurationError` | struct | Configuration error with config key and suggested fix |
| `ErrorCollector` | struct | Collects multiple errors; supports `failFast` mode |
| `SharedWorkflowError` | struct | Error for shared/reusable workflow violations |

#### Error Functions

| Function | Signature | Description |
|----------|-----------|-------------|
| `NewValidationError` | `func(field, value, reason, suggestion string) *WorkflowValidationError` | Creates a validation error |
| `NewOperationError` | `func(operation, entityType, entityID string, cause error, suggestion string) *OperationError` | Creates an operation error |
| `NewConfigurationError` | `func(configKey, value, reason, suggestion string) *ConfigurationError` | Creates a configuration error |
| `NewErrorCollector` | `func(failFast bool) *ErrorCollector` | Creates an error collector |

### Workflow Resolution

| Function | Signature | Description |
|----------|-----------|-------------|
| `ResolveWorkflowName` | `func(string) (string, error)` | Resolves a workflow input string to a canonical name |
| `FindWorkflowName` | `func(string) (string, error)` | Finds the workflow name from a string or file path |
| `GetWorkflowLockFileName` | `func(string) (string, error)` | Returns the `.lock.yml` path for a workflow |
| `GetAllWorkflows` | `func() ([]WorkflowNameMatch, error)` | Returns all installed workflow names |
| `GetWorkflowIDFromPath` | `func(string) string` | Derives the workflow ID from its markdown path |

### Action Pinning

| Type | Kind | Description |
|------|------|-------------|
| `ActionPin` | struct | An action pin (repo + SHA) |
| `ActionPinsData` | struct | Map of all action pins |
| `ActionMode` | string alias | Action reference mode (`sha`, `tag`, `local`) |
| `ActionCache` | struct | Cache for resolved action SHAs |
| `ActionResolver` | struct | Resolves action SHAs from GitHub |

| Function | Signature | Description |
|----------|-----------|-------------|
| `GetActionPin` | `func(actionRepo string) string` | Returns the pinned SHA for an action |
| `GetActionPinByRepo` | `func(string) (ActionPin, bool)` | Looks up a pin by repo |
| `DetectActionMode` | `func(version string) ActionMode` | Detects the action reference mode |
| `ParseTagRefTSV` | `func(line string) (sha, objType string, err error)` | Parses tab-separated tag ref output into SHA and object type |
| `ExtractActionsFromLockFile` | `func(lockFilePath string) ([]ActionUsage, error)` | Extracts action usages from a lock file |
| `CheckActionSHAUpdates` | `func(actions []ActionUsage, resolver *ActionResolver) []ActionUpdateCheck` | Checks whether action SHAs need updates |
| `ApplyActionPinsToTypedSteps` | `func([]*WorkflowStep, *WorkflowData) []*WorkflowStep` | Applies pins to all steps |
| `ValidateActionSHAsInLockFile` | `func(string, *ActionCache, bool) error` | Validates action SHAs in a lock file |

### String Utilities (Workflow-Specific)

| Function | Signature | Description |
|----------|-----------|-------------|
| `SanitizeName` | `func(string, *SanitizeOptions) string` | Sanitizes a name for use in GitHub Actions |
| `SanitizeWorkflowName` | `func(string) string` | Sanitizes a workflow name |
| `SanitizeIdentifier` | `func(string) string` | Sanitizes a generic identifier |
| `SanitizeWorkflowIDForCacheKey` | `func(string) string` | Sanitizes a workflow ID for use as a cache key |
| `PrettifyToolName` | `func(string) string` | Returns a human-readable tool name |
| `ShortenCommand` | `func(string) string` | Shortens a long command for display |
| `GenerateHeredocDelimiterFromSeed` | `func(name, seed string) string` | Generates a stable heredoc delimiter |
| `ValidateHeredocContent` | `func(content, delimiter string) error` | Validates heredoc content safety |
| `ValidateHeredocDelimiter` | `func(string) error` | Validates a heredoc delimiter |

### Secret Handling

| Function | Signature | Description |
|----------|-----------|-------------|
| `ExtractSecretName` | `func(string) string` | Extracts the secret name from a `${{ secrets.NAME }}` expression |
| `ExtractSecretsFromValue` | `func(string) map[string]string` | Extracts all secrets from a template value |
| `ReplaceSecretsWithEnvVars` | `func(string, map[string]string) string` | Replaces secret references with env var references |
| `ExtractGitHubContextExpressionsFromValue` | `func(string) map[string]string` | Extracts GitHub context expressions |
| `CollectSecretReferences` | `func(string) []string` | Collects all secret references from YAML content |
| `CollectActionReferences` | `func(string) []string` | Collects all action references from YAML content |

### Concurrency & Scheduling

| Function | Signature | Description |
|----------|-----------|-------------|
| `GenerateConcurrencyConfig` | `func(*WorkflowData, bool) string` | Generates `concurrency:` YAML for a workflow |
| `GenerateJobConcurrencyConfig` | `func(*WorkflowData) string` | Generates job-level concurrency YAML |
| `ResolveRelativeDate` | `func(dateStr string, baseTime time.Time) (string, error)` | Resolves relative date strings (e.g., "2 weeks ago") |

### YAML Utilities

| Function | Signature | Description |
|----------|-----------|-------------|
| `UnquoteYAMLKey` | `func(yamlStr, key string) string` | Removes unnecessary quotes from a YAML key |
| `MarshalWithFieldOrder` | `func(map[string]any, []string) ([]byte, error)` | Marshals a map with priority-ordered fields |
| `OrderMapFields` | `func(map[string]any, []string) yaml.MapSlice` | Returns an ordered map slice |
| `CleanYAMLNullValues` | `func(string) string` | Removes null values from YAML output |
| `ConvertStepToYAML` | `func(map[string]any) (string, error)` | Converts a step map to YAML text |

### Trigger Parsing

| Type | Kind | Description |
|------|------|-------------|
| `TriggerIR` | struct | Intermediate representation of a workflow trigger |

| Function | Signature | Description |
|----------|-----------|-------------|
| `ParseTriggerShorthand` | `func(string) (*TriggerIR, error)` | Parses a trigger shorthand string |

### AWF Command Building

| Function | Signature | Description |
|----------|-----------|-------------|
| `BuildAWFCommand` | `func(AWFCommandConfig) string` | Builds the `gh aw` command string for a workflow step |
| `BuildAWFArgs` | `func(AWFCommandConfig) []string` | Builds CLI argument list for `gh aw` |
| `GetAWFCommandPrefix` | `func(*WorkflowData) string` | Returns the `gh aw` command prefix |
| `WrapCommandInShell` | `func(string) string` | Wraps a command in a shell `run:` block |
| `GetCopilotAPITarget` | `func(*WorkflowData) string` | Returns the Copilot API target URL |
| `GetGeminiAPITarget` | `func(*WorkflowData, string) string` | Returns the Gemini API target hostname |
| `ComputeAWFExcludeEnvVarNames` | `func(*WorkflowData, []string) []string` | Computes secret-backed env var names to exclude from AWF |

### Versioning

| Function | Signature | Description |
|----------|-----------|-------------|
| `SetVersion` | `func(string)` | Sets the package version at startup |
| `GetVersion` | `func() string` | Returns the current package version |
| `SetIsRelease` | `func(bool)` | Marks whether this is a release build |
| `IsRelease` | `func() bool` | Returns whether this is a release build |
| `IsReleasedVersion` | `func(string) bool` | Checks whether a version string is a release |

### Validation Functions

| Function | Signature | Description |
|----------|-----------|-------------|
| `ValidateEventFilters` | `func(map[string]any) error` | Validates `on:` event filter patterns |
| `ValidateGlobPatterns` | `func(map[string]any) error` | Validates glob patterns in trigger filters |

### Step Types

| Type | Kind | Description |
|------|------|-------------|
| `WorkflowStep` | struct | A single GitHub Actions step with all standard fields |
| `GitHubActionStep` | `[]string` | A multi-line run step (slice of command strings) |

| Function | Signature | Description |
|----------|-----------|-------------|
| `MapToStep` | `func(map[string]any) (*WorkflowStep, error)` | Converts a YAML map to a typed `WorkflowStep` |
| `SliceToSteps` | `func([]any) ([]*WorkflowStep, error)` | Converts a YAML slice to typed steps |
| `StepsToSlice` | `func([]*WorkflowStep) []any` | Converts typed steps back to a YAML slice |

### Repository Configuration

| Type | Kind | Description |
|------|------|-------------|
| `RepoConfig` | struct | Repository-level configuration from `.github/gh-aw.yml` |

| Function | Signature | Description |
|----------|-----------|-------------|
| `LoadRepoConfig` | `func(gitRoot string) (*RepoConfig, error)` | Loads and parses the repo config file |
| `FormatRunsOn` | `func(RunsOnValue, string) string` | Formats a `runs-on:` value for YAML output |

### Threat Detection

| Type | Kind | Description |
|------|------|-------------|
| `ThreatDetectionConfig` | struct | Configuration for the threat detection job |

| Function | Signature | Description |
|----------|-----------|-------------|
| `IsDetectionJobEnabled` | `func(*SafeOutputsConfig) bool` | Returns whether threat detection is enabled |

### Safe Update Manifest

| Type | Kind | Description |
|------|------|-------------|
| `GHAWManifest` | struct | Signed manifest embedded in lock files for integrity checking |

| Function | Signature | Description |
|----------|-----------|-------------|
| `NewGHAWManifest` | `func(secretNames, actionRefs []string, containers []GHAWManifestContainer) *GHAWManifest` | Creates a new manifest |
| `ExtractGHAWManifestFromLockFile` | `func(string) (*GHAWManifest, error)` | Extracts the manifest from a lock file |
| `EnforceSafeUpdate` | `func(*GHAWManifest, []string, []string) error` | Validates that a lock file update passes manifest checks |

## Usage Examples

### Compile a workflow file

```go
compiler := workflow.NewCompiler(
    workflow.WithVerbose(true),
    workflow.WithEngineOverride("copilot"),
)
err := compiler.CompileWorkflow(".github/workflows/my-workflow.md")
```

### Look up an engine

```go
registry := workflow.GetGlobalEngineRegistry()
engine, ok := registry.Get("copilot")
if ok {
    steps := engine.GetExecutionSteps(workflowData)
}
```

### Compute permissions for safe outputs

```go
perms := workflow.ComputePermissionsForSafeOutputs(safeOutputsConfig)
```

### Resolve a workflow name

```go
name, err := workflow.ResolveWorkflowName("my-workflow")
lockFile, err := workflow.GetWorkflowLockFileName(name)
```

## Architecture

```
markdown file
    │
    ▼
pkg/parser ─── ExtractFrontmatterFromContent
    │               ProcessImportsFromFrontmatterWithSource
    │
    ▼
pkg/workflow ── FrontmatterConfig (typed structs)
    │               Compiler.CompileWorkflow()
    │                 ├─ schema validation
    │                 ├─ permissions computation
    │                 ├─ engine step generation
    │                 ├─ safe-output job generation
    │                 ├─ YAML serialization
    │                 └─ lock file write (if changed)
    │
    ▼
.github/workflows/my-workflow.lock.yml
```

## Design Decisions

- **File-per-domain decomposition**: Each validation concern and job-builder lives in its own file. The 300-line limit is enforced by convention; validation files exceeding it SHOULD be split.
- **Functional compiler options**: `CompilerOption` functions follow the standard Go functional-options pattern, keeping `NewCompiler` signature stable as options are added.
- **Engine interface composition**: Rather than one monolithic `Engine` interface, capabilities are split into focused interfaces (`CapabilityProvider`, `WorkflowExecutor`, etc.) and combined via `CodingAgentEngine`. This prevents engines from being forced to implement unused methods.
- **Content-addressed lock files**: Lock files are only written when the normalized YAML content changes (heredoc delimiters are normalized before comparison). This avoids unnecessary git churn.
- **YAML 1.1/1.2 compatibility**: The package uses `goccy/go-yaml` for all GitHub Actions YAML generation to ensure compatibility with GitHub Actions' YAML parser.

## Dependencies

**Internal**:
- `pkg/parser` — frontmatter extraction and import processing
- `pkg/constants` — engine names, feature flags, job/step IDs
- `pkg/console` — terminal formatting
- `pkg/logger` — debug logging
- `pkg/actionpins` — action pin data and pin lookup helpers
- `pkg/semverutil` — semantic version helpers
- `pkg/typeutil` — safe type conversions
- `pkg/tty` — terminal capability detection
- `pkg/stringutil`, `pkg/fileutil`, `pkg/gitutil`, `pkg/sliceutil` — utilities
- `pkg/types` — shared MCP types

**External**:
- `goccy/go-yaml` — YAML 1.1/1.2 compatible marshaling
- `go.yaml.in/yaml/v3` — standard YAML marshaling for non-Actions YAML

## Thread Safety

`Compiler` instances are NOT safe for concurrent use. Create a new `Compiler` for each concurrent compilation. The `GetGlobalEngineRegistry()` singleton is initialized once at startup and is safe for concurrent reads thereafter.

Constants (`MaxLockFileSize`) and action pin data are read-only after initialization and are safe for concurrent access.

---

*This specification is automatically maintained by the [spec-extractor](../../.github/workflows/spec-extractor.md) workflow.*
