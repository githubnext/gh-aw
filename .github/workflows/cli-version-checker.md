---
on:
  schedule:
    - cron: "0 15 * * *"  # Daily at 3 PM UTC
  workflow_dispatch:
permissions:
  contents: read
  actions: read
engine: copilot
network: 
   allowed: [defaults, node, "api.github.com", "ghcr.io"]
imports:
  - shared/jqschema.md
tools:
  web-fetch:
  cache-memory: true
  bash:
    - "*"
  edit:
safe-outputs:
  create-issue:
    title-prefix: "[ca] "
    labels: [automation, dependencies]
timeout_minutes: 15
---

# CLI Version Checker

Monitor and update agentic CLI tools: Claude Code, GitHub Copilot CLI, OpenAI Codex, and GitHub MCP Server.

**Repository**: ${{ github.repository }} | **Run**: ${{ github.run_id }}

## Process

**EFFICIENCY FIRST**: Before starting:
1. Check cache-memory at `/tmp/gh-aw/cache-memory/` for previous version checks and help outputs
2. If cached versions exist and are recent (< 24h), verify if updates are needed before proceeding
3. If no version changes detected, exit early with success

**CRITICAL**: If ANY version changes are detected, you MUST create an issue using safe-outputs.create-issue. Do not skip issue creation even for minor updates.

For each CLI/MCP server:
1. Fetch latest version from NPM registry or GitHub releases (use npm view commands for package metadata)
2. Compare with current version in `./pkg/constants/constants.go`
3. If newer version exists, research changes and prepare update

### Version Sources
- **Claude Code**: Use `npm view @anthropic-ai/claude-code version` (faster than web-fetch)
- **Copilot CLI**: Use `npm view @github/copilot version`
- **Codex**: Use `npm view @openai/codex version`
- **GitHub MCP Server**: `https://api.github.com/repos/github/github-mcp-server/releases/latest`

**Optimization**: Fetch all versions in parallel using multiple npm view or WebFetch calls in a single turn.

### Research & Analysis
For each update, analyze intermediate versions:
- Categorize changes: Breaking, Features, Fixes, Security, Performance
- Assess impact on gh-aw workflows
- Document migration requirements
- Assign risk level (Low/Medium/High)

### Tool Installation & Discovery
**CACHE OPTIMIZATION**: 
- Before installing, check cache-memory for previous help outputs
- Only install and run --help if version has changed
- Store help outputs in cache-memory at `/tmp/gh-aw/cache-memory/[tool]-[version]-help.txt`

For each CLI tool update:
1. Install the new version globally (skip if already installed from cache check):
   - Claude Code: `npm install -g @anthropic-ai/claude-code@<version>`
   - Copilot CLI: `npm install -g @github/copilot@<version>`
   - Codex: `npm install -g @openai/codex@<version>`
2. Invoke help to discover commands and flags (compare with cached output if available):
   - Run `claude-code --help`
   - Run `copilot --help`
   - Run `codex --help`
3. Compare help output with previous version to identify:
   - New commands or subcommands
   - New command-line flags or options
   - Deprecated or removed features
   - Changed default behaviors
4. Save new help output to cache-memory for future runs

### Update Process
1. Edit `./pkg/constants/constants.go` with new version(s)
2. Run `make recompile` to update workflows
3. Verify changes with `git status`
4. **REQUIRED**: Create issue via safe-outputs with detailed analysis (do NOT skip this step)

## Issue Format
Include for each updated CLI:
- **Version**: old → new (list intermediate versions if multiple)
- **Release Timeline**: dates and intervals
- **Changes**: Categorized as Breaking/Features/Fixes/Security/Performance
- **Impact Assessment**: Risk level, affected features, migration notes
- **Changelog Links**: NPM/GitHub release notes
- **CLI Changes**: New commands, flags, or removed features discovered via help

Template structure:
```
# Update [CLI Name]
- Previous: [version] → New: [version]
- Timeline: [dates and frequency]
- Breaking Changes: [list or "None"]
- New Features: [list]
- Bug Fixes: [list]
- Security: [CVEs/patches or "None"]
- CLI Discovery: [New commands/flags or "None detected"]
- Impact: Risk [Low/Medium/High], affects [features]
- Migration: [Yes/No - details if yes]
```

## Guidelines
- Only update stable versions (no pre-releases)
- Prioritize security updates
- Document all intermediate versions
- **USE NPM COMMANDS**: Use `npm view` instead of web-fetch for package metadata queries
- **CHECK CACHE FIRST**: Before re-analyzing versions, check cache-memory for recent results
- **PARALLEL FETCHING**: Fetch all versions in parallel using multiple npm/WebFetch calls in one turn
- **EARLY EXIT**: If no version changes detected, save check timestamp to cache and exit successfully
- Install and test CLI tools to discover new features via `--help`
- Compare help output between old and new versions
- **SAVE TO CACHE**: Store help outputs and version check results in cache-memory
- Test with `make recompile` before creating PR
- **DO NOT COMMIT** `*.lock.yml` or `pkg/workflow/js/*.js` files directly

## Error Handling
- **SAVE PROGRESS**: Before exiting on errors, save current state to cache-memory
- **RESUME ON RESTART**: Check cache-memory on startup to resume from where you left off
- Retry NPM registry failures once after 30s
- Continue if individual changelog fetch fails
- Skip PR creation if recompile fails
- Exit successfully if no updates found
- Document incomplete research if rate-limited
