# GitHub Actions Workflow Layout Specification

> Auto-generated specification documenting patterns used in compiled `.lock.yml` files.
> Last updated: 2025-12-29

## Overview

This document catalogs all file paths, folder names, artifact names, job patterns, and other structural elements used across our compiled GitHub Actions workflows (`.lock.yml` files). This specification serves as a reference for developers working on workflow compilation, safe outputs, and GitHub Actions integration.

**Scope**: This document covers 128 compiled `.lock.yml` files in `.github/workflows/`.

## GitHub Actions

Common GitHub Actions used across workflows, pinned to specific commit SHAs for security:

| Action | SHA | Usage Count | Description | Context |
|--------|-----|-------------|-------------|---------|
| actions/github-script | ed597411d8f9... | 2,436 | Runs GitHub API scripts inline | Primary tool for GitHub API interactions, safe outputs, detection logic |
| actions/upload-artifact | 330a01c490ac... | 1,263 | Uploads build artifacts | Used for agent outputs, patches, prompts, logs, and detection results |
| actions/checkout | 93cb6efe1820... | 956 | Checks out repository code | Standard initial step for accessing repository content |
| actions/download-artifact | 018cc2cf5baa... | 758 | Downloads artifacts from previous jobs | Used in safe-output jobs and conclusion jobs to access agent outputs |
| ./actions/setup | (local) | 754 | Custom setup action | Prepares workflow environment, installs engines, configures MCP servers |
| actions/setup-node | 395ad3262231... | 93 | Sets up Node.js environment | Used when workflows require npm/node packages |
| actions/cache/save | 0057852bfaa8... | 60 | Saves cache entries | Caches dependencies, build artifacts, MCP server state |
| actions/cache/restore | 0057852bfaa8... | 60 | Restores cache entries | Restores previously cached data |
| actions/setup-go | 4dc6199c7b1a... | 31 | Sets up Go environment | Used for Go-based tools and compilation |
| actions/setup-python | a26af69be951... | 19 | Sets up Python environment | Used for Python runtime and pip packages |
| astral-sh/setup-uv | d4b2f3b6ecc6... | 17 | Sets up UV package manager | Fast Python package installer |
| actions/create-github-app-token | 29824e69f546... | 7 | Creates GitHub App token | For elevated permissions in specific workflows |
| githubnext/gh-aw/actions/setup | 523f6cfa6283... | 5 | Pinned version of setup action | Used when specific version is required |
| actions/cache | 0057852bfaa8... | 3 | Cache action (combined) | Combined save/restore cache action |
| anchore/sbom-action | fbfd9c6c1892... | 2 | Generates SBOM artifacts | Software Bill of Materials generation |
| actions/ai-inference | 334892bb2038... | 2 | AI inference action | Direct AI model inference |
| super-linter/super-linter | 2bdd90ed3262... | 1 | Lints code across multiple languages | Code quality checks |
| github/stale-repos | a21e55567b83... | 1 | Identifies stale repositories | Repository health monitoring |
| cli/gh-extension-precompile | 9e2237c30f86... | 1 | Precompiles gh extensions | For gh CLI extension development |

## Artifact Names

Artifacts uploaded/downloaded between workflow jobs. These enable data sharing across jobs in the workflow DAG:

### Core Workflow Artifacts

| Name | Type | Description | Producer | Consumer | Context |
|------|------|-------------|----------|----------|---------|
| agent_output.json | JSON | AI agent execution output | agent job | detection, safe_outputs, conclusion | Contains agent's response, analysis, and tool calls |
| safe_output.jsonl | JSONL | Safe outputs configuration | detection job | safe_outputs jobs | Structured directives for GitHub API operations |
| aw.patch | Git patch | Git diff of agent changes | agent job | create_pull_request | Used by PR creation to apply changes |
| prompt.txt | Text | Agent prompt content | agent job | conclusion | Stored for debugging and audit purposes |
| aw_info.json | JSON | Workflow metadata | setup step | multiple | Contains workflow run context and configuration |
| cache-memory | JSON | Persistent workflow memory | agent/detection | agent (next run) | Agent's long-term memory across runs |
| cache-memory-focus-areas | JSON | Focused memory areas | agent/detection | agent (next run) | Prioritized memory segments |
| repo-memory-default | JSON | Repository-specific memory | agent | agent (next run) | Repository context and patterns |

### Logging and Debugging Artifacts

| Name | Type | Description | Context |
|------|------|-------------|---------|
| mcp-logs | Logs | MCP server logs | Debug logs from Model Context Protocol servers |
| agent-stdio.log | Log | Agent standard I/O | Captures agent process output |
| playwright-debug-logs-${{ github.run_id }} | Logs | Browser automation logs | Playwright MCP server debugging |
| super-linter-log | Log | Linter execution log | Code quality analysis output |
| threat-detection.log | Log | Security threat detection | Firewall and security scan results |

### Firewall Logs (per workflow)

| Name Pattern | Description | Context |
|--------------|-------------|---------|
| firewall-logs-{workflow-name} | Firewall logs for specific workflow | Each workflow gets its own firewall log artifact (128 unique patterns) |

Examples:
- `firewall-logs-dev-hawk`
- `firewall-logs-daily-workflow-updater`
- `firewall-logs-copilot-pr-nlp-analysis`

### Data and Visualization Artifacts

| Name | Type | Description | Context |
|------|------|-------------|---------|
| data-charts | Images/Charts | Generated visualization charts | Data analysis workflows |
| trending-charts | Images | Trending data visualizations | Metrics and analytics |
| trending-source-and-data | CSV/JSON | Source data for trending | Data pipeline outputs |
| python-source-and-data | Python/Data | Python analysis outputs | Python-based workflows |

### Safe Output Assets

| Name | Type | Description | Context |
|------|------|-------------|---------|
| safe-outputs-assets | Mixed | Assets for safe output operations | Files to upload with issues/PRs/discussions |
| safeinputs | JSON | Safe inputs configuration | Input validation and sanitization |

### SBOM Artifacts

| Name | Format | Description | Context |
|------|--------|-------------|---------|
| sbom-artifacts | SBOM | Software Bill of Materials | Security and compliance tracking |

## Common Job Names

Standard job names across workflows. These follow a consistent naming pattern:

### Core Workflow Jobs

| Job Name | Description | Runs After | Context |
|----------|-------------|------------|---------|
| activation | Determines if workflow should run | (first) | Uses skip-if-match, team membership, time windows |
| pre_activation | Pre-flight checks before activation | (first) | Enhanced activation with multiple conditions |
| agent | Main AI agent execution job | activation | Runs the copilot/claude/codex engine |
| detection | Post-agent analysis job | agent | Analyzes agent output for safe-output patterns |
| conclusion | Final status reporting job | all jobs | Runs after all other jobs complete (always) |
| safe_outputs | Safe outputs orchestration | detection | Coordinates all safe-output operations |

### Safe Output Jobs

| Job Name | Description | Trigger | Context |
|----------|-------------|---------|---------|
| post-issue | Creates or updates GitHub issues | safe_outputs | Uses create_issue safe-output |
| post_to_slack_channel | Posts message to Slack | safe_outputs | Slack integration |
| notion_add_comment | Adds comment to Notion | safe_outputs | Notion integration |
| push_repo_memory | Updates repository memory | agent | Stores agent learnings |
| update_cache_memory | Updates cache memory | agent | Persists workflow memory |
| upload_assets | Uploads assets to releases | safe_outputs | Asset management |

### Utility Jobs

| Job Name | Description | Context |
|----------|-------------|---------|
| check_ci_status | Checks CI status before proceeding | Pre-condition checking |
| check_external_user | Validates user permissions | Security validation |
| search_issues | Searches for related issues | Issue management |
| test_environment | Tests workflow environment | Testing and validation |
| ast_grep | AST-based code search | Code analysis |
| super_linter | Runs super-linter | Code quality |
| generate-sbom | Generates Software Bill of Materials | Security compliance |
| release | Release automation | Publishing |

## File Paths

Common file paths referenced in workflows. Organized by category:

### Temporary Working Directories

| Path | Description | Usage |
|------|-------------|-------|
| /tmp/gh-aw/ | Root temporary directory | Base directory for all workflow temporary files |
| /tmp/gh-aw/agent-stdio.log | Agent standard I/O log | Captures agent process output |
| /tmp/gh-aw/aw-prompts/prompt.txt | Agent prompt file | Stores generated prompt for agent |
| /tmp/gh-aw/aw.patch | Git patch file | Agent-generated changes |
| /tmp/gh-aw/aw_info.json | Workflow info JSON | Metadata about current workflow run |
| /tmp/gh-aw/cache-memory | Cache memory file | Persistent agent memory |
| /tmp/gh-aw/cache-memory-focus-areas | Focused memory areas | Priority memory segments |
| /tmp/gh-aw/layout-cache | Layout specification cache | Cached layout patterns |
| /tmp/gh-aw/prompt-cache | Prompt cache | Cached prompt templates |
| /tmp/gh-aw/redacted-urls.log | Redacted URL log | Security audit log |
| /tmp/gh-aw/repo-memory/default | Repository memory | Repo-specific context |

### MCP and Logging Directories

| Path | Description | Usage |
|------|-------------|-------|
| /tmp/gh-aw/mcp-config/logs/ | MCP configuration logs | MCP server configuration debugging |
| /tmp/gh-aw/mcp-logs/ | MCP server logs | MCP server operation logs |
| /tmp/gh-aw/sandbox/agent/logs/ | Sandboxed agent logs | Agent execution in sandbox mode |
| /tmp/gh-aw/sandbox/firewall/logs/ | Firewall logs | Network firewall logs |
| /tmp/gh-aw/safe-inputs/logs/ | Safe inputs logs | Input validation logs |

### Safe Outputs Directories

| Path | Description | Usage |
|------|-------------|-------|
| /tmp/gh-aw/safe-jobs/ | Safe job definitions | Generated safe-output job configs |
| /tmp/gh-aw/safeoutputs/ | Safe outputs working directory | Safe-output processing |
| /tmp/gh-aw/safeoutputs/assets/ | Safe output assets | Files to attach to issues/PRs |

### Data and Analysis Directories

| Path | Description | Usage |
|------|-------------|-------|
| /tmp/gh-aw/python/*.py | Python source files | Generated Python analysis scripts |
| /tmp/gh-aw/python/charts/*.png | Chart images | Generated data visualizations |
| /tmp/gh-aw/python/data/* | Data files | Analysis data outputs |
| /tmp/gh-aw/playwright-debug-logs/ | Playwright logs | Browser automation debugging |
| /tmp/gh-aw/threat-detection/ | Threat detection files | Security analysis outputs |
| /tmp/gh-aw/threat-detection/detection.log | Detection log | Security detection results |

### SBOM Artifacts

| Path | Description | Usage |
|------|-------------|-------|
| sbom.cdx.json | CycloneDX SBOM | SBOM in CycloneDX format |
| sbom.spdx.json | SPDX SBOM | SBOM in SPDX format |
| super-linter.log | Super-linter output | Linter results |

### Repository Structure

| Path | Description | Usage |
|------|-------------|-------|
| .github/workflows/ | Workflow definition directory | Contains all .md and .lock.yml files |
| .github/workflows/shared/ | Shared workflow components | Reusable workflow imports |
| .github/aw/ | Agentic workflow configuration | Contains actions-lock.json and configs |
| .github/agents/ | Agent definitions | Custom agent markdown files |

### Source Code Directories

| Path | Description | Usage |
|------|-------------|-------|
| pkg/workflow/ | Workflow compilation code | Go package for compiling workflows |
| pkg/workflow/js/ | JavaScript runtime code | CommonJS modules for GitHub Actions |
| pkg/cli/ | CLI command implementations | gh-aw command handlers |
| pkg/parser/ | Markdown frontmatter parsing | Schema validation and parsing |
| pkg/constants/ | Constants definitions | Version numbers, defaults, limits |
| actions/setup/ | Custom setup action | Workflow environment preparation |
| actions/setup/js/ | Setup action JavaScript | Action implementation scripts |
| actions/setup/sh/ | Setup action shell scripts | Bash scripts for setup |
| specs/ | Specification documents | Documentation and specs directory |
| docs/ | Documentation site | Astro Starlight documentation |

## Working Directories

Working directories used in workflow steps:

| Directory | Context | Usage |
|-----------|---------|-------|
| ./actions/setup/js | JavaScript action development | Building and testing setup action |
| ./docs | Documentation site | Building documentation with Astro |

## Environment Variables

Key environment variables used across workflows:

### Core Workflow Variables

| Variable | Type | Description | Set By |
|----------|------|-------------|--------|
| GH_AW_AGENT_OUTPUT | Path | Path to agent output JSON | setup action |
| GH_AW_SAFE_OUTPUTS | Path | Path to safe outputs config | detection job |
| GITHUB_TOKEN | Secret | GitHub API authentication | GitHub Actions |
| GITHUB_WORKSPACE | Path | Workspace directory | GitHub Actions |

### Model Configuration Variables

| Variable | Description | Default |
|----------|-------------|---------|
| GH_AW_MODEL_AGENT_COPILOT | Copilot model for agent execution | (engine default) |
| GH_AW_MODEL_AGENT_CLAUDE | Claude model for agent execution | (engine default) |
| GH_AW_MODEL_AGENT_CODEX | Codex model for agent execution | (engine default) |
| GH_AW_MODEL_DETECTION_COPILOT | Copilot model for detection | gpt-5-mini |
| GH_AW_MODEL_DETECTION_CLAUDE | Claude model for detection | (engine default) |
| GH_AW_MODEL_DETECTION_CODEX | Codex model for detection | (engine default) |

## Constants and Patterns

Patterns found in Go source code (`pkg/constants/constants.go`):

### Path Constants

| Constant | Value | Description |
|----------|-------|-------------|
| ScriptsBasePath | /tmp/gh-aw/scripts | Base path for generated scripts |
| SetupActionDestination | /tmp/gh-aw/actions | Destination for setup action files |
| RedactedURLsLogPath | /tmp/gh-aw/redacted-urls.log | Path to redacted URLs log |
| SafeInputsDirectory | /tmp/gh-aw/safe-inputs | Directory for safe inputs |

### Artifact Name Constants

| Constant | Value | Description |
|----------|-------|-------------|
| SafeOutputArtifactName | safe_output.jsonl | Safe outputs artifact name |
| AgentOutputArtifactName | agent_output.json | Agent output artifact name |

### Job Name Constants

| Constant | Type | Value | Description |
|----------|------|-------|-------------|
| AgentJobName | JobName | agent | Main agent execution job |
| ActivationJobName | JobName | activation | Workflow activation job |
| PreActivationJobName | JobName | pre_activation | Pre-activation checks job |
| DetectionJobName | JobName | detection | Safe-output detection job |

### Step ID Constants

| Constant | Type | Value | Description |
|----------|------|-------|-------------|
| CheckMembershipStepID | StepID | check_membership | Team membership check |
| CheckStopTimeStepID | StepID | check_stop_time | Time window validation |
| CheckSkipIfMatchStepID | StepID | check_skip_if_match | Pattern-based skip |
| CheckCommandPositionStepID | StepID | check_command_position | Command position check |

### Output Name Constants

| Constant | Value | Description |
|----------|-------|-------------|
| IsTeamMemberOutput | is_team_member | Team membership result |
| StopTimeOkOutput | stop_time_ok | Time window validation result |
| SkipCheckOkOutput | skip_check_ok | Skip check result |
| CommandPositionOkOutput | command_position_ok | Command position result |
| ActivatedOutput | activated | Final activation decision |

### MCP Server IDs

| Constant | Value | Description |
|----------|-------|-------------|
| SafeOutputsMCPServerID | safeoutputs | Safe outputs MCP server identifier |
| SafeInputsMCPServerID | safeinputs | Safe inputs MCP server identifier |

### Version Constants

| Constant | Type | Default Value | Description |
|----------|------|---------------|-------------|
| DefaultCopilotVersion | Version | 0.0.372 | GitHub Copilot CLI version |
| DefaultClaudeCodeVersion | Version | 2.0.76 | Claude Code CLI version |
| DefaultCodexVersion | Version | 0.77.0 | OpenAI Codex CLI version |
| DefaultGitHubMCPServerVersion | Version | v0.26.3 | GitHub MCP server Docker image |
| DefaultFirewallVersion | Version | v0.7.0 | gh-aw-firewall (AWF) binary |
| DefaultPlaywrightMCPVersion | Version | 0.0.54 | @playwright/mcp package |
| DefaultPlaywrightBrowserVersion | Version | v1.57.0 | Playwright browser Docker image |
| DefaultMCPSDKVersion | Version | 1.24.0 | @modelcontextprotocol/sdk package |
| DefaultGitHubScriptVersion | Version | v8 | actions/github-script action |
| DefaultBunVersion | Version | 1.1 | Bun runtime version |
| DefaultNodeVersion | Version | 24 | Node.js runtime version |
| DefaultPythonVersion | Version | 3.12 | Python runtime version |
| DefaultRubyVersion | Version | 3.3 | Ruby runtime version |
| DefaultDotNetVersion | Version | 8.0 | .NET runtime version |
| DefaultJavaVersion | Version | 21 | Java runtime version |
| DefaultElixirVersion | Version | 1.17 | Elixir runtime version |
| DefaultGoVersion | Version | 1.25 | Go runtime version |
| DefaultHaskellVersion | Version | 9.10 | GHC runtime version |
| DefaultDenoVersion | Version | 2.x | Deno runtime version |
| SafeInputsMCPVersion | Version | 1.0.0 | Safe inputs MCP server version |

### Timeout Constants

| Constant | Value | Description |
|----------|-------|-------------|
| DefaultAgenticWorkflowTimeout | 20 minutes | Agentic workflow execution timeout |
| DefaultToolTimeout | 60 seconds | Tool/MCP server operation timeout |
| DefaultMCPStartupTimeout | 120 seconds | MCP server startup timeout |

### Feature Flags

| Constant | Value | Description |
|----------|-------|-------------|
| SafeInputsFeatureFlag | safe-inputs | Safe inputs feature flag |
| MCPGatewayFeatureFlag | mcp-gateway | MCP gateway feature flag |
| SandboxRuntimeFeatureFlag | sandbox-runtime | Sandbox runtime feature flag |

### Size Limits

| Constant | Type | Value | Description |
|----------|------|-------|-------------|
| MaxExpressionLineLength | LineLength | 120 | Max line length for expressions |
| ExpressionBreakThreshold | LineLength | 100 | Threshold for line breaking |

### Default Tool Lists

| Constant | Description | Tool Count |
|----------|-------------|------------|
| DefaultReadOnlyGitHubTools | Read-only GitHub MCP tools | 58 |
| DefaultGitHubToolsLocal | Local (Docker) mode tools | 58 |
| DefaultGitHubToolsRemote | Remote (hosted) mode tools | 58 |
| DefaultBashTools | Basic bash commands | 12 |
| AgenticEngines | Supported AI engines | 3 (claude, codex, copilot) |

## Field Ordering Patterns

GitHub Actions YAML fields are ordered conventionally for readability:

### Step Fields Priority Order

1. name
2. id
3. if
4. run
5. uses
6. script
7. env
8. with
9. (remaining fields alphabetically)

### Job Fields Priority Order

1. name
2. runs-on
3. needs
4. if
5. permissions
6. environment
7. concurrency
8. outputs
9. env
10. steps
11. (remaining fields alphabetically)

### Workflow Fields Priority Order

1. on
2. permissions
3. if
4. network
5. imports
6. safe-outputs
7. steps
8. (remaining fields alphabetically)

## Usage Guidelines

### Artifact Naming

- Use descriptive hyphenated names (e.g., `agent-output`, `mcp-logs`)
- Include workflow context in artifact names for debugging (e.g., `firewall-logs-{workflow-name}`)
- Use `.jsonl` extension for line-delimited JSON
- Use `.json` extension for single JSON objects

### Job Naming

- Use snake_case for job names (e.g., `create_pull_request`, `safe_outputs`)
- Use descriptive names that indicate job purpose
- Prefix safe-output jobs with action type (e.g., `post-issue`, `notion_add_comment`)

### Path References

- Use relative paths from repository root
- Use absolute paths in /tmp/gh-aw/ for temporary files
- Never use /tmp/ directly - always use /tmp/gh-aw/ subdirectory
- Organize temporary files by purpose (e.g., /tmp/gh-aw/mcp-logs/, /tmp/gh-aw/safe-inputs/)

### Action Pinning

- Always pin actions to full commit SHA for security
- Pin to specific versions for third-party actions
- Use local actions (./actions/setup) for custom functionality
- Update pins regularly but test thoroughly

### Environment Variables

- Prefix custom variables with GH_AW_ for namespacing
- Use SCREAMING_SNAKE_CASE for environment variables
- Document all custom environment variables
- Use GitHub Actions built-in variables when available

### GitHub Actions Expressions

- Keep expressions under 120 characters per line
- Break complex expressions at logical operators
- Use allowed expressions from constants.AllowedExpressions
- Validate expressions against GitHub Actions schema

## Working with This Specification

### When Adding New Patterns

1. Add the pattern to the appropriate section
2. Include description and context
3. Document producer/consumer relationships for artifacts
4. Update version constants when bumping dependencies
5. Follow existing naming conventions

### When Modifying Paths

1. Check all references in Go code
2. Update path constants if needed
3. Test workflow compilation
4. Update this document

### When Adding New Jobs

1. Add to Common Job Names section
2. Document dependencies (needs)
3. Specify trigger conditions
4. Add to field ordering if special rules apply

## Related Documentation

- **Workflow Compilation**: `pkg/workflow/compiler.go`
- **Safe Outputs**: `pkg/workflow/safe_outputs.go`
- **Constants**: `pkg/constants/constants.go`
- **Actions Build**: `pkg/cli/actions_build_command.go`
- **Schema Validation**: `pkg/parser/schemas/`

## Statistics

- **Lock files analyzed**: 128
- **GitHub Actions cataloged**: 19 unique actions
- **Artifacts documented**: 140+ (including 128 firewall logs)
- **Job patterns found**: 20+ common patterns
- **File paths listed**: 50+
- **Constants extracted**: 70+

---

*This document is automatically maintained by the Layout Specification Maintainer workflow.*
*Last updated: 2025-12-29*
*Source: Extracted from .github/workflows/*.lock.yml and pkg/ source code*
