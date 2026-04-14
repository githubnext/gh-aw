# constants Package

The `constants` package provides shared semantic type aliases and named constants used across multiple `gh-aw` packages. Centralizing these values ensures consistency and type safety throughout the codebase.

## Overview

The package is organized into focused files:

| File | Contents |
|------|----------|
| `constants.go` | Core types, formatting constants, runtime config, container images |
| `engine_constants.go` | AI engine names, options, system secrets, Copilot CLI commands |
| `feature_constants.go` | Feature flag identifiers |
| `job_constants.go` | GitHub Actions job names, step IDs, artifact names, output keys |
| `tool_constants.go` | Allowed GitHub tool expressions and default tool lists |
| `url_constants.go` | URL semantic types and well-known URL constants |
| `version_constants.go` | Default version strings for all pinned dependencies |

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
| `DocURL` | Documentation URL | — |

All semantic types implement `String() string` and `IsValid() bool` methods.

## Engine Constants

```go
import "github.com/github/gh-aw/pkg/constants"

// Engine names
constants.CopilotEngine  // "copilot"
constants.ClaudeEngine   // "claude"
constants.CodexEngine    // "codex"
constants.GeminiEngine   // "gemini"
constants.DefaultEngine  // "copilot"

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
- `KeyURL` — URL where users can obtain their API key
- `WhenNeeded` — human-readable description of when this secret is needed

## Feature Flags

```go
constants.MCPScriptsFeatureFlag           // "mcp-scripts"
constants.MCPGatewayFeatureFlag           // "mcp-gateway"
constants.DisableXPIAPromptFeatureFlag    // "disable-xpia-prompt"
constants.CopilotRequestsFeatureFlag      // "copilot-requests"
constants.CliProxyFeatureFlag             // "cli-proxy"
constants.IntegrityReactionsFeatureFlag   // "integrity-reactions"
```

## Job and Step Constants

```go
// Job names
constants.AgentJobName          // "agent"
constants.ActivationJobName     // "activation"
constants.PreActivationJobName  // "pre_activation"
constants.DetectionJobName      // "detection"
constants.SafeOutputsJobName    // "safe_outputs"
constants.ConclusionJobName     // "conclusion"

// Artifact names
constants.SafeOutputArtifactName    // "safe-output"
constants.AgentOutputArtifactName   // "agent-output"
constants.ActivationArtifactName    // "activation"

// Step IDs
constants.CheckMembershipStepID     // "check_membership"
constants.CheckRateLimitStepID      // "check_rate_limit"
constants.CheckMembershipStepID     // "check_membership"

// Step output keys
constants.IsTeamMemberOutput        // "is_team_member"
constants.ActivatedOutput           // "activated"
constants.MatchedCommandOutput      // "matched_command"
```

## Version Constants

All pinned dependency versions are defined here:

```go
constants.DefaultCopilotVersion         // Copilot CLI version
constants.DefaultClaudeCodeVersion      // Claude Code version
constants.DefaultCodexVersion           // Codex version
constants.DefaultGeminiVersion          // Gemini CLI version
constants.DefaultGitHubMCPServerVersion // GitHub MCP server version
constants.DefaultFirewallVersion        // AWF firewall version
constants.DefaultNodeVersion            // Node.js runtime version
constants.DefaultPythonVersion          // Python runtime version
constants.DefaultGoVersion              // Go runtime version
```

## Formatting Constants

```go
constants.MaxExpressionLineLength    // 120 — maximum line length for YAML expressions
constants.ExpressionBreakThreshold   // 100 — threshold at which long lines get broken
```

## Runtime Configuration

```go
constants.GhAwRootDir                // "${{ runner.temp }}/gh-aw"
constants.GhAwRootDirShell           // "${RUNNER_TEMP}/gh-aw"
constants.DefaultAgenticWorkflowTimeout // 20 minutes
constants.DefaultToolTimeout         // 60 seconds
constants.DefaultMCPStartupTimeout   // 120 seconds
constants.DefaultRateLimitMax        // 5 runs per window
constants.DefaultRateLimitWindow     // 60 minutes

// GetWorkflowDir returns the workflows directory respecting GH_AW_WORKFLOWS_DIR env var
dir := constants.GetWorkflowDir()
```

## Container Images

```go
constants.DefaultNodeAlpineLTSImage      // "node:lts-alpine"
constants.DefaultPythonAlpineLTSImage    // "python:alpine"
constants.DefaultAlpineImage             // "alpine:latest"
constants.DefaultMCPGatewayContainer     // ghcr.io/github/gh-aw-mcpg
constants.DefaultFirewallRegistry        // ghcr.io/github/gh-aw-firewall
```

## Tool Lists

```go
// GitHub API tools allowed in workflow expressions
constants.AllowedExpressions      // []string of allowed GitHub tool names
constants.AllowedExpressionsSet   // map[string]struct{} for O(1) lookup

// Dangerous property names (blocked in expressions)
constants.DangerousPropertyNames
constants.DangerousPropertyNamesSet

// Default tools for read-only GitHub operations
constants.DefaultReadOnlyGitHubTools
constants.DefaultGitHubTools
constants.DefaultBashTools
```

## Design Notes

- All semantic types implement `String()` and `IsValid()` to allow consistent validation across the codebase.
- Version constants are intentionally plain string literals (not derived from build tags or embedded files) so that individual upgrades can be made as targeted one-line changes.
- `GetWorkflowDir()` reads `GH_AW_WORKFLOWS_DIR` from the environment at call time, allowing the directory to be overridden in tests and CI.
