# constants Package

The `constants` package provides shared semantic type aliases and named constants used across multiple `gh-aw` packages. Centralizing these values ensures consistency and type safety throughout the codebase.

## Overview

The package is organized into focused files:

| File | Contents |
|------|----------|
| `constants.go` | Core types, formatting constants, runtime config, container images, mounts, AWF |
| `engine_constants.go` | AI engine names, options, system secrets, model env vars, Copilot CLI commands |
| `feature_constants.go` | Feature flag identifiers |
| `job_constants.go` | GitHub Actions job names, step IDs, artifact names, output keys |
| `tool_constants.go` | Allowed GitHub tool expressions and default tool lists |
| `url_constants.go` | URL semantic types, well-known URLs, documentation URLs |
| `version_constants.go` | Default version strings and minimum version constraints |

## Semantic Types

The package uses typed aliases to prevent mixing unrelated string or integer values:

| Type | Description | Example constant |
|------|-------------|-----------------|
| `EngineName` | AI engine identifier | `CopilotEngine`, `ClaudeEngine`, `CodexEngine`, `GeminiEngine` |
| `FeatureFlag` | Feature flag identifier | `MCPGatewayFeatureFlag`, `MCPScriptsFeatureFlag` |
| `JobName` | GitHub Actions job name | `AgentJobName`, `ActivationJobName` |
| `StepID` | GitHub Actions step identifier | `CheckMembershipStepID`, `CheckRateLimitStepID` |
| `MCPServerID` | MCP server identifier | `SafeOutputsMCPServerID`, `MCPScriptsMCPServerID` |
| `LineLength` | Character count for formatting | `MaxExpressionLineLength` (120) |
| `CommandPrefix` | CLI command prefix | `CLIExtensionPrefix` ("gh aw") |
| `WorkflowID` | User-provided workflow basename (no `.md`) | — |
| `Version` | Software version string | `DefaultCopilotVersion`, `DefaultNodeVersion` |
| `ModelName` | AI model name | — |
| `URL` | URL string | `DefaultMCPRegistryURL`, `PublicGitHubHost` |
| `DocURL` | Documentation URL | `DocsEnginesURL`, `DocsToolsURL` |

All semantic types implement `String() string` and `IsValid() bool` methods.

## Engine Constants

```go
import "github.com/github/gh-aw/pkg/constants"

// Engine names
constants.CopilotEngine  // "copilot"
constants.ClaudeEngine   // "claude"
constants.CodexEngine    // "codex"
constants.GeminiEngine   // "gemini"
constants.CrushEngine    // "crush"
constants.DefaultEngine  // "copilot"

// All supported engine names
constants.AgenticEngines // []string{"claude", "codex", "copilot", "gemini", "crush"}

// Get engine metadata
opt := constants.GetEngineOption("copilot")
// opt.Label = "GitHub Copilot"
// opt.SecretName = "COPILOT_GITHUB_TOKEN"
// opt.KeyURL = "https://github.com/settings/personal-access-tokens/new"

// Get all secret names for all engines
secrets := constants.GetAllEngineSecretNames()
```

### `EngineOption`

Describes a selectable AI engine with display metadata and required secret information:
- `Value`, `Label`, `Description` — display information
- `SecretName` — the primary secret required (e.g. `COPILOT_GITHUB_TOKEN`)
- `AlternativeSecrets` — secondary secret names that can be used instead
- `EnvVarName` — alternative environment variable name (if different from `SecretName`)
- `KeyURL` — URL where users can obtain their API key
- `WhenNeeded` — human-readable description of when this secret is needed

### `SystemSecretSpec`

Describes a system-level secret not tied to a specific engine (e.g. `GH_AW_GITHUB_TOKEN`):
- `Name` — environment variable name
- `WhenNeeded` — when to configure this secret
- `Description` — what the secret is and what permissions it needs
- `Optional` — whether the secret is optional

`SystemSecrets` is the global `[]SystemSecretSpec` slice containing `GH_AW_GITHUB_TOKEN`, `GH_AW_AGENT_TOKEN`, and `GH_AW_GITHUB_MCP_SERVER_TOKEN`.

### Model Environment Variables

These constants are used to configure AI model overrides at runtime:

```go
// gh-aw model override env vars (set in workflow environment or runner)
constants.EnvVarModelAgentCopilot    // "GH_AW_MODEL_AGENT_COPILOT"
constants.EnvVarModelAgentClaude     // "GH_AW_MODEL_AGENT_CLAUDE"
constants.EnvVarModelAgentCodex      // "GH_AW_MODEL_AGENT_CODEX"
constants.EnvVarModelAgentCustom     // "GH_AW_MODEL_AGENT_CUSTOM"
constants.EnvVarModelAgentGemini     // "GH_AW_MODEL_AGENT_GEMINI"
constants.EnvVarModelAgentCrush      // "GH_AW_MODEL_AGENT_CRUSH"
constants.EnvVarModelDetectionCopilot// "GH_AW_MODEL_DETECTION_COPILOT"
constants.EnvVarModelDetectionClaude // "GH_AW_MODEL_DETECTION_CLAUDE"
constants.EnvVarModelDetectionCodex  // "GH_AW_MODEL_DETECTION_CODEX"
constants.EnvVarModelDetectionGemini // "GH_AW_MODEL_DETECTION_GEMINI"
constants.EnvVarModelDetectionCrush  // "GH_AW_MODEL_DETECTION_CRUSH"

// Native CLI model env vars (passed directly to the engine CLI)
constants.CopilotCLIModelEnvVar         // "COPILOT_MODEL"
constants.CopilotCLIIntegrationIDEnvVar // "GITHUB_COPILOT_INTEGRATION_ID"
constants.ClaudeCLIModelEnvVar          // "ANTHROPIC_MODEL"
constants.GeminiCLIModelEnvVar          // "GEMINI_MODEL"
constants.CrushCLIModelEnvVar           // "CRUSH_MODEL"

// gh-aw runtime env vars
constants.EnvVarPrompt          // "GH_AW_PROMPT"
constants.EnvVarMCPConfig       // "GH_AW_MCP_CONFIG"
constants.EnvVarSafeOutputs     // "GH_AW_SAFE_OUTPUTS"
constants.EnvVarMaxTurns        // "GH_AW_MAX_TURNS"
constants.EnvVarStartupTimeout  // "GH_AW_STARTUP_TIMEOUT"
constants.EnvVarToolTimeout     // "GH_AW_TOOL_TIMEOUT"
constants.EnvVarGitHubToken     // "GH_AW_GITHUB_TOKEN"
```

### Copilot Stem Commands

`CopilotStemCommands` (`map[string]bool`) lists the shell commands (e.g. `git`, `gh`, `npm`) that the Copilot CLI is permitted to invoke. Used during security validation.

## Feature Flags

```go
constants.MCPScriptsFeatureFlag             // "mcp-scripts"
constants.MCPGatewayFeatureFlag             // "mcp-gateway"
constants.DisableXPIAPromptFeatureFlag      // "disable-xpia-prompt"
constants.CopilotRequestsFeatureFlag        // "copilot-requests"
constants.DIFCProxyFeatureFlag              // "difc-proxy" (deprecated — use tools.github.integrity-proxy)
constants.CliProxyFeatureFlag               // "cli-proxy"
constants.AwfDiagnosticLogsFeatureFlag      // "awf-diagnostic-logs"
constants.ByokCopilotFeatureFlag            // "byok-copilot"
constants.IntegrityReactionsFeatureFlag     // "integrity-reactions"
constants.MCPCLIFeatureFlag                 // "mcp-cli"
```

## Job and Step Constants

### Job Names

```go
constants.AgentJobName               // "agent"
constants.ActivationJobName          // "activation"
constants.PreActivationJobName       // "pre_activation"
constants.DetectionJobName           // "detection"
constants.SafeOutputsJobName         // "safe_outputs"
constants.UploadAssetsJobName        // "upload_assets"
constants.UploadCodeScanningJobName  // "upload_code_scanning_sarif"
constants.ConclusionJobName          // "conclusion"
constants.UnlockJobName              // "unlock"
```

### Artifact Names

```go
// Artifact containers (uploaded as GitHub Actions artifacts)
constants.SafeOutputArtifactName      // "safe-output"
constants.AgentOutputArtifactName     // "agent-output"
constants.AgentArtifactName           // "agent" (unified agent artifact)
constants.DetectionArtifactName       // "detection"
constants.LegacyDetectionArtifactName // "threat-detection.log" (backward compat)
constants.ActivationArtifactName      // "activation"
constants.SafeOutputItemsArtifactName // "safe-outputs-items"
constants.SarifArtifactName           // "code-scanning-sarif"

// Files written inside artifact directories
constants.AgentOutputFilename         // "agent_output.json"
constants.SafeOutputsFilename         // "safeoutputs.jsonl"
constants.TokenUsageFilename          // "agent_usage.json"
constants.GithubRateLimitsFilename    // "github_rate_limits.jsonl"
constants.OtelJsonlFilename           // "otel.jsonl"
constants.TemporaryIdMapFilename      // "temporary-id-map.json"
constants.SarifFileName               // "code-scanning-alert.sarif"
constants.SarifArtifactDownloadPath   // "/tmp/gh-aw/sarif/"

// Job output names
constants.ArtifactPrefixOutputName    // "artifact_prefix"
constants.FirewallAuditArtifactName   // "firewall-audit-logs" (legacy, for old runs)
```

### Step IDs

```go
// Pre-activation gate step IDs
constants.CheckMembershipStepID          // "check_membership"
constants.CheckRateLimitStepID           // "check_rate_limit"
constants.CheckStopTimeStepID            // "check_stop_time"
constants.CheckSkipIfMatchStepID         // "check_skip_if_match"
constants.CheckSkipIfNoMatchStepID       // "check_skip_if_no_match"
constants.CheckCommandPositionStepID     // "check_command_position"
constants.CheckSkipRolesStepID           // "check_skip_roles"
constants.CheckSkipBotsStepID            // "check_skip_bots"
constants.CheckSkipIfCheckFailingStepID  // "check_skip_if_check_failing"
constants.RemoveTriggerLabelStepID       // "remove_trigger_label"
constants.GetTriggerLabelStepID          // "get_trigger_label"
constants.PreActivationAppTokenStepID    // "pre-activation-app-token"

// Agent job step IDs
constants.ParseMCPGatewayStepID          // "parse-mcp-gateway"
```

### Step Output Keys

```go
constants.IsTeamMemberOutput          // "is_team_member"
constants.ActivatedOutput             // "activated"
constants.MatchedCommandOutput        // "matched_command"
constants.StopTimeOkOutput            // "stop_time_ok"
constants.SkipCheckOkOutput           // "skip_check_ok"
constants.SkipNoMatchCheckOkOutput    // "skip_no_match_check_ok"
constants.CommandPositionOkOutput     // "command_position_ok"
constants.RateLimitOkOutput           // "rate_limit_ok"
constants.SkipRolesOkOutput           // "skip_roles_ok"
constants.SkipBotsOkOutput            // "skip_bots_ok"
constants.SkipIfCheckFailingOkOutput  // "skip_if_check_failing_ok"
```

### MCP Server IDs

```go
constants.SafeOutputsMCPServerID       // "safeoutputs"
constants.MCPScriptsMCPServerID        // "mcpscripts"
constants.MCPScriptsMCPVersion         // "1.0.0"
constants.AgenticWorkflowsMCPServerID  // "agenticworkflows"
```

## Version Constants

### Default Versions (pinned dependencies)

```go
// AI engine CLIs
constants.DefaultCopilotVersion         // Copilot CLI version (e.g. "1.0.31")
constants.DefaultClaudeCodeVersion      // Claude Code CLI version
constants.DefaultCodexVersion           // OpenAI Codex CLI version
constants.DefaultGeminiVersion          // Google Gemini CLI version
constants.DefaultCrushVersion           // Crush CLI version

// Infrastructure
constants.DefaultGitHubMCPServerVersion // GitHub MCP server Docker image version
constants.DefaultFirewallVersion        // AWF firewall version
constants.DefaultMCPGatewayVersion      // MCP Gateway (gh-aw-mcpg) Docker image version

// MCP tooling
constants.DefaultPlaywrightMCPVersion   // @playwright/mcp npm package version
constants.DefaultPlaywrightBrowserVersion // Playwright browser Docker image version
constants.DefaultMCPSDKVersion          // @modelcontextprotocol/sdk npm package version
constants.DefaultGitHubScriptVersion    // actions/github-script action version

// Runtime setup versions
constants.DefaultNodeVersion            // Node.js (e.g. "24")
constants.DefaultPythonVersion          // Python (e.g. "3.12")
constants.DefaultGoVersion              // Go (e.g. "1.25")
constants.DefaultBunVersion             // Bun
constants.DefaultRubyVersion            // Ruby
constants.DefaultDotNetVersion          // .NET
constants.DefaultJavaVersion            // Java (JDK)
constants.DefaultElixirVersion          // Elixir
constants.DefaultHaskellVersion         // GHC (Haskell)
constants.DefaultDenoVersion            // Deno
```

### Minimum Version Constraints

These constants guard feature flag emission: the compiler MUST NOT emit certain flags unless the pinned version is at or above the minimum.

```go
constants.AWFExcludeEnvMinVersion       // "v0.25.3"  — minimum AWF for --exclude-env
constants.AWFCliProxyMinVersion         // "v0.25.17" — minimum AWF for CLI proxy flags
constants.CopilotNoAskUserMinVersion    // "1.0.19"   — minimum Copilot CLI for --no-ask-user
constants.MCPGIntegrityReactionsMinVersion // "v0.2.18" — minimum MCPG for integrity-reactions policy
```

## URL Constants

```go
// Registry and host URLs
constants.DefaultMCPRegistryURL     // "https://api.mcp.github.com/v0.1"
constants.PublicGitHubHost          // "https://github.com"
constants.GitHubCopilotMCPDomain    // "api.githubcopilot.com" (remote MCP mode)

// Documentation URLs (type DocURL)
constants.DocsEnginesURL      // engines reference documentation
constants.DocsToolsURL        // tools and MCP server configuration
constants.DocsGitHubToolsURL  // GitHub tools configuration
constants.DocsPermissionsURL  // GitHub permissions configuration
constants.DocsNetworkURL      // network configuration
constants.DocsSandboxURL      // sandbox configuration
```

## Formatting Constants

```go
constants.MaxExpressionLineLength    // 120 — maximum line length for YAML expressions
constants.ExpressionBreakThreshold   // 100 — threshold at which long lines get broken
constants.CLIExtensionPrefix         // "gh aw" — user-facing CLI prefix
```

## Runtime Configuration

```go
// Paths (GitHub Actions expression form vs shell form)
constants.GhAwRootDir                // "${{ runner.temp }}/gh-aw" (use in with:/env: YAML)
constants.GhAwRootDirShell           // "${RUNNER_TEMP}/gh-aw"     (use inside run: blocks)

// Timeouts
constants.DefaultAgenticWorkflowTimeout // 20 * time.Minute
constants.DefaultToolTimeout            // 60 * time.Second
constants.DefaultMCPStartupTimeout      // 120 * time.Second

// Rate limits
constants.DefaultRateLimitMax    // 5  — max runs per window
constants.DefaultRateLimitWindow // 60 — window in minutes

// Runner image
constants.DefaultActivationJobRunnerImage // "ubuntu-slim"

// Symlink resolution
constants.MaxSymlinkDepth         // 5 — max recursive symlink depth for remote file fetching

// Memory file extension allowlist (empty = all extensions allowed)
constants.DefaultAllowedMemoryExtensions // []string{}

// GetWorkflowDir returns ".github/workflows" (or override from GH_AW_WORKFLOWS_DIR env var)
dir := constants.GetWorkflowDir()
```

## Container Images and Mounts

### Images

```go
constants.DefaultNodeAlpineLTSImage     // "node:lts-alpine"
constants.DefaultPythonAlpineLTSImage   // "python:alpine"
constants.DefaultAlpineImage            // "alpine:latest"
constants.DevModeGhAwImage              // "localhost/gh-aw:dev" (local dev only)
constants.DefaultMCPGatewayContainer    // "ghcr.io/github/gh-aw-mcpg"
constants.DefaultFirewallRegistry       // "ghcr.io/github/gh-aw-firewall"
```

### Docker Volume Mounts

These strings are passed to Docker's `--volume` / `-v` flag in containerized MCP server steps:

```go
constants.DefaultGhAwMount        // "${RUNNER_TEMP}/gh-aw:${RUNNER_TEMP}/gh-aw:ro"
constants.DefaultGhBinaryMount    // "/usr/bin/gh:/usr/bin/gh:ro"
constants.DefaultTmpGhAwMount     // "/tmp/gh-aw:/tmp/gh-aw:rw"
constants.DefaultWorkspaceMount   // "\${GITHUB_WORKSPACE}:\${GITHUB_WORKSPACE}:rw"
```

### MCP Gateway

```go
constants.DefaultMCPGatewayPayloadDir              // "/tmp/gh-aw/mcp-payloads"
constants.DefaultMCPGatewayPayloadSizeThreshold    // 524288 (512 KB)
```

## AWF (Agentic Workflow Firewall) Constants

```go
constants.AWFDefaultCommand          // "sudo -E awf"
constants.AWFProxyLogsDir            // "/tmp/gh-aw/sandbox/firewall/logs"
constants.AWFAuditDir                // "/tmp/gh-aw/sandbox/firewall/audit"
constants.AWFDefaultLogLevel         // "info"
constants.DefaultGitHubLockdown      // false — GitHub MCP server lockdown default
```

## Validation Field Lists

These variables control YAML key ordering and validation during workflow compilation:

```go
// Conventional field order for GitHub Actions primitives
constants.PriorityStepFields      // []string{"name","id","if","run","uses","script","env","with"}
constants.PriorityJobFields       // []string{"name","runs-on","needs","if","permissions",...}
constants.PriorityWorkflowFields  // []string{"on","permissions","if","network","imports",...}

// Fields silently ignored during frontmatter validation
constants.IgnoredFrontmatterFields // []string{"user-invokable"}

// Fields forbidden in shared/imported workflows (only valid in main workflows)
constants.SharedWorkflowForbiddenFields // []string{"on","command","concurrency",...}

// Events that do not require permission checks
constants.SafeWorkflowEvents      // []string{"workflow_dispatch","schedule"}

// Domains always allowed for Playwright browser automation
constants.DefaultAllowedDomains   // []string{"localhost","localhost:*","127.0.0.1","127.0.0.1:*"}
```

## Network Port Constants

```go
constants.DefaultMCPGatewayPort     // 80    — MCP gateway HTTP service
constants.DefaultMCPServerPort      // 3000  — mcp-scripts MCP server
constants.DefaultMCPInspectorPort   // 3001  — safe-outputs MCP inspector
constants.MinNetworkPort            // 1
constants.MaxNetworkPort            // 65535
constants.ClaudeLLMGatewayPort      // 10000
constants.CodexLLMGatewayPort       // 10001
constants.CopilotLLMGatewayPort     // 10002
constants.GeminiLLMGatewayPort      // 10003
constants.CrushLLMGatewayPort       // 10004
```

## Tool Lists

```go
// GitHub API expressions allowed in workflow markdown
constants.AllowedExpressions      // []string of allowed ${{ github.* }} expression paths
constants.AllowedExpressionsSet   // map[string]struct{} for O(1) lookup

// Property names blocked in expressions (XSS/injection prevention)
constants.DangerousPropertyNames
constants.DangerousPropertyNamesSet

// Default GitHub tool lists used when no explicit tools: configuration is present
constants.DefaultReadOnlyGitHubTools
constants.DefaultGitHubTools
constants.DefaultBashTools
```

## Design Notes

- All semantic types implement `String()` and `IsValid()` to allow consistent validation across the codebase.
- Version constants are intentionally plain string literals (not derived from build tags or embedded files) so that individual upgrades can be made as targeted one-line changes.
- `GetWorkflowDir()` reads `GH_AW_WORKFLOWS_DIR` from the environment at call time, allowing the directory to be overridden in tests and CI.
- `AgenticEngines` is deprecated in favour of `workflow.NewEngineCatalog(workflow.NewEngineRegistry()).IDs()` but is kept for backward compatibility.

---

*This specification is automatically maintained by the [spec-extractor](../../.github/workflows/spec-extractor.md) workflow.*
