# GitHub Actions Workflow Layout Specification

> Auto-generated specification documenting patterns used in compiled `.lock.yml` files.
> Last updated: 2025-12-11

## Overview

This document catalogs all file paths, folder names, artifact names, and other patterns used across our compiled GitHub Actions workflows (`.lock.yml` files). The repository currently contains **118 compiled lock.yml files** with **19 unique GitHub Actions**, **85+ unique artifact patterns**, and **38 unique job types**.

## GitHub Actions

Common GitHub Actions used across workflows (19 unique actions):

| Action | Description | Context |
|--------|-------------|---------|
| `actions/ai-inference@b81b2afb8390ee6839b494a404766bef6493c7d9` | AI inference action for model execution | Used for AI/ML model integration |
| `actions/cache@0057852bfaa89a56745cba8c7296529d2fc39830` | Caches dependencies and build outputs | Used for speeding up workflows by caching dependencies |
| `actions/cache/restore@0057852bfaa89a56745cba8c7296529d2fc39830` | Restores cache from previous runs | Used to restore cached dependencies |
| `actions/cache/save@0057852bfaa89a56745cba8c7296529d2fc39830` | Saves cache for future runs | Used to save dependencies and build outputs |
| `actions/checkout@93cb6efe18208431cddfb8368fd83d5badbf9bfd` | Checks out repository code | Used in almost all workflows for accessing repo content |
| `actions/create-github-app-token@29824e69f54612133e76f7eaac726eef6c875baf` | Creates GitHub App installation token | Used for GitHub App authentication |
| `actions/download-artifact@018cc2cf5baa6db3ef3c5f8a56943fffe632ef53` | Downloads artifacts from previous jobs | Used in safe-output jobs and conclusion jobs |
| `actions/github-script@60a0d83039c74a4aee543508d2ffcb1c3799cdea` | Runs GitHub API scripts (primary version) | Used for GitHub API interactions and custom scripts |
| `actions/github-script@ed597411d8f924073f98dfc5c65a23a2325f34cd` | Runs GitHub API scripts (alternate version) | Legacy version still in use in some workflows |
| `actions/setup-go@d35c59abb061a4a6fb18e82ac0862c26744d6ab5` | Sets up Go environment | Used in workflows requiring Go compilation or execution |
| `actions/setup-node@395ad3262231945c25e8478fd5baf05154b1d79f` | Sets up Node.js environment | Used in workflows requiring npm/node |
| `actions/setup-python@a26af69be951a213d495a4c3e4e4022e16d87065` | Sets up Python environment | Used in workflows requiring Python execution |
| `actions/upload-artifact@330a01c490aca151604b8cf639adc76d48f6c5d4` | Uploads build artifacts | Used for agent outputs, patches, prompts, and logs |
| `anchore/sbom-action@fbfd9c6c189226748411491745178e0c2017392d` | Generates Software Bill of Materials | Used for security and dependency tracking |
| `astral-sh/setup-uv@e58605a9b6da7c637471fab8847a5e5a6b8df081` | Sets up UV Python package manager | Used for faster Python dependency management |
| `cli/gh-extension-precompile@9e2237c30f869ad3bcaed6a4be2cd43564dd421b` | Precompiles GitHub CLI extensions | Used for building and packaging CLI extensions |
| `github/codeql-action/upload-sarif@6d10af73d5896080e7b530ea0822d927c05661f1` | Uploads CodeQL SARIF results | Used for code security scanning |
| `github/stale-repos@v3` | Identifies stale repositories | Used for repository health monitoring |
| `super-linter/super-linter@2bdd90ed3262e023ac84bf8fe35dc480721fc1f2` | Runs multiple linters on codebase | Used for code quality and style enforcement |

## Artifact Names

Artifacts uploaded/downloaded between workflow jobs (85+ unique artifact names):

### Core Agent Artifacts

| Name | Description | Context |
|------|-------------|---------|
| `agent_output.json` | AI agent execution output | Contains the agent's response and analysis (see `AgentOutputArtifactName` constant) |
| `agent-stdio.log` | Agent standard I/O logs | Captures console output from agent execution |
| `aw.patch` | Git patch file for changes | Used by create-pull-request safe-output |
| `aw_info.json` | Workflow metadata and info | Contains workflow execution metadata |
| `prompt.txt` | Agent prompt content | Stored for debugging and audit purposes |
| `safe_output.jsonl` | Safe outputs configuration | Passed from agent to safe-output jobs (see `SafeOutputArtifactName` constant) |
| `safe-outputs-assets` | Assets for safe output jobs | Files needed by safe-output processing |

### Memory & Cache Artifacts

| Name | Description | Context |
|------|-------------|---------|
| `cache-memory` | Default cache memory storage | Persistent memory across workflow runs |
| `cache-memory-focus-areas` | Focused memory areas | Specialized cache for focus-based workflows |
| `repo-memory-default` | Default repository memory | Shared memory for repository state |

### Log & Debug Artifacts

| Name | Description | Context |
|------|-------------|---------|
| `mcp-logs` | MCP server logs | Debug logs from Model Context Protocol servers |
| `playwright-debug-logs-${{ github.run_id }}` | Playwright browser debug logs | Dynamic artifact name with run ID |
| `firewall-logs-*` | Firewall security logs (60+ variants) | Per-workflow firewall logs named `firewall-logs-{workflow-name}` |
| `super-linter-log` | Super linter execution logs | Output from super-linter action |
| `threat-detection.log` | Security threat detection logs | Logs from security scanning |

### Data & Visualization Artifacts

| Name | Description | Context |
|------|-------------|---------|
| `data-charts` | Generated chart files | Data visualization outputs |
| `python-source-and-data` | Python scripts and data files | Python workflow artifacts |
| `trending-charts` | Trending data visualizations | Charts showing trends over time |
| `trending-source-and-data` | Source files for trending analysis | Data sources for trend analysis |

### Security & Quality Artifacts

| Name | Description | Context |
|------|-------------|---------|
| `code-scanning-alert.sarif` | Code scanning results in SARIF format | Used by CodeQL and security scanners |
| `sbom-artifacts` | Software Bill of Materials artifacts | SBOM files from Anchore scan |
| `sbom.cdx.json` | CycloneDX SBOM format | Standard SBOM format output |
| `sbom.spdx.json` | SPDX SBOM format | Alternative SBOM format output |

### Safe Input Artifacts

| Name | Description | Context |
|------|-------------|---------|
| `safeinputs` | Safe inputs processing artifacts | Data from safe-inputs MCP server |

## Job Names

Standard job names across workflows (37 unique job types):

### Core Workflow Jobs

| Job Name | Description | Context |
|----------|-------------|---------|
| `activation` | Determines if workflow should run | Uses skip-if-match and other filters (see `ActivationJobName` constant) |
| `pre_activation` | Pre-flight checks before activation | Permission and membership checks (see `PreActivationJobName` constant) |
| `agent` | Main AI agent execution job | Runs the copilot/claude/codex engine (see `AgentJobName` constant) |
| `detection` | Post-agent analysis job | Analyzes agent output for patterns (see `DetectionJobName` constant) |
| `conclusion` | Final status reporting job | Runs after all other jobs complete |

### Safe Output Jobs - GitHub Resource Creation

| Job Name | Description | Context |
|----------|-------------|---------|
| `create_pull_request` | Creates PR from agent changes | Safe-output job for PR creation |
| `create_issue` | Creates a new issue | Safe-output job for issue creation |
| `create_discussion` | Creates a new discussion | Safe-output job for discussion creation |
| `create_pr_review_comment` | Adds review comment to PR | Safe-output job for PR review feedback |
| `create_agent_task` | Creates task for agent execution | Safe-output job for task creation |
| `create_code_scanning_alert` | Creates code scanning security alert | Safe-output job for security findings |

### Safe Output Jobs - Resource Updates

| Job Name | Description | Context |
|----------|-------------|---------|
| `update_issue` | Updates existing issue | Safe-output job for issue modification |
| `update_pull_request` | Updates existing PR | Safe-output job for PR modification |
| `update_project` | Updates project board | Safe-output job for project management |
| `update_release` | Updates release notes/assets | Safe-output job for release management |
| `update_cache_memory` | Updates cache memory storage | Safe-output job for memory persistence |

### Safe Output Jobs - Comments & Interaction

| Job Name | Description | Context |
|----------|-------------|---------|
| `add_comment` | Adds comment to issue/PR | Safe-output job for commenting |
| `add_labels` | Adds labels to issue/PR | Safe-output job for labeling |
| `hide_comment` | Hides a comment | Safe-output job for comment moderation |
| `notion_add_comment` | Adds comment to Notion | Safe-output job for Notion integration |

### Safe Output Jobs - Assignment & Linking

| Job Name | Description | Context |
|----------|-------------|---------|
| `assign_to_agent` | Assigns issue/PR to agent | Safe-output job for agent assignment |
| `assign_to_user` | Assigns issue/PR to user | Safe-output job for user assignment |
| `link_sub_issue` | Links sub-issues to parent | Safe-output job for issue hierarchy |

### Safe Output Jobs - State Changes

| Job Name | Description | Context |
|----------|-------------|---------|
| `close_issue` | Closes an issue | Safe-output job for issue closure |
| `close_pull_request` | Closes a PR | Safe-output job for PR closure |
| `close_discussion` | Closes a discussion | Safe-output job for discussion closure |
| `hide_comment` | Hides a comment (duplicate - see Comments section) | Safe-output job for comment moderation |
| `minimize_comment` | Minimizes a comment | Safe-output job for comment moderation |

### Safe Output Jobs - External Integration

| Job Name | Description | Context |
|----------|-------------|---------|
| `post_to_slack_channel` | Posts message to Slack | Safe-output job for Slack notifications |
| `post-issue` | Posts to external issue tracker | Safe-output job for external integration |

### Safe Output Jobs - Git Operations

| Job Name | Description | Context |
|----------|-------------|---------|
| `push_repo_memory` | Pushes memory to repository | Safe-output job for memory sync |
| `push_to_pull_request_branch` | Pushes commits to PR branch | Safe-output job for PR updates |

### Safe Output Jobs - Search & Assets

| Job Name | Description | Context |
|----------|-------------|---------|
| `search_issues` | Searches issues with criteria | Safe-output job for issue discovery |
| `upload_assets` | Uploads release assets | Safe-output job for asset management |

### Specialized Jobs

| Job Name | Description | Context |
|----------|-------------|---------|
| `ast_grep` | AST-based code pattern search | Uses ast-grep for code analysis |
| `check_external_user` | Checks if user is external contributor | Permission validation job |
| `generate-sbom` | Generates Software Bill of Materials | Security and compliance job |
| `release` | Performs release operations | Release automation job |
| `super_linter` | Runs super-linter on codebase | Code quality enforcement job |

## File Paths

Common file paths referenced in workflows (32 unique path patterns):

### Temporary Agent Paths

| Path | Description | Context |
|------|-------------|---------|
| `/tmp/gh-aw/` | Root temporary directory | Base path for all workflow temporary files |
| `/tmp/gh-aw/agent-stdio.log` | Agent standard I/O logs | Console output from agent execution |
| `/tmp/gh-aw/aw.patch` | Generated git patch file | Changes to be applied by safe-outputs |
| `/tmp/gh-aw/aw_info.json` | Workflow metadata | Information about workflow execution |
| `/tmp/gh-aw/aw-prompts/prompt.txt` | Agent prompt file | Full prompt sent to AI engine |

### Cache & Memory Paths

| Path | Description | Context |
|------|-------------|---------|
| `/tmp/gh-aw/cache-memory` | Cache memory storage | Persistent cache across runs |
| `/tmp/gh-aw/cache-memory-focus-areas` | Focus-based cache | Specialized memory storage |
| `/tmp/gh-aw/layout-cache` | Layout specification cache | Cache for layout patterns |
| `/tmp/gh-aw/prompt-cache` | Prompt template cache | Cached prompt templates |
| `/tmp/gh-aw/repo-memory-default` | Repository memory | Shared repo state |

### MCP & Tool Paths

| Path | Description | Context |
|------|-------------|---------|
| `/tmp/gh-aw/mcp-config/logs/` | MCP configuration logs | MCP server setup logs |
| `/tmp/gh-aw/mcp-logs/` | MCP execution logs | Runtime logs from MCP servers |
| `/tmp/gh-aw/safe-inputs/logs/` | Safe inputs logs | Logs from safe-inputs MCP server |

### Safe Output Paths

| Path | Description | Context |
|------|-------------|---------|
| `/tmp/gh-aw/safe-jobs/` | Safe job outputs | Outputs from safe-output jobs |
| `/tmp/gh-aw/safeoutputs/` | Safe outputs directory | Base directory for safe outputs |
| `/tmp/gh-aw/safeoutputs/assets/` | Safe output assets | Files needed by safe-output processing |

### Security & Firewall Paths

| Path | Description | Context |
|------|-------------|---------|
| `/tmp/gh-aw/sandbox/agent/logs/` | Agent sandbox logs | Sandboxed agent execution logs |
| `/tmp/gh-aw/sandbox/firewall/logs/` | Firewall sandbox logs | Firewall execution logs |
| `/tmp/gh-aw/redacted-urls.log` | Redacted URL log | URLs that were redacted for security |
| `/tmp/gh-aw/threat-detection/` | Threat detection directory | Security scanning outputs |
| `/tmp/gh-aw/threat-detection/detection.log` | Threat detection log | Security threat findings |

### Python & Data Paths

| Path | Description | Context |
|------|-------------|---------|
| `/tmp/gh-aw/python/*.py` | Python script files | Generated Python scripts |
| `/tmp/gh-aw/python/charts/*.png` | Generated chart images | Data visualization outputs |
| `/tmp/gh-aw/python/data/*` | Data files | Input/output data files |

### Playwright & Debug Paths

| Path | Description | Context |
|------|-------------|---------|
| `/tmp/gh-aw/playwright-debug-logs/` | Playwright debug logs | Browser automation debug output |

### SBOM & Quality Paths

| Path | Description | Context |
|------|-------------|---------|
| `sbom.cdx.json` | CycloneDX SBOM | Software Bill of Materials in CycloneDX format |
| `sbom.spdx.json` | SPDX SBOM | Software Bill of Materials in SPDX format |
| `super-linter.log` | Super linter logs | Linting results and errors |

### Environment Variable Paths

| Path | Description | Context |
|------|-------------|---------|
| `${{ env.GH_AW_AGENT_OUTPUT }}` | Dynamic agent output path | Environment variable for agent output location |
| `${{ env.GH_AW_SAFE_OUTPUTS }}` | Dynamic safe outputs path | Environment variable for safe outputs location |
| `${{ steps.create_code_scanning_alert.outputs.sarif_file }}` | Dynamic SARIF file path | Step output reference for SARIF location |

## Folder Patterns

Key directories used across the codebase:

### Workflow Directories

| Folder | Description | Context |
|--------|-------------|---------|
| `.github/workflows/` | Workflow files (source and compiled) | Primary location for all `.md` and `.lock.yml` files (118 workflows) |
| `.github/workflows/shared/` | Shared workflow components | Reusable workflow imports |
| `.github/aw/` | Agentic workflow configuration | Contains `actions-lock.json` and cache files |
| `.github/agents/` | Agent definition files | Custom agent markdown definitions |
| `.github/actions/` | Custom GitHub Actions | Local composite actions |

### Source Code Directories

| Folder | Description | Context |
|--------|-------------|---------|
| `pkg/workflow/` | Workflow compilation code | Go package for compiling workflows |
| `pkg/workflow/js/` | JavaScript runtime code | CommonJS modules for GitHub Actions (`.cjs` files) |
| `pkg/cli/` | CLI command implementations | gh-aw command handlers |
| `pkg/parser/` | Markdown frontmatter parsing | Schema validation and parsing |
| `pkg/parser/schemas/` | JSON schemas | Workflow validation schemas |
| `pkg/constants/` | Constants and configuration | Shared constants including artifact names |

### Documentation & Specs

| Folder | Description | Context |
|--------|-------------|---------|
| `specs/` | Specification documents | Documentation and specs directory |
| `docs/` | User documentation | Astro Starlight documentation site |
| `skills/` | AI skill definitions | Specialized knowledge for AI agents |

### Working Directories (in Actions)

| Folder | Description | Context |
|--------|-------------|---------|
| `./docs` | Documentation working directory | Used in documentation build workflows |
| `./pkg/workflow/js` | JavaScript working directory | Used in JavaScript linting/testing |

## Constants and Patterns

### Go Constants (from pkg/constants/constants.go)

**Artifact Names:**
- `SafeOutputArtifactName = "safe_output.jsonl"` - Safe outputs configuration artifact
- `AgentOutputArtifactName = "agent_output.json"` - Agent execution output artifact

**Job Names:**
- `AgentJobName = "agent"` - Main agent execution job
- `ActivationJobName = "activation"` - Workflow activation check job
- `PreActivationJobName = "pre_activation"` - Pre-activation validation job
- `DetectionJobName = "detection"` - Post-agent detection job

**Step IDs (Pre-Activation):**
- `CheckMembershipStepID = "check_membership"` - Team membership validation
- `CheckStopTimeStepID = "check_stop_time"` - Workflow stop time check
- `CheckSkipIfMatchStepID = "check_skip_if_match"` - Skip pattern matching
- `CheckCommandPositionStepID = "check_command_position"` - Command position validation

**MCP Server IDs:**
- `SafeOutputsMCPServerID = "safeoutputs"` - Safe outputs MCP server identifier
- `SafeInputsMCPServerID = "safeinputs"` - Safe inputs MCP server identifier
- `SafeInputsMCPVersion = "1.0.0"` - Safe inputs version

**Default Versions:**
- `DefaultCopilotVersion = "0.0.367"` - GitHub Copilot CLI version
- `DefaultClaudeCodeVersion = "2.0.62"` - Claude Code CLI version
- `DefaultCodexVersion = "0.66.0"` - OpenAI Codex CLI version
- `DefaultGitHubMCPServerVersion = "v0.24.1"` - GitHub MCP server version
- `DefaultFirewallVersion = "v0.6.0"` - Firewall (AWF) binary version
- `DefaultPlaywrightMCPVersion = "0.0.51"` - Playwright MCP package version
- `DefaultPlaywrightBrowserVersion = "v1.57.0"` - Playwright browser Docker version
- `DefaultMCPSDKVersion = "1.24.0"` - MCP SDK version
- `DefaultMCPRegistryURL = "https://api.mcp.github.com/v0"` - MCP registry URL

**Runtime Versions:**
- Node: `24`, Python: `3.12`, Go: `1.25`, Ruby: `3.3`, .NET: `8.0`
- Java: `21`, Bun: `1.1`, Deno: `2.x`, Elixir: `1.17`, Haskell: `9.10`

**Timeout Constants:**
- `DefaultAgenticWorkflowTimeout = 20 * time.Minute` - Workflow timeout
- `DefaultToolTimeout = 60 * time.Second` - Tool/MCP operation timeout
- `DefaultMCPStartupTimeout = 120 * time.Second` - MCP server startup timeout

**Runner Images:**
- `DefaultActivationJobRunnerImage = "ubuntu-slim"` - 1 vCPU runner for activation jobs

### JavaScript Patterns (from pkg/workflow/js/*.cjs)

**Path Construction:**
```javascript
const workflowMdFile = path.join(workspace, ".github", "workflows", `${workflowBasename}.md`);
const lockFile = path.join(workspace, ".github", "workflows", workflowFile);
```

**Artifact References:**
- Patch file available as artifact: `aw.patch`
- View run details: `${runUrl}` - Link to workflow run with artifacts
- Artifact directory: `process.env.ARTIFACT_DIR` - Environment variable for artifact location

**SARIF Creation:**
```javascript
artifactLocation: { uri: finding.file }
```

**Script Loading:**
```javascript
const scriptPath = path.join(process.cwd(), "script_name.cjs");
const utilsPath = path.join(import.meta.dirname, "utils.cjs");
```

## Usage Guidelines

### Artifact Naming
- Use descriptive hyphenated names (e.g., `agent-output`, `mcp-logs`)
- For per-workflow artifacts, use pattern: `{type}-{workflow-name}`
- Firewall logs follow pattern: `firewall-logs-{workflow-name}`
- Dynamic artifacts can include `${{ github.run_id }}` for uniqueness

### Job Naming
- Use snake_case for job names (e.g., `create_pull_request`)
- Prefix safe-output jobs with action verb: `create_`, `update_`, `add_`, `close_`, etc.
- Core jobs use simple names: `agent`, `activation`, `detection`, `conclusion`
- Pre-activation jobs use `pre_` prefix: `pre_activation`

### Path References
- Always use `/tmp/gh-aw/` as the base temporary directory
- Organize by purpose: `/tmp/gh-aw/{category}/`
- Use environment variables for dynamic paths: `${{ env.GH_AW_AGENT_OUTPUT }}`
- Relative paths from repository root: `.github/workflows/`, `pkg/workflow/`

### Action Pinning
- Always pin actions to full commit SHA for security
- Current pinned SHAs are from 2024-2025 timeframe
- Regular updates needed to maintain security patches

### Constants Usage
- Import from `pkg/constants/constants.go` for consistency
- Use semantic constants like `AgentOutputArtifactName` instead of hardcoded strings
- Version constants should be centrally managed
- Timeout constants use `time.Duration` for type safety

### Environment Variables
- `GH_AW_AGENT_OUTPUT` - Path to agent output file
- `GH_AW_SAFE_OUTPUTS` - Path to safe outputs directory
- `GH_AW_MODEL_AGENT_*` - Model configuration (Copilot, Claude, Codex)
- `GH_AW_MODEL_DETECTION_*` - Detection model configuration

## Statistics Summary

- **Total lock.yml files**: 118
- **Unique GitHub Actions**: 19
- **Unique upload artifact names**: 85+
- **Unique download artifact names**: 8 (core artifacts reused across workflows)
- **Unique job names**: 38
- **Unique file path patterns**: 32
- **Safe output job types**: 27
- **Total action uses across all workflows**: 4,374
- **Total artifact uploads across all workflows**: 1,041

## Safe Output Jobs Overview

The repository uses 27 different safe-output job types for controlled GitHub API interactions:

**Resource Creation** (6): create_pull_request, create_issue, create_discussion, create_pr_review_comment, create_agent_task, create_code_scanning_alert

**Resource Updates** (5): update_issue, update_pull_request, update_project, update_release, update_cache_memory

**Comments & Interaction** (4): add_comment, add_labels, hide_comment, notion_add_comment

**Assignment & Linking** (3): assign_to_agent, assign_to_user, link_sub_issue

**State Changes** (4): close_issue, close_pull_request, close_discussion, minimize_comment

**External Integration** (2): post_to_slack_channel, post-issue

**Git Operations** (2): push_repo_memory, push_to_pull_request_branch

**Search & Assets** (2): search_issues, upload_assets

## Firewall Log Artifacts

The repository has 60+ firewall log artifacts, one per workflow, following the pattern:
`firewall-logs-{workflow-name}`

This ensures each workflow's security logs are isolated and traceable to specific workflow executions.

---

*This document is automatically maintained by the Layout Specification Maintainer workflow.*
*For questions or issues, please refer to the workflow definition in `.github/workflows/`.*
