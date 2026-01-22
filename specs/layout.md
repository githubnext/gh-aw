# GitHub Actions Workflow Layout Specification

> Auto-generated specification documenting patterns used in compiled `.lock.yml` files.
> Last updated: 2026-01-05

## Overview

This document catalogs all file paths, folder names, artifact names, and other patterns used across our compiled GitHub Actions workflows (`.lock.yml` files). This specification is derived from analyzing **127 workflow lock files** in `.github/workflows/`, along with Go source code in `pkg/workflow/` and JavaScript code in `actions/setup/js/`.

## GitHub Actions

Common GitHub Actions used across workflows, pinned to specific commit SHAs for security:

| Action | SHA (abbreviated) | Description | Context |
|--------|-------------------|-------------|---------|
| `actions/checkout` | `93cb6efe` | Checks out repository code | Used in almost all workflows for accessing repo content |
| `actions/upload-artifact` | `330a01c4` | Uploads build artifacts | Used for agent outputs, patches, prompts, and logs |
| `actions/download-artifact` | `018cc2cf` | Downloads artifacts from previous jobs | Used in safe-output jobs and conclusion jobs |
| `actions/setup-node` | `395ad326` | Sets up Node.js environment | Used in workflows requiring npm/node |
| `actions/setup-python` | `a26af69b` | Sets up Python environment | Used for Python-based analysis and tools |
| `actions/setup-go` | `4dc61999` | Sets up Go environment | Used for Go compilation and testing |
| `actions/github-script` | `ed597411` | Runs GitHub API scripts | Used for GitHub API interactions |
| `actions/cache` | `0057852b` | Caches dependencies and artifacts | Used for performance optimization |
| `actions/cache/restore` | `0057852b` | Restores cached dependencies | Used to restore previously cached items |
| `actions/cache/save` | `0057852b` | Saves dependencies to cache | Used to save items for future runs |
| `actions/create-github-app-token` | `29824e69` | Creates GitHub App authentication token | Used for elevated API access |
| `actions/ai-inference` | `334892bb` | AI inference action | Used for AI-powered analysis |
| `astral-sh/setup-uv` | `d4b2f3b6` | Sets up uv Python package manager | Fast Python package management |
| `cli/gh-extension-precompile` | `9e2237c3` | Precompiles GitHub CLI extensions | Used in release workflows |
| `super-linter/super-linter` | `2bdd90ed` | Runs code linting across multiple languages | Code quality enforcement |
| `anchore/sbom-action` | `fbfd9c6c` | Generates Software Bill of Materials | Security and compliance |
| `github/stale-repos` | `a21e5556` | Identifies stale repositories | Repository maintenance |
| `githubnext/gh-aw/actions/setup` | `623e612f` | Sets up gh-aw environment | **Local action** - workflow setup |
| `./actions/setup` | N/A | Local setup action | Used to reference local action in repository |

## Artifact Names

### Standard Artifacts

Core artifacts uploaded/downloaded between workflow jobs:

| Artifact Name | File Inside | Type | Description | Context |
|---------------|-------------|------|-------------|---------|
| `agent-output` | `agent_output.json` | File | AI agent execution output | Contains the agent's response and analysis. Constant: `constants.AgentOutputArtifactName` |
| `safe-output` | `safe_output.jsonl` | File | Safe outputs configuration | Passed from agent to safe-output jobs. Constant: `constants.SafeOutputArtifactName` |
| `aw.patch` | `aw.patch` | File | Git patch file for changes | Used by create-pull-request safe-output |
| `prompt` | `prompt.txt` | File | Agent prompt content | Stored for debugging and audit purposes |
| `mcp-logs` | (multiple) | Directory | MCP server logs | Debug logs from Model Context Protocol servers |
| `agent-stdio.log` | `agent-stdio.log` | File | Agent standard I/O logs | Captures agent execution console output |
| `aw-info` | `aw_info.json` | File | Workflow metadata | Information about workflow execution |

**Note**: Artifact names use hyphens (upload-artifact@v5 requirement), while file names inside artifacts preserve original naming (underscores/dots for backward compatibility).

### Memory and Cache Artifacts

| Name | Type | Description | Context |
|------|------|-------------|---------|
| `cache-memory` | Directory | Default cache memory | Used when no cache ID specified |
| `cache-memory-<ID>` | Directory | Named cache memory | Used when cache.ID is specified |
| `cache-memory-focus-areas` | File | Cache memory focus areas | Specific cache for focus areas |
| `repo-memory-default` | Directory | Default repository memory | Persistent memory across workflow runs |
| `repo-memory-campaigns` | Directory | Campaign repository memory | Persistent memory for campaign workflows |
| `repo-memory-<ID>` | Directory | Custom repo memory | Named repository memory with custom ID |

### Specialized Artifacts

| Name | Type | Description | Context |
|------|------|-------------|---------|
| `firewall-logs-<workflow-name>` | Directory | Firewall logs per workflow | Security monitoring logs |
| `agent_outputs` | Directory | Multiple agent outputs | Used when agent produces multiple files |
| `safe-outputs-assets` | Directory | Safe output assets | Assets for safe-output operations |
| `safeinputs` | Directory | Safe input data | Validated input data |
| `playwright-debug-logs-${{ github.run_id }}` | Directory | Playwright browser logs | Browser automation debugging |
| `data-charts` | Directory | Generated data charts | Visualization outputs |
| `python-source-and-data` | Directory | Python source and data files | Python-based analysis inputs/outputs |
| `trending-charts` | Directory | Trending data visualizations | Chart outputs for trending analysis |
| `trending-source-and-data` | Directory | Trending analysis data | Source data for trending reports |
| `sbom-artifacts` | Directory | SBOM files | Software Bill of Materials |
| `super-linter-log` | File | Super Linter output | Linting results |
| `threat-detection.log` | File | Security threat detection log | Security scan results |

## Common Job Names

Standard job names across workflows (using snake_case convention):

| Job Name | Description | Context |
|----------|-------------|---------|
| `activation` | Determines if workflow should run | Uses skip-if-match and other filters. Constant: `ActivationJobName` |
| `pre_activation` | Pre-activation checks | Early filtering before main activation. Constant: `PreActivationJobName` |
| `agent` | Main AI agent execution job | Runs the copilot/claude/codex engine. Constant: `AgentJobName` |
| `detection` | Post-agent analysis job | Analyzes agent output for patterns. Constant: `DetectionJobName` |
| `conclusion` | Final status reporting job | Runs after all other jobs complete |
| `safe_outputs` | Safe output execution job | Executes validated safe outputs |
| `test_environment` | Environment validation | Verifies execution environment |
| `release` | Release workflow job | Package and release management |
| `super_linter` | Code linting job | Code quality checks |
| `generate-sbom` | SBOM generation | Security and compliance |
| `upload_assets` | Asset upload job | Uploads artifacts and assets |
| `update_cache_memory` | Cache memory update | Updates persistent cache |
| `push_repo_memory` | Repository memory push | Pushes repository memory |
| `post-issue` | Post-workflow issue update | Updates GitHub issues after workflow |
| `post_to_slack_channel` | Slack notification | Sends notifications to Slack |
| `notion_add_comment` | Notion integration | Updates Notion pages |
| `search_issues` | GitHub issue search | Searches for related issues |
| `check_ci_status` | CI status verification | Checks CI pipeline status |
| `check_external_user` | External contributor check | Validates external contributors |
| `ast_grep` | AST-based code search | Structural code analysis |

## File Paths

### Workflow Directories

Common file paths referenced in workflows:

| Path | Description | Context |
|------|-------------|---------|
| `.github/workflows/` | Workflow definition directory | Contains all .md and .lock.yml files |
| `.github/workflows/shared/` | Shared workflow components | Reusable workflow imports |
| `.github/workflows/shared/mcp/` | Shared MCP configurations | Reusable MCP server configs (e.g., arxiv.lock.yml, context7.lock.yml) |
| `.github/aw/` | Agentic workflow configuration | Contains actions-lock.json and other configs |
| `.github/aw/actions-lock.json` | Action version lock file | Pins action versions. Constant: `CacheFileName` |
| `.github/agents/` | Custom agent definitions | Custom agent markdown files |

### Source Code Directories

| Path | Description | Context |
|------|-------------|---------|
| `pkg/workflow/` | Workflow compilation code | Go package for compiling workflows |
| `pkg/workflow/js/` | JavaScript runtime code (generated) | CommonJS modules synced from actions/setup/js/ |
| `pkg/cli/` | CLI command implementations | gh-aw command handlers |
| `pkg/parser/` | Markdown frontmatter parsing | Schema validation and parsing |
| `pkg/parser/schemas/` | JSON schema definitions | Workflow validation schemas |
| `pkg/constants/` | Constants and configuration | Version numbers, timeouts, artifact names |
| `actions/setup/` | Setup action source | Source of truth for setup scripts |
| `actions/setup/js/` | JavaScript source files | Source for .cjs modules (synced to pkg/workflow/js/) |
| `actions/setup/sh/` | Shell script source files | Source for shell scripts (synced to pkg/workflow/sh/) |
| `specs/` | Specification documents | Documentation and specs directory |
| `docs/` | User documentation | Astro Starlight documentation site |

### Temporary and Runtime Paths

All temporary paths use the `/tmp/gh-aw/` prefix:

| Path | Description | Context |
|------|-------------|---------|
| `/tmp/gh-aw/` | Root temporary directory | Base directory for all temporary files |
| `/tmp/gh-aw/agent-stdio.log` | Agent stdio log | Console output from agent execution |
| `/tmp/gh-aw/aw.patch` | Generated patch file | Git diff for proposed changes |
| `/tmp/gh-aw/aw_info.json` | Workflow info | Metadata about workflow execution |
| `/tmp/gh-aw/aw-prompts/` | Prompt storage | Stores agent prompts |
| `/tmp/gh-aw/aw-prompts/prompt.txt` | Agent prompt text | The actual prompt sent to agent |
| `/tmp/gh-aw/cache-memory` | Cache memory directory | Agent cache memory storage |
| `/tmp/gh-aw/cache-memory-focus-areas` | Focus areas cache | Specialized cache storage |
| `/tmp/gh-aw/layout-cache` | Layout specification cache | Cache for layout maintainer |
| `/tmp/gh-aw/prompt-cache` | Prompt cache | Cached prompts |
| `/tmp/gh-aw/mcp-config/logs/` | MCP configuration logs | MCP server config logs |
| `/tmp/gh-aw/mcp-logs/` | MCP server logs | Runtime MCP logs |
| `/tmp/gh-aw/playwright-debug-logs/` | Playwright logs | Browser automation logs |
| `/tmp/gh-aw/python/` | Python working directory | Python script execution |
| `/tmp/gh-aw/python/charts/` | Generated charts | PNG chart outputs |
| `/tmp/gh-aw/python/data/` | Python data files | Data for Python analysis |
| `/tmp/gh-aw/redacted-urls.log` | Redacted URL log | Security log for URL filtering |
| `/tmp/gh-aw/repo-memory/` | Repository memory | Persistent memory storage |
| `/tmp/gh-aw/repo-memory/default` | Default repo memory | Default memory file |
| `/tmp/gh-aw/safe-inputs/logs/` | Safe inputs logs | Validated input logs |
| `/tmp/gh-aw/safe-jobs/` | Safe job data | Safe output job artifacts |
| `/tmp/gh-aw/safeoutputs/` | Safe outputs directory | Safe output execution data |
| `/tmp/gh-aw/safeoutputs/assets/` | Safe output assets | Assets for safe outputs |
| `/opt/gh-aw/safeoutputs/config.json` | Safe outputs config | MCP server configuration (read-only) |
| `/opt/gh-aw/safeoutputs/tools.json` | Safe outputs tools | Tool definitions for MCP (read-only) |
| `/opt/gh-aw/safeoutputs/validation.json` | Safe outputs validation | Validation rules (read-only) |
| `/tmp/gh-aw/safeoutputs/mcp-server.cjs` | Safe outputs MCP server | MCP server implementation |
| `/opt/gh-aw/safeoutputs/outputs.jsonl` | Safe outputs log | JSONL output log (read-only for agent) |
| `/tmp/gh-aw/sandbox/agent/logs/` | Agent sandbox logs | Sandboxed agent execution logs |
| `/tmp/gh-aw/sandbox/firewall/logs/` | Firewall sandbox logs | Sandboxed firewall logs |
| `/tmp/gh-aw/threat-detection/` | Threat detection data | Security analysis data |
| `/tmp/gh-aw/threat-detection/detection.log` | Threat detection log | Security scan log |
| `/tmp/gh-aw/session-data/` | Session data directory | Stores session information |
| `/tmp/gh-aw/session-data/sessions-list.json` | Session list | List of active sessions |
| `/tmp/gh-aw/session-data/sessions-schema.json` | Session schema | JSON schema for sessions |
| `/tmp/gh-aw/session-data/logs/` | Session logs | Log files for sessions |
| `/tmp/gh-aw/weekly-issues-data/` | Weekly issues data | Data for weekly issue reports |
| `/tmp/gh-aw/weekly-issues-data/issues.json` | Issues JSON | Collected issues data |
| `/tmp/gh-aw/weekly-issues-data/issues-schema.json` | Issues schema | JSON schema for issues |
| `/tmp/gh-aw/workflow-logs` | Workflow logs | Aggregated workflow logs |
| `/tmp/gh-aw/test-results.json` | Test results | JSON test results output |
| `/tmp/gh-aw/super-linter.log` | Super Linter log | Linting output log |
| `/tmp/gh-aw/prompts/` | Prompt templates | Template prompts (temp_folder_prompt.md, playwright_prompt.md) |
| `/tmp/gh-aw/scripts` | Runtime scripts | Temporary scripts directory |
| `/tmp/gh-aw/actions/` | Runtime action files | Copied from actions/setup/ at runtime (*.cjs, *.sh) |

### SBOM Output Paths

| Path | Description | Context |
|------|-------------|---------|
| `sbom.cdx.json` | CycloneDX SBOM | CycloneDX format SBOM output |
| `sbom.spdx.json` | SPDX SBOM | SPDX format SBOM output |

### Runtime Action Scripts (`/tmp/gh-aw/actions/`)

All action scripts are copied from `actions/setup/js/*.cjs` and `actions/setup/sh/*.sh` to `/tmp/gh-aw/actions/` at runtime by the setup action.

#### JavaScript Action Scripts (`.cjs`)

| Script | Description | Context |
|--------|-------------|---------|
| `add_reaction_and_edit_comment.cjs` | Adds reactions and edits comments | Safe-output handler |
| `assign_to_agent.cjs` | Assigns issues to agents | GitHub API integration |
| `check_command_position.cjs` | Validates command position in comments | Activation check |
| `check_membership.cjs` | Checks team membership | Authorization |
| `check_skip_if_match.cjs` | Checks skip-if-match patterns | Activation filter |
| `check_skip_if_no_match.cjs` | Checks skip-if-no-match patterns | Activation filter |
| `check_stop_time.cjs` | Validates workflow stop time | Time-based activation |
| `check_workflow_timestamp_api.cjs` | Checks workflow timestamps via API | Activation validation |
| `checkout_pr_branch.cjs` | Checks out pull request branch | Git operations |
| `collect_ndjson_output.cjs` | Collects NDJSON output | Data aggregation |
| `compute_text.cjs` | Computes text transformations | Text processing |
| `create_agent_task.cjs` | Creates agent sessions | Task orchestration |
| `determine_automatic_lockdown.cjs` | Determines lockdown status | Security |
| `generate_workflow_overview.cjs` | Generates workflow overview | Documentation |
| `interpolate_prompt.cjs` | Interpolates prompt templates | Prompt processing |
| `lock-issue.cjs` | Locks GitHub issues | Issue management |
| `unlock-issue.cjs` | Unlocks GitHub issues | Issue management |
| `missing_tool.cjs` | Reports missing tools | Safe-output handler |
| `noop.cjs` | No-operation handler | Safe-output handler |
| `notify_comment_error.cjs` | Notifies comment errors | Error handling |
| `parse_claude_log.cjs` | Parses Claude engine logs | Log parsing |
| `parse_codex_log.cjs` | Parses Codex engine logs | Log parsing |
| `parse_copilot_log.cjs` | Parses Copilot engine logs | Log parsing |
| `parse_custom_log.cjs` | Parses custom engine logs | Log parsing |
| `parse_firewall_logs.cjs` | Parses firewall logs | Security monitoring |
| `parse_safe_inputs_logs.cjs` | Parses safe inputs logs | Input validation |
| `parse_threat_detection_results.cjs` | Parses threat detection results | Security analysis |
| `push_repo_memory.cjs` | Pushes repository memory | Memory persistence |
| `redact_secrets.cjs` | Redacts secrets from logs | Security |
| `safe_output_handler_manager.cjs` | Manages safe output handlers | Safe-output orchestration |
| `setup_globals.cjs` | Sets up global configuration | Environment setup |
| `setup_threat_detection.cjs` | Sets up threat detection | Security initialization |
| `substitute_placeholders.cjs` | Substitutes placeholders | Template processing |
| `update_project.cjs` | Updates GitHub projects | Project management |
| `upload_assets.cjs` | Uploads assets | Asset management |
| `validate_errors.cjs` | Validates error outputs | Error handling |

#### Shell Action Scripts (`.sh`)

| Script | Description | Context |
|--------|-------------|---------|
| `clone_repo_memory_branch.sh` | Clones repo-memory branch | Memory initialization |
| `create_cache_memory_dir.sh` | Creates cache-memory directory | Cache setup |
| `create_gh_aw_tmp_dir.sh` | Creates /tmp/gh-aw/ directory | Temp directory setup |
| `create_prompt_first.sh` | Creates initial prompt file | Prompt initialization |
| `download_docker_images.sh` | Downloads Docker images | Container setup |
| `print_prompt_summary.sh` | Prints prompt summary | Debugging |
| `start_safe_inputs_server.sh` | Starts safe-inputs server | Safe-inputs initialization |
| `validate_multi_secret.sh` | Validates multiple secrets | Secret validation |
| `verify_mcp_gateway_health.sh` | Verifies MCP gateway health | Health check |

## Constants and Patterns

### Go Constants (from pkg/constants/constants.go)

#### Version Constants

| Constant | Type | Value | Description |
|----------|------|-------|-------------|
| `DefaultCopilotVersion` | `Version` | `"0.0.374"` | GitHub Copilot CLI version |
| `DefaultClaudeCodeVersion` | `Version` | `"2.0.76"` | Claude Code CLI version |
| `DefaultCodexVersion` | `Version` | `"0.78.0"` | OpenAI Codex CLI version |
| `DefaultGitHubMCPServerVersion` | `Version` | `"v0.27.0"` | GitHub MCP server Docker image |
| `DefaultFirewallVersion` | `Version` | `"v0.10.0"` | gh-aw-firewall (AWF) binary |
| `DefaultPlaywrightMCPVersion` | `Version` | `"0.0.54"` | @playwright/mcp package |
| `DefaultPlaywrightBrowserVersion` | `Version` | `"v1.57.0"` | Playwright browser Docker image |
| `DefaultMCPSDKVersion` | `Version` | `"1.24.0"` | @modelcontextprotocol/sdk package |
| `DefaultGitHubScriptVersion` | `Version` | `"v8"` | actions/github-script action version |
| `DefaultBunVersion` | `Version` | `"1.1"` | Bun runtime version |
| `DefaultNodeVersion` | `Version` | `"24"` | Node.js runtime version |
| `DefaultPythonVersion` | `Version` | `"3.12"` | Python runtime version |
| `DefaultRubyVersion` | `Version` | `"3.3"` | Ruby runtime version |
| `DefaultDotNetVersion` | `Version` | `"8.0"` | .NET runtime version |
| `DefaultJavaVersion` | `Version` | `"21"` | Java runtime version |
| `DefaultElixirVersion` | `Version` | `"1.17"` | Elixir runtime version |
| `DefaultGoVersion` | `Version` | `"1.25"` | Go runtime version |
| `DefaultHaskellVersion` | `Version` | `"9.10"` | GHC Haskell version |
| `DefaultDenoVersion` | `Version` | `"2.x"` | Deno runtime version |

#### Artifact Name Constants

| Constant | Type | Value | Description |
|----------|------|-------|-------------|
| `SafeOutputArtifactName` | `string` | `"safe-output"` | Safe outputs artifact name (file inside: `safe_output.jsonl`) |
| `AgentOutputArtifactName` | `string` | `"agent-output"` | Agent output artifact name (file inside: `agent_output.json`) |

#### Job Name Constants

| Constant | Type | Value | Description |
|----------|------|-------|-------------|
| `AgentJobName` | `JobName` | `"agent"` | Agent execution job name |
| `ActivationJobName` | `JobName` | `"activation"` | Activation job name |
| `PreActivationJobName` | `JobName` | `"pre_activation"` | Pre-activation job name |
| `DetectionJobName` | `JobName` | `"detection"` | Detection job name |

#### Step ID Constants

| Constant | Type | Value | Description |
|----------|------|-------|-------------|
| `CheckMembershipStepID` | `StepID` | `"check_membership"` | Checks team membership |
| `CheckStopTimeStepID` | `StepID` | `"check_stop_time"` | Validates stop time |
| `CheckSkipIfMatchStepID` | `StepID` | `"check_skip_if_match"` | Skip-if-match validation |
| `CheckSkipIfNoMatchStepID` | `StepID` | `"check_skip_if_no_match"` | Skip-if-no-match validation |
| `CheckCommandPositionStepID` | `StepID` | `"check_command_position"` | Command position validation |

#### Other Constants

| Constant | Type | Value | Description |
|----------|------|-------|-------------|
| `DefaultMCPRegistryURL` | `URL` | `"https://api.mcp.github.com/v0"` | Default MCP registry URL |
| `DefaultCopilotDetectionModel` | `ModelName` | `"gpt-5-mini"` | Default Copilot detection model |
| `MaxExpressionLineLength` | `LineLength` | `120` | Max line length for expressions |
| `ExpressionBreakThreshold` | `LineLength` | `100` | Threshold for breaking long lines |
| `DefaultActivationJobRunnerImage` | `string` | `"ubuntu-slim"` | Default runner for activation jobs |
| `SafeOutputsMCPServerID` | `string` | `"safeoutputs"` | Safe-outputs MCP server identifier |
| `SafeInputsMCPServerID` | `string` | `"safeinputs"` | Safe-inputs MCP server identifier |
| `SafeInputsMCPVersion` | `string` | `"1.0.0"` | Safe-inputs MCP server version |

#### Feature Flag Constants

| Constant | Type | Value | Description |
|----------|------|-------|-------------|
| `SafeInputsFeatureFlag` | `FeatureFlag` | `"safe-inputs"` | Safe-inputs feature flag |
| `MCPGatewayFeatureFlag` | `FeatureFlag` | `"mcp-gateway"` | MCP gateway feature flag |
| `SandboxRuntimeFeatureFlag` | `FeatureFlag` | `"sandbox-runtime"` | Sandbox runtime feature flag |

#### Pre-Activation Output Names

| Constant | Type | Value | Description |
|----------|------|-------|-------------|
| `IsTeamMemberOutput` | `string` | `"is_team_member"` | Team membership check output |
| `StopTimeOkOutput` | `string` | `"stop_time_ok"` | Stop time validation output |
| `SkipCheckOkOutput` | `string` | `"skip_check_ok"` | Skip-if-match check output |
| `SkipNoMatchCheckOkOutput` | `string` | `"skip_no_match_check_ok"` | Skip-if-no-match check output |
| `CommandPositionOkOutput` | `string` | `"command_position_ok"` | Command position check output |
| `ActivatedOutput` | `string` | `"activated"` | Activation status output |

#### Path Constants (Go)

| Constant | Type | Value | Description |
|----------|------|-------|-------------|
| `ScriptsBasePath` | `string` | `"/tmp/gh-aw/scripts"` | Runtime scripts directory |
| `RedactedURLsLogPath` | `string` | `"/tmp/gh-aw/redacted-urls.log"` | Redacted URLs log path |
| `SafeInputsDirectory` | `string` | `"/tmp/gh-aw/safe-inputs"` | Safe inputs directory |

#### Timeout Constants

| Constant | Type | Value | Description |
|----------|------|-------|-------------|
| `DefaultAgenticWorkflowTimeout` | `time.Duration` | `20 * time.Minute` | Default workflow timeout |
| `DefaultToolTimeout` | `time.Duration` | `60 * time.Second` | Default tool timeout |
| `DefaultMCPStartupTimeout` | `time.Duration` | `120 * time.Second` | Default MCP server startup timeout |

### JavaScript Patterns (from actions/setup/js/)

#### Path Construction Patterns

```javascript
// Temporary file patterns
path.join("/tmp", `test_agent_output_${Date.now()}_${Math.random().toString(36).slice(2)}.json`)

// Workflow file patterns
path.join(workspace, ".github", "workflows", `${workflowBasename}.md`)
path.join(workspace, ".github", "workflows", workflowFile)

// Common directory patterns
path.join(process.cwd(), "script_name.cjs")
path.join(__dirname, "module_name.cjs")
path.join(import.meta.dirname, "module_name.cjs")
path.join(os.tmpdir(), "prefix-")
```

#### Module Import Patterns

```javascript
// Common imports in CJS files
const path = require("path");
const fs = require("fs");
const os = require("os");

// Dynamic imports
const path = await import("path");
```

## Environment Variables

### Core Workflow Variables

| Variable | Description | Usage |
|----------|-------------|-------|
| `GH_AW_AGENT_OUTPUT` | Path to agent output file | Points to `/tmp/gh-aw/safeoutputs/agent-output/agent_output.json` (accounts for artifact subdirectory) |
| `GH_AW_SAFE_OUTPUTS` | Path to safe outputs file | Points to safe_output.jsonl location |
| `GITHUB_WORKSPACE` | GitHub Actions workspace path | Root directory of the repository |
| `GITHUB_TOKEN` | GitHub API authentication token | Used for API calls |

### Model Configuration Variables

| Variable | Description | Usage |
|----------|-------------|-------|
| `GH_AW_MODEL_AGENT_COPILOT` | Copilot model for agent | Overrides default Copilot model |
| `GH_AW_MODEL_AGENT_CLAUDE` | Claude model for agent | Overrides default Claude model |
| `GH_AW_MODEL_AGENT_CODEX` | Codex model for agent | Overrides default Codex model |
| `GH_AW_MODEL_DETECTION_COPILOT` | Copilot model for detection | Overrides detection model |
| `GH_AW_MODEL_DETECTION_CLAUDE` | Claude model for detection | Overrides detection model |
| `GH_AW_MODEL_DETECTION_CODEX` | Codex model for detection | Overrides detection model |

## GitHub Context Expressions

Common GitHub Actions expressions used in workflows:

### Event Properties

```yaml
# Issue events
github.event.issue.number
github.event.issue.state
github.event.issue.title

# Pull request events
github.event.pull_request.number
github.event.pull_request.state
github.event.pull_request.title
github.event.pull_request.head.sha
github.event.pull_request.base.sha

# Discussion events
github.event.discussion.number
github.event.discussion.title
github.event.discussion.category.name

# Comment events
github.event.comment.id

# Release events
github.event.release.id
github.event.release.tag_name
github.event.release.name
```

### Context Properties

```yaml
# Repository context
github.actor
github.repository
github.owner
github.workspace
github.server_url

# Workflow context
github.workflow
github.job
github.run_id
github.run_number

# Event context
github.event_name
github.event.after
github.event.before
```

### Common Conditional Patterns

```yaml
# Event type checks
(github.event_name == 'issues')
(github.event_name == 'issue_comment')
(github.event_name == 'pull_request')
(github.event_name == 'pull_request_review_comment')

# Draft PR checks
(github.event_name != 'pull_request') || (github.event.pull_request.draft == false)
(github.event_name != 'pull_request') || (github.event.pull_request.draft == true)

# Bot exclusions
(github.actor != 'dependabot[bot]')

# Body content checks
contains(github.event.issue.body, '@aw')
contains(github.event.pull_request.body, '@aw')
contains(github.event.comment.body, '@aw')
```

## Usage Guidelines

### Naming Conventions

- **Artifact naming**: Use descriptive hyphenated names (e.g., `agent-output`, `mcp-logs`)
  - Exception: Core artifacts use underscores for backwards compatibility (`agent_output.json`, `safe_output.jsonl`)
- **Job naming**: Use snake_case for job names (e.g., `create_pull_request`, `safe_outputs`)
- **Path references**: Use relative paths from repository root
- **Action pinning**: Always pin actions to full commit SHA for security

### Directory Organization

- **Workflow files**: `.github/workflows/` for all workflow definitions (.md and .lock.yml)
- **Shared configs**: `.github/workflows/shared/` for reusable components
- **Agent files**: `.github/agents/` for custom agent definitions
- **Configuration**: `.github/aw/` for workflow system configuration
- **Temporary files**: `/tmp/gh-aw/` for all runtime temporary data

### File Synchronization

Some files are automatically synced during build:

- **Shell scripts**: `actions/setup/sh/*.sh` → `pkg/workflow/sh/*.sh` (source → generated)
- **JavaScript files**: `actions/setup/js/*.cjs` → `pkg/workflow/js/*.cjs` (source → generated)
- **Never edit generated files directly** - always edit source files and run `make build`

### Security Best Practices

1. **Pin actions to commit SHAs**: Never use tags or branches
2. **Validate artifact paths**: Always check artifact paths before use
3. **Use safe expressions**: Follow allowed expressions list in `constants.go`
4. **Temporary file isolation**: Keep all temporary files in `/tmp/gh-aw/`
5. **Secret handling**: Never log secrets, use environment variables

## Workflow Patterns

### Typical Workflow Structure

```yaml
name: workflow-name
on: [trigger-events]
jobs:
  activation:
    # Determines if workflow should run
  pre_activation:
    # Early filtering (optional)
  agent:
    # Main AI agent execution
    needs: [activation]
  detection:
    # Post-agent analysis (optional)
    needs: [agent]
  safe_outputs:
    # Execute safe outputs
    needs: [agent]
  conclusion:
    # Final status reporting
    needs: [agent, safe_outputs]
    if: always()
```

### Artifact Flow Pattern

```yaml
# Job 1: Upload artifact
- uses: actions/upload-artifact@SHA
  with:
    name: agent_output.json
    path: /tmp/gh-aw/agent_output.json

# Job 2: Download artifact
- uses: actions/download-artifact@SHA
  with:
    name: agent_output.json
    path: /tmp/gh-aw/
```

### Environment Variable Setup Pattern

```yaml
- name: Set up environment variables
  run: |
    echo "GH_AW_AGENT_OUTPUT=/tmp/gh-aw/agent_output.json" >> "$GITHUB_ENV"
    echo "GH_AW_SAFE_OUTPUTS=/tmp/gh-aw/safe_output.jsonl" >> "$GITHUB_ENV"
```

## Extraction Summary

This specification was generated by analyzing:

- **Lock files analyzed**: 127 workflow files
- **Actions cataloged**: 18 unique GitHub Actions with pinned SHAs
- **Artifacts documented**: 45+ artifact patterns (standard, memory, specialized)
- **Job patterns found**: 27 common job names
- **File paths listed**: 90+ paths across workflows, source, and temporary locations
- **Runtime action scripts**: 40+ JavaScript scripts and 9 shell scripts
- **Go constants**: 70+ version, timeout, path, and configuration constants
- **JavaScript patterns**: Common path construction and import patterns
- **Environment variables**: 10+ workflow and model configuration variables
- **Step IDs**: 5 pre-activation step identifiers
- **Feature flags**: 3 feature flag constants
- **GitHub expressions**: 50+ allowed expressions for workflow conditions

### Source Analysis

- Scanned all `.lock.yml` files in `.github/workflows/`
- Reviewed Go code in `pkg/workflow/`, `pkg/cli/`, `pkg/constants/`, `pkg/parser/`
- Reviewed JavaScript code in `actions/setup/js/`
- Extracted patterns using `yq` for YAML parsing
- Extracted constants from Go source files
- Documented JavaScript path construction patterns

## Maintenance

This specification should be updated when:

- New workflow patterns are introduced
- New artifacts are created
- Job names are standardized or changed
- File path conventions evolve
- New actions are added or version numbers change
- Constants are added or modified in `pkg/constants/constants.go`

To regenerate this specification, run the Layout Specification Maintainer workflow, which will automatically scan all lock files and source code to update this document.

---

*This document is automatically maintained by the Layout Specification Maintainer workflow.*
*Generated by scanning 127 workflow files and associated Go/JavaScript source code.*
