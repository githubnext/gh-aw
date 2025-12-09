# GitHub Actions Workflow Layout Specification

> Auto-generated specification documenting patterns used in compiled `.lock.yml` files.
> Last updated: 2025-12-09

## Overview

This document catalogs all file paths, folder names, artifact names, and other patterns used across our compiled GitHub Actions workflows (`.lock.yml` files). The repository contains **114 compiled workflow files** that collectively use dozens of GitHub Actions, artifacts, and file path patterns.

## GitHub Actions

Common GitHub Actions used across workflows (sorted by frequency of use):

| Action | Version/SHA | Description | Context |
|--------|-------------|-------------|---------|
| `actions/checkout` | `93cb6efe18208431cddfb8368fd83d5badbf9bfd` | Checks out repository code | Used in almost all workflows for accessing repo content |
| `actions/upload-artifact` | `330a01c490aca151604b8cf639adc76d48f6c5d4` | Uploads build artifacts | Used for agent outputs, patches, prompts, and logs |
| `actions/download-artifact` | `018cc2cf5baa6db3ef3c5f8a56943fffe632ef53` | Downloads artifacts from previous jobs | Used in safe-output jobs and conclusion jobs |
| `actions/github-script` | `60a0d83039c74a4aee543508d2ffcb1c3799cdea` | Runs GitHub API scripts (v7) | Used for GitHub API interactions |
| `actions/github-script` | `ed597411d8f924073f98dfc5c65a23a2325f34cd` | Runs GitHub API scripts (v8) | Used for GitHub API interactions |
| `actions/setup-node` | `395ad3262231945c25e8478fd5baf05154b1d79f` | Sets up Node.js environment | Used in workflows requiring npm/node |
| `actions/cache` | `0057852bfaa89a56745cba8c7296529d2fc39830` | Cache dependencies and build outputs | Used for caching agent memory and dependencies |
| `actions/cache/restore` | `0057852bfaa89a56745cba8c7296529d2fc39830` | Restore cached files | Used for restoring specific cache entries |
| `actions/cache/save` | `0057852bfaa89a56745cba8c7296529d2fc39830` | Save files to cache | Used for saving specific cache entries |
| `actions/setup-python` | `a26af69be951a213d495a4c3e4e4022e16d87065` | Sets up Python environment | Used in workflows requiring Python tools |
| `actions/setup-go` | `d35c59abb061a4a6fb18e82ac0862c26744d6ab5` | Sets up Go environment | Used in workflows requiring Go tools |
| `actions/ai-inference` | `b81b2afb8390ee6839b494a404766bef6493c7d9` | AI inference action | Used for AI/ML workflows |
| `actions/create-github-app-token` | `29824e69f54612133e76f7eaac726eef6c875baf` | Creates GitHub App installation token | Used for workflows requiring GitHub App authentication |
| `anchore/sbom-action` | `fbfd9c6c189226748411491745178e0c2017392d` | Generates Software Bill of Materials | Used for dependency tracking and security |
| `astral-sh/setup-uv` | `e58605a9b6da7c637471fab8847a5e5a6b8df081` | Sets up UV Python package manager | Used in Python workflows |

## Artifact Names

Artifacts uploaded/downloaded between workflow jobs (sorted alphabetically):

| Name | Description | Context |
|------|-------------|---------|
| `agent-stdio.log` | Standard input/output log from agent | Debug logs from agent execution |
| `agent_output.json` | AI agent execution output | Contains the agent's response and analysis |
| `agent_outputs` | Agent output data | General agent output directory/artifact |
| `aw.patch` | Git patch file for changes | Used by create-pull-request safe-output |
| `aw_info.json` | Workflow metadata | Contains workflow run information |
| `cache-memory` | Agent memory cache | Persistent memory for agent across runs |
| `cache-memory-focus-areas` | Focused cache areas | Specialized cache for specific topics |
| `code-scanning-alert.sarif` | SARIF security scan results | Code security scanning output |
| `data-charts` | Generated data visualizations | Charts and graphs from data analysis |
| `firewall-logs-*` | Firewall log files | Logs from network firewall per workflow (e.g., `firewall-logs-ai-moderator`, `firewall-logs-archie`) |
| `mcp-logs` | MCP server logs | Debug logs from Model Context Protocol servers |
| `prompt` | Agent prompt content | Stored for debugging and audit purposes |
| `safe-outputs-config` | Safe outputs configuration | Passed from agent to safe-output jobs |

## Common Job Names

Standard job names across workflows (sorted alphabetically):

| Job Name | Description | Context |
|----------|-------------|---------|
| `activation` | Determines if workflow should run | Uses skip-if-match and other filters to decide execution |
| `add_comment` | Adds comment to issue/PR | Safe-output job for commenting |
| `add_labels` | Adds labels to issues/PRs | Safe-output job for labeling |
| `agent` | Main AI agent execution job | Runs the copilot/claude/codex engine |
| `assign_to_agent` | Assigns issue to agent | Safe-output job for assignment |
| `assign_to_user` | Assigns issue to user | Safe-output job for user assignment |
| `ast_grep` | AST-based code search | Job using ast-grep for pattern matching |
| `check_external_user` | Checks if user is external | Validation job for user permissions |
| `close_discussion` | Closes GitHub discussion | Safe-output job for closing discussions |
| `close_issue` | Closes GitHub issue | Safe-output job for closing issues |
| `close_pull_request` | Closes pull request | Safe-output job for closing PRs |
| `conclusion` | Final status reporting job | Runs after all other jobs complete |
| `create_agent_task` | Creates agent task | Safe-output job for task creation |
| `create_code_scanning_alert` | Creates code scanning alert | Safe-output job for security alerts |
| `create_discussion` | Creates GitHub discussion | Safe-output job for discussion creation |
| `create_issue` | Creates GitHub issue | Safe-output job for issue creation |
| `create_pr_review_comment` | Creates PR review comment | Safe-output job for review comments |
| `create_pull_request` | Creates PR from agent changes | Safe-output job for PR creation |
| `detection` | Post-agent analysis job | Analyzes agent output for patterns |
| `generate-sbom` | Generates Software Bill of Materials | Security and dependency tracking |

## File Paths

Common file paths referenced in workflows:

| Path | Description | Context |
|------|-------------|---------|
| `.github/workflows/` | Workflow definition directory | Contains all .md and .lock.yml workflow files |
| `.github/workflows/shared/` | Shared workflow components | Reusable workflow imports and templates |
| `.github/aw/` | Agentic workflow configuration | Contains actions-lock.json and cache files |
| `.github/aw/actions-lock.json` | Pinned GitHub Actions versions | Maintains SHA pins for all actions |
| `.github/aw/cache.json` | Action cache metadata | Cache file for action resolution |
| `pkg/workflow/` | Workflow compilation code | Go package for compiling workflows |
| `pkg/workflow/js/` | JavaScript runtime code | CommonJS modules for GitHub Actions |
| `pkg/cli/` | CLI command implementations | gh-aw command handlers |
| `pkg/parser/` | Markdown frontmatter parsing | Schema validation and parsing |
| `specs/` | Specification documents | Documentation and specs directory |
| `/tmp/gh-aw/` | Temporary workflow data | Runtime temporary directory for agent execution |

## Folder Patterns

Key directories used across the codebase:

| Folder | Description | Context |
|--------|-------------|---------|
| `.github/workflows/` | Workflow files (source and compiled) | Primary location for all workflows |
| `.github/workflows/shared/` | Shared workflow components | Reusable workflow imports |
| `.github/workflows/shared/mcp/` | MCP server workflows | MCP server configurations |
| `.github/workflows/tests/` | Test workflows | Workflow test cases |
| `.github/aw/` | Agentic workflow data | Configuration and cache storage |
| `pkg/cli/` | CLI command implementations | gh-aw command handlers |
| `pkg/parser/` | Markdown frontmatter parsing | Schema validation and parsing |
| `pkg/workflow/` | Workflow compilation engine | Core workflow compiler |
| `pkg/workflow/js/` | JavaScript bundles | MCP servers, safe-output handlers |
| `pkg/workflow/data/` | Embedded data files | Action pins and other data |
| `pkg/workflow/prompts/` | Agent prompts | System prompts for AI engines |
| `specs/` | Specifications | Documentation and specs |
| `docs/` | Documentation site | Astro Starlight documentation |

## Constants and Patterns from Source Code

### Go Constants

**Cache paths** (from `pkg/workflow/*.go`):
```go
".github/aw/cache.json"  // Action cache file
"/tmp/gh-aw/"            // Temporary workflow directory
```

**Artifact patterns** (referenced in Go code):
- `agent_output.json` - Main agent output
- `aw.patch` - Git patch for changes
- `safe-outputs-config` - Safe output configuration
- `aw_info.json` - Workflow metadata

### JavaScript Patterns

**MCP server files** (in `pkg/workflow/js/`):
- `safe_outputs_mcp_server.cjs` - Safe outputs MCP server
- `safe_inputs_mcp_server.cjs` - Safe inputs MCP server
- `safe_outputs_bootstrap.cjs` - Bootstrap configuration
- `safe_outputs_handlers.cjs` - Safe output handlers
- `safe_outputs_tools_loader.cjs` - Tools loader
- `messages_footer.cjs` - Message footer templates
- `log_parser_shared.cjs` - Log parsing utilities

**Common patterns in JavaScript**:
- `process.env.GITHUB_WORKSPACE` - Repository root path
- `process.env.RUNNER_TEMP` - Temporary directory
- `core.info()`, `core.warning()`, `core.error()` - GitHub Actions logging
- `core.setOutput()`, `core.getInput()` - GitHub Actions I/O

## Usage Guidelines

### Naming Conventions

- **Artifacts**: Use descriptive hyphenated names (e.g., `agent-output`, `mcp-logs`, `firewall-logs-{workflow}`)
- **Jobs**: Use snake_case for job names (e.g., `create_pull_request`, `add_comment`)
- **Paths**: Use relative paths from repository root
- **Workflow IDs**: Use kebab-case for tracker-id (e.g., `layout-spec-maintainer`)

### Action Pinning

- **Always pin to full commit SHA** for security and reproducibility
- Use `actions-lock.json` to manage action versions
- Update actions via `gh aw update` command

### Artifact Management

- Upload artifacts in agent/detection jobs
- Download artifacts in safe-output/conclusion jobs
- Use consistent naming across workflows
- Clean up artifacts after workflow completion

### File Path Conventions

- Store workflow files in `.github/workflows/`
- Place shared components in `.github/workflows/shared/`
- Keep configuration in `.github/aw/`
- Use `/tmp/gh-aw/` for runtime temporary files

---

*This document is automatically maintained by the Layout Specification Maintainer workflow.*
*Run `gh aw compile` to regenerate lock files after workflow changes.*
