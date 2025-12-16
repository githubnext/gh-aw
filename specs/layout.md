# GitHub Actions Workflow Layout Specification

> Auto-generated specification documenting patterns used in compiled `.lock.yml` files.
> Last updated: 2025-12-16

## Overview

This document catalogs all file paths, folder names, artifact names, and other patterns used across our compiled GitHub Actions workflows (`.lock.yml` files). Based on analysis of **116 lock files** in `.github/workflows/`.

## GitHub Actions

Common GitHub Actions used across workflows (pinned to specific commit SHAs):

| Action | SHA | Description | Context |
|--------|-----|-------------|---------|
| actions/checkout | 93cb6efe18208431cddfb8368fd83d5badbf9bfd | Checks out repository code | Used in almost all workflows for accessing repo content |
| actions/upload-artifact | 330a01c490aca151604b8cf639adc76d48f6c5d4 | Uploads build artifacts | Used for agent outputs, patches, prompts, and logs |
| actions/download-artifact | 018cc2cf5baa6db3ef3c5f8a56943fffe632ef53 | Downloads artifacts from previous jobs | Used in safe-output jobs and conclusion jobs |
| actions/github-script | 60a0d83039c74a4aee543508d2ffcb1c3799cdea | Runs GitHub API scripts | Used for GitHub API interactions and safe-output implementations |
| actions/github-script | ed597411d8f924073f98dfc5c65a23a2325f34cd | Alternative version for specific use cases | Used in some workflows |
| actions/setup-node | 395ad3262231945c25e8478fd5baf05154b1d79f | Sets up Node.js environment | Used in workflows requiring npm/node |
| actions/setup-python | a26af69be951a213d495a4c3e4e4022e16d87065 | Sets up Python environment | Used for Python-based workflows and safe-inputs |
| actions/setup-go | 4dc6199c7b1a012772edbd06daecab0f50c9053c | Sets up Go environment | Used for Go development workflows |
| actions/cache | 0057852bfaa89a56745cba8c7296529d2fc39830 | Caches dependencies | Improves workflow performance |
| actions/cache/restore | 0057852bfaa89a56745cba8c7296529d2fc39830 | Restores cached dependencies | Used with cache/save for granular control |
| actions/cache/save | 0057852bfaa89a56745cba8c7296529d2fc39830 | Saves dependencies to cache | Used with cache/restore for granular control |
| actions/ai-inference | b81b2afb8390ee6839b494a404766bef6493c7d9 | Runs AI inference | Used for AI-powered analysis |
| actions/create-github-app-token | 29824e69f54612133e76f7eaac726eef6c875baf | Creates GitHub App token | Used for authentication in workflows |
| github/codeql-action/upload-sarif | 323fb8c0ad5be63b7a6ebf1f32c35882fcfea2cf | Uploads code scanning results | Used for security scanning workflows |
| super-linter/super-linter | 2bdd90ed3262e023ac84bf8fe35dc480721fc1f2 | Runs super-linter | Used for code quality checks |
| cli/gh-extension-precompile | 9e2237c30f869ad3bcaed6a4be2cd43564dd421b | Precompiles gh extensions | Used for CLI extension workflows |
| github/stale-repos | 3477b6488008d9411aaf22a0924ec7c1f6a69980 | Identifies stale repositories | Used for repository health workflows |
| anchore/sbom-action | fbfd9c6c189226748411491745178e0c2017392d | Generates Software Bill of Materials | Used for dependency tracking |
| astral-sh/setup-uv | e58605a9b6da7c637471fab8847a5e5a6b8df081 | Sets up UV package manager | Used for Python workflows |

## Artifact Names

Artifacts uploaded/downloaded between workflow jobs:

### Core Artifacts

| Name | Description | Upload Context | Download Context |
|------|-------------|----------------|------------------|
| agent_output.json | AI agent execution output JSON | Uploaded by agent job | Downloaded by detection and safe-output jobs |
| safe_output.jsonl | Safe outputs configuration (JSONL) | Uploaded by agent job | Downloaded by all safe-output jobs |
| aw.patch | Git patch file for changes | Uploaded by agent job | Downloaded by create_pull_request job |
| prompt.txt | Agent prompt content | Uploaded by agent job | Downloaded for debugging and audit purposes |
| aw_info.json | Agentic workflow run information | Uploaded by agent job | Used for metadata tracking |
| mcp-logs | MCP server logs directory | Uploaded by agent job | Downloaded for debugging |
| agent-stdio.log | Agent standard I/O logs | Uploaded by agent job | Downloaded for debugging |

### Cache Memory Artifacts

| Name | Description | Context |
|------|-------------|---------|
| cache-memory | Default cache memory artifact | Used for persistent agent memory |
| cache-memory-{id} | Named cache memory artifact | Used when cache has custom ID |
| cache-memory-focus-areas | Focus areas cache | Used by specific workflows |
| repo-memory-default | Repository-specific memory | Used for repository context persistence |

### Firewall Log Artifacts

Firewall logs are named following the pattern: `firewall-logs-{workflow-name}`

Examples include:
- firewall-logs-ai-moderator
- firewall-logs-archie
- firewall-logs-daily-news
- firewall-logs-dev
- firewall-logs-mergefest
- firewall-logs-release
- (and many more, one per workflow)

### Specialized Artifacts

| Name | Description | Context |
|------|-------------|---------|
| safe-outputs-assets | Assets from safe-output operations | Used by safe-output jobs |
| safeinputs | Safe inputs data | Used by workflows with safe-input configuration |
| playwright-debug-logs-${{ github.run_id }} | Playwright browser debug logs | Used in browser automation workflows |
| data-charts | Generated data visualizations | Used by Python data chart workflows |
| python-source-and-data | Python scripts and data files | Used by Python workflows |
| trending-charts | Trending data visualizations | Used by analytics workflows |
| trending-source-and-data | Trending analysis data | Used by analytics workflows |
| super-linter-log | Super-linter output logs | Used by linting workflows |
| code-scanning-alert.sarif | SARIF format security scan results | Used by security scanning workflows |
| sbom-artifacts | Software Bill of Materials | Used by SBOM generation workflows |
| threat-detection.log | Security threat detection logs | Used by security workflows |

## Common Job Names

Standard job names across workflows:

| Job Name | Description | Context |
|----------|-------------|---------|
| activation | Determines if workflow should run | Uses skip-if-match, stop-time, and permission filters |
| pre_activation | Pre-activation checks | Runs before activation job for early filtering |
| agent | Main AI agent execution job | Runs the copilot/claude/codex engine |
| detection | Post-agent analysis job | Analyzes agent output for patterns |
| conclusion | Final status reporting job | Runs after all other jobs complete |
| create_pull_request | Creates PR from agent changes | Safe-output job for PR creation |
| push_to_pull_request_branch | Pushes changes to PR branch | Safe-output job for PR updates |
| add_comment | Adds comment to issue/PR | Safe-output job for commenting |
| add_labels | Adds labels to issue/PR | Safe-output job for label management |
| create_issue | Creates new issue | Safe-output job for issue creation |
| update_issue | Updates existing issue | Safe-output job for issue updates |
| close_issue | Closes issue | Safe-output job for issue closure |
| create_discussion | Creates discussion | Safe-output job for discussion creation |
| close_discussion | Closes discussion | Safe-output job for discussion closure |
| create_pull_request | Creates pull request | Safe-output job for PR creation |
| update_pull_request | Updates pull request | Safe-output job for PR updates |
| close_pull_request | Closes pull request | Safe-output job for PR closure |
| create_pr_review_comment | Creates PR review comment | Safe-output job for code review |
| assign_to_user | Assigns issue/PR to user | Safe-output job for assignment |
| assign_to_agent | Assigns to GitHub Copilot agent | Safe-output job for agent task creation |
| create_agent_task | Creates agent task | Safe-output job for agent task management |
| link_sub_issue | Links sub-issue to parent | Safe-output job for issue hierarchy |
| hide_comment | Hides comment | Safe-output job for comment moderation |
| search_issues | Searches issues | Safe-output job for issue queries |
| update_cache_memory | Updates cache memory | Safe-output job for memory persistence |
| push_repo_memory | Pushes repository memory | Safe-output job for repo context |
| notion_add_comment | Adds comment to Notion | Safe-output job for Notion integration |
| post_to_slack_channel | Posts to Slack channel | Safe-output job for Slack notifications |
| update_project | Updates GitHub project | Safe-output job for project management |
| update_release | Updates release | Safe-output job for release management |
| upload_assets | Uploads release assets | Safe-output job for asset management |
| create_code_scanning_alert | Creates code scanning alert | Safe-output job for security alerts |
| check_external_user | Checks if user is external | Safe-output job for permission validation |
| ast_grep | Runs AST grep analysis | Safe-output job for code pattern matching |
| super_linter | Runs super-linter | Safe-output job for code quality |
| generate-sbom | Generates Software Bill of Materials | Safe-output job for dependency tracking |
| post-issue | Posts issue from workflow | Safe-output job for issue creation |
| release | Performs release operations | Safe-output job for release automation |

## Common Step Names

Most frequently used step names across workflows (top 40):

| Count | Step Name | Description |
|-------|-----------|-------------|
| 411 | Download agent output artifact | Downloads agent_output.json from agent job |
| 300 | Setup agent output environment variable | Sets GH_AW_AGENT_OUTPUT env var |
| 222 | Substitute placeholders | Replaces template placeholders in safe-output configs |
| 175 | Configure Git credentials | Sets up Git authentication for commits |
| 175 | Checkout repository | Checks out repository code |
| 142 | Validate COPILOT_GITHUB_TOKEN secret | Ensures Copilot token is available |
| 142 | Install GitHub Copilot CLI | Installs gh-copilot extension |
| 142 | Execute GitHub Copilot CLI | Runs copilot agent |
| 136 | Download patch artifact | Downloads aw.patch from agent job |
| 114 | Upload prompt | Uploads prompt.txt artifact |
| 114 | Upload agentic run info | Uploads aw_info.json artifact |
| 114 | Upload MCP logs | Uploads mcp-logs artifact |
| 114 | Upload Agent Stdio | Uploads agent-stdio.log artifact |
| 114 | Setup MCPs | Configures MCP servers |
| 114 | Redact secrets in logs | Removes sensitive data from logs |
| 114 | Print prompt | Outputs prompt to step summary |
| 114 | Generate workflow overview | Creates workflow metadata |
| 114 | Generate agentic run info | Creates run metadata |
| 114 | Create prompt | Generates agent prompt |
| 114 | Create gh-aw temp directory | Creates /tmp/gh-aw/ directory |
| 114 | Check workflow file timestamps | Validates workflow file freshness |
| 114 | Append temporary folder instructions to prompt | Adds /tmp/gh-aw/agent/ instructions |
| 114 | Append XPIA security instructions to prompt | Adds security warnings to prompt |
| 113 | Validate agent logs for errors | Checks for agent execution errors |
| 113 | Parse agent logs for step summary | Extracts summary from logs |
| 113 | Interpolate variables and render templates | Processes template variables |
| 112 | Checkout PR branch | Checks out pull request branch |
| 111 | Append GitHub context to prompt | Adds GitHub metadata to prompt |
| 110 | Write Safe Outputs JavaScript Files | Writes safe-output .cjs files |
| 110 | Write Safe Outputs Config | Writes safe-outputs configuration |
| 110 | Upload sanitized agent output | Uploads redacted agent output |
| 110 | Upload Safe Outputs | Uploads safe_output.jsonl artifact |
| 110 | Update reaction comment with completion status | Updates comment with status emoji |
| 110 | Record Missing Tool | Records missing tool messages |
| 110 | Process No-Op Messages | Processes no-op safe outputs |
| 110 | Ingest agent output | Parses and validates agent output |
| 110 | Debug job inputs | Outputs job input for debugging |
| 110 | Append safe outputs instructions to prompt | Adds safe-outputs guidance to prompt |
| 108 | Upload threat detection log | Uploads threat detection results |
| 108 | Setup threat detection | Configures threat detection system |

## File Paths

Common file paths referenced in workflows:

### Workflow Directory Structure

| Path | Description | Context |
|------|-------------|---------|
| .github/workflows/ | Workflow definition directory | Contains all .md and .lock.yml files |
| .github/workflows/shared/ | Shared workflow components | Reusable workflow imports |
| .github/workflows/shared/mcp/ | Shared MCP server configs | MCP configuration imports |
| .github/aw/ | Agentic workflow configuration | Contains actions-lock.json and cache |
| .github/aw/actions-lock.json | Action version lock file | Stores pinned action versions |

### Source Code Paths

| Path | Description | Context |
|------|-------------|---------|
| pkg/workflow/ | Workflow compilation code | Go package for compiling workflows |
| pkg/workflow/js/ | JavaScript runtime code | CommonJS modules for GitHub Actions |
| pkg/constants/ | Constants definitions | Go package for shared constants |
| pkg/cli/ | CLI command implementations | gh-aw command handlers |
| pkg/parser/ | Markdown frontmatter parsing | Schema validation and parsing |
| internal/ | Internal packages | Private implementation code |
| cmd/gh-aw/ | CLI entry point | Main executable code |
| specs/ | Specification documents | Documentation and specs directory |
| docs/ | Documentation site | Astro Starlight documentation |

### Temporary File Paths

All temporary files use the `/tmp/gh-aw/` directory:

| Path | Description | Context |
|------|-------------|---------|
| /tmp/gh-aw/ | Root temporary directory | Created by all workflows |
| /tmp/gh-aw/agent/ | Agent workspace | Agent working directory |
| /tmp/gh-aw/agent-stdio.log | Agent I/O log | Captures agent console output |
| /tmp/gh-aw/aw-prompts/prompt.txt | Prompt file | Generated agent prompt |
| /tmp/gh-aw/aw.patch | Git patch file | Changes to be applied |
| /tmp/gh-aw/aw_info.json | Workflow run info | Metadata about execution |
| /tmp/gh-aw/cache-memory | Cache memory directory | Persistent agent memory |
| /tmp/gh-aw/cache-memory-focus-areas | Focus areas cache | Specialized cache |
| /tmp/gh-aw/layout-cache | Layout cache | Layout-specific cache |
| /tmp/gh-aw/prompt-cache | Prompt cache | Cached prompts |
| /tmp/gh-aw/repo-memory-default | Repository memory | Repo-specific context |
| /tmp/gh-aw/mcp-config/logs/ | MCP config logs | MCP setup logs |
| /tmp/gh-aw/mcp-logs/ | MCP server logs | MCP runtime logs |
| /tmp/gh-aw/playwright-debug-logs/ | Playwright debug logs | Browser automation logs |
| /tmp/gh-aw/python/*.py | Python scripts | Generated Python code |
| /tmp/gh-aw/python/charts/*.png | Chart images | Generated visualizations |
| /tmp/gh-aw/python/data/* | Data files | Python workflow data |
| /tmp/gh-aw/redacted-urls.log | Redacted URLs log | URLs removed from logs |
| /tmp/gh-aw/safe-inputs/logs/ | Safe inputs logs | Safe input processing logs |
| /tmp/gh-aw/safe-jobs/ | Safe jobs data | Safe job execution data |
| /tmp/gh-aw/safeoutputs/ | Safe outputs directory | Safe output processing |
| /tmp/gh-aw/safeoutputs/assets/ | Safe output assets | Assets from safe outputs |
| /tmp/gh-aw/sandbox/agent/logs/ | Sandboxed agent logs | Agent logs in sandbox |
| /tmp/gh-aw/sandbox/firewall/logs/ | Firewall logs | Firewall execution logs |
| /tmp/gh-aw/threat-detection/ | Threat detection data | Security scan data |
| /tmp/gh-aw/threat-detection/detection.log | Detection log | Threat detection results |

### Output File Paths

| Path | Description | Context |
|------|-------------|---------|
| super-linter.log | Super-linter output | Linting results |
| sbom.cdx.json | CycloneDX SBOM | Dependency manifest |
| sbom.spdx.json | SPDX SBOM | Alternative SBOM format |

## Working Directories

Common working directories used in workflow steps:

| Path | Description | Context |
|------|-------------|---------|
| ./docs | Documentation directory | Used for documentation build steps |
| ./pkg/workflow/js | JavaScript source | Used for JavaScript compilation |

## Artifact Retention Policies

Retention days for uploaded artifacts:

| Days | Description | Context |
|------|-------------|---------|
| 1 | Short-term artifacts | Temporary debug logs |
| 7 | Standard artifacts | Most agent outputs and logs |
| 30 | Medium-term storage | Important analysis results |
| 90 | Long-term storage | Critical audit data |

## Go Constants

Key constants from `pkg/constants/constants.go`:

### Job and Artifact Constants

```go
const AgentJobName = "agent"
const ActivationJobName = "activation"
const PreActivationJobName = "pre_activation"
const DetectionJobName = "detection"
const SafeOutputArtifactName = "safe_output.jsonl"
const AgentOutputArtifactName = "agent_output.json"
```

### MCP Server Constants

```go
const SafeOutputsMCPServerID = "safeoutputs"
const SafeInputsMCPServerID = "safeinputs"
const SafeInputsMCPVersion = "1.0.0"
```

### Step and Output Constants

```go
// Pre-activation step IDs
const CheckMembershipStepID = "check_membership"
const CheckStopTimeStepID = "check_stop_time"
const CheckSkipIfMatchStepID = "check_skip_if_match"
const CheckCommandPositionStepID = "check_command_position"

// Output names
const IsTeamMemberOutput = "is_team_member"
const StopTimeOkOutput = "stop_time_ok"
const SkipCheckOkOutput = "skip_check_ok"
const CommandPositionOkOutput = "command_position_ok"
const ActivatedOutput = "activated"
```

### Version Constants

```go
const DefaultCopilotVersion = "0.0.369"
const DefaultCopilotDetectionModel = "gpt-5-mini"
const DefaultClaudeCodeVersion = "2.0.69"
const DefaultCodexVersion = "0.72.0"
const DefaultGitHubMCPServerVersion = "v0.24.1"
const DefaultFirewallVersion = "v0.6.0"
const DefaultPlaywrightMCPVersion = "0.0.52"
const DefaultPlaywrightBrowserVersion = "v1.57.0"
const DefaultMCPSDKVersion = "1.24.0"
```

### Runtime Version Constants

```go
const DefaultBunVersion = "1.1"
const DefaultNodeVersion = "24"
const DefaultPythonVersion = "3.12"
const DefaultRubyVersion = "3.3"
const DefaultDotNetVersion = "8.0"
const DefaultJavaVersion = "21"
const DefaultElixirVersion = "1.17"
const DefaultGoVersion = "1.25"
const DefaultHaskellVersion = "9.10"
const DefaultDenoVersion = "2.x"
```

### Timeout Constants

```go
const DefaultAgenticWorkflowTimeout = 20 * time.Minute
const DefaultToolTimeout = 60 * time.Second
const DefaultMCPStartupTimeout = 120 * time.Second
const DefaultAgenticWorkflowTimeoutMinutes = 20
const DefaultToolTimeoutSeconds = 60
const DefaultMCPStartupTimeoutSeconds = 120
```

### Expression Formatting Constants

```go
const MaxExpressionLineLength = 120
const ExpressionBreakThreshold = 100
```

### Runner Image Constants

```go
const DefaultActivationJobRunnerImage = "ubuntu-slim"
```

### Directory Helper

```go
func GetWorkflowDir() string {
    return filepath.Join(".github", "workflows")
}
```

## JavaScript Patterns

Common patterns from `pkg/workflow/js/*.cjs` files:

### Path Construction

```javascript
const path = require("path");
const workflowBasename = path.basename(workflowFile, ".lock.yml");
const workflowMdFile = path.join(workspace, ".github", "workflows", `${workflowBasename}.md`);
const lockFile = path.join(workspace, ".github", "workflows", workflowFile);
```

### Script Path References

JavaScript files reference each other using relative paths:
- `add_comment.cjs`
- `add_labels.cjs`
- `add_reaction_and_edit_comment.cjs`
- `assign_issue.cjs`
- `assign_milestone.cjs`
- `check_command_position.cjs`
- `check_membership.cjs`
- `check_permissions.cjs`
- `check_skip_if_match.cjs`
- `check_stop_time.cjs`
- `check_team_member.cjs`
- `check_workflow_timestamp.cjs`
- `safe_outputs_config.cjs`

## Environment Variables

Common environment variables used in workflows:

| Variable | Description | Context |
|----------|-------------|---------|
| GH_AW_AGENT_OUTPUT | Path to agent_output.json | Set after downloading agent output |
| GH_AW_SAFE_OUTPUTS | Path to safe_output.jsonl | Set after downloading safe outputs |
| GH_AW_COMMENT_ID | Comment ID from activation | Passed to agent from activation job |
| GH_AW_COMMENT_REPO | Repository for comment | Passed to agent from activation job |
| ORGANIZATION | Target organization | Used in org-wide workflows |

## Safe Output Types

Safe output types follow the pattern: `BuildSafeOutputType(jobName)`

Common safe output jobs include all job names listed in the "Common Job Names" section above that start with actions like:
- create_*
- update_*
- close_*
- add_*
- assign_*
- link_*
- hide_*
- search_*
- push_*
- post_*
- upload_*

## Container Images

While no custom container images were found in the scanned workflows, the following patterns are used:

- **Default runner**: `ubuntu-latest` or `ubuntu-slim` for activation jobs
- **Docker containers**: MCP servers run in Docker containers with images like:
  - GitHub MCP server: `ghcr.io/github/github-mcp-server:${DefaultGitHubMCPServerVersion}`
  - Playwright: `mcr.microsoft.com/playwright:${DefaultPlaywrightBrowserVersion}`

## Usage Guidelines

### Artifact Naming

- Use descriptive hyphenated names (e.g., `agent-output`, `mcp-logs`)
- For workflow-specific artifacts, use prefix pattern: `{type}-{workflow-name}`
- For cache artifacts with IDs, use: `cache-memory-{id}`

### Job Naming

- Use snake_case for job names (e.g., `create_pull_request`)
- Follow action pattern: `{verb}_{noun}` (e.g., `update_issue`, `close_discussion`)
- Use consistent prefixes: `create_`, `update_`, `close_`, `add_`, `assign_`, etc.

### Path References

- Always use relative paths from repository root
- Use `filepath.Join()` in Go for path construction
- Use `path.join()` in JavaScript for path construction
- Temporary files MUST go in `/tmp/gh-aw/` subdirectories

### Action Pinning

- Always pin actions to full commit SHA for security
- Use `GetActionPin("actions/checkout")` helper in Go code
- Update pins in centralized action_pins.go file

### Step Naming

- Use descriptive action-based names (e.g., "Download agent output artifact")
- Be consistent with common patterns shown in "Common Step Names" section
- Use title case for step names

### Environment Variables

- Use `GH_AW_` prefix for workflow-specific variables
- Document all custom environment variables
- Pass variables between jobs using needs.{job}.outputs pattern

## Statistics Summary

- **Lock files analyzed**: 116
- **GitHub Actions cataloged**: 19 unique actions
- **Artifact types documented**: 60+ unique artifacts
- **Job patterns found**: 41 standard job names
- **Common step patterns**: 40 most frequently used steps
- **File paths listed**: 50+ distinct paths
- **Constants defined**: 35+ core constants
- **Working directories**: 2 standard paths
- **Environment variables**: 5+ core variables

---

*This document is automatically maintained by the Layout Specification Maintainer workflow.*
*Generated from analysis of all .lock.yml files in .github/workflows/ and source code in pkg/workflow/.*
