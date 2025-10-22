---
on:
  schedule:
    - cron: "0 15 * * *"  # Daily at 3 PM UTC
  workflow_dispatch:
permissions:
  contents: read
  actions: read
engine: claude
network: 
   allowed: [defaults, "registry.npmjs.org", "api.github.com", "ghcr.io"]
imports:
  - shared/jqschema.md
  - shared/check-existing-pr.md
tools:
  web-fetch:
  bash:
    - "cat *"
    - "ls *"
    - "grep *"
    - "git *"
    - "make *"
    - "npm install *"
    - "claude-code --help"
    - "copilot --help"
    - "codex --help"
  edit:
safe-outputs:
  create-issue:
    title-prefix: "[ca] "
    labels: [automation, dependencies]
    assignees: copilot
timeout_minutes: 15
strict: true
---

# CLI Version Checker

Monitor and update agentic CLI tools: Claude Code, GitHub Copilot CLI, OpenAI Codex, and GitHub MCP Server.

**Repository**: ${{ github.repository }} | **Run**: ${{ github.run_id }}

## Process

### 0. Check for Existing Pull Request
Before starting version checks and updates, verify if there's already an open PR from this workflow:
- Use `search_pull_requests` with query: `repo:${{ github.repository }} is:pr is:open "[ca]" label:automation label:dependencies`
- If an existing open PR is found:
  - Print message: "An open pull request already exists for CLI version updates"
  - Include PR number, URL, and title
  - Stop execution immediately
- If no existing PR found, proceed with version checks

### 1. Check for Updates
For each CLI/MCP server:
1. Fetch latest version from NPM registry or GitHub releases
2. Compare with current version in `./pkg/constants/constants.go`
3. If newer version exists, research changes and prepare update

### Version Sources
- **Claude Code**: `https://registry.npmjs.org/@anthropic-ai/claude-code/latest`
- **Copilot CLI**: `https://registry.npmjs.org/@github/copilot/latest`
- **Codex**: `https://registry.npmjs.org/@openai/codex/latest`
- **GitHub MCP Server**: `https://api.github.com/repos/github/github-mcp-server/releases/latest`

### Research & Analysis
For each update, analyze intermediate versions:
- Categorize changes: Breaking, Features, Fixes, Security, Performance
- Assess impact on gh-aw workflows
- Document migration requirements
- Assign risk level (Low/Medium/High)

### Tool Installation & Discovery
For each CLI tool update:
1. Install the new version globally:
   - Claude Code: `npm install -g @anthropic-ai/claude-code@<version>`
   - Copilot CLI: `npm install -g @github/copilot@<version>`
   - Codex: `npm install -g @openai/codex@<version>`
2. Invoke help to discover commands and flags:
   - Run `claude-code --help`
   - Run `copilot --help`
   - Run `codex --help`
3. Compare help output with previous version to identify:
   - New commands or subcommands
   - New command-line flags or options
   - Deprecated or removed features
   - Changed default behaviors

### Update Process
1. Edit `./pkg/constants/constants.go` with new version(s)
2. Run `make recompile` to update workflows
3. Verify changes with `git status`
4. Create issue via safe-outputs with detailed analysis

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
- Install and test CLI tools to discover new features via `--help`
- Compare help output between old and new versions
- Test with `make recompile` before creating PR
- **DO NOT COMMIT** `*.lock.yml` or `pkg/workflow/js/*.js` files directly

## Error Handling
- Retry NPM registry failures once after 30s
- Continue if individual changelog fetch fails
- Skip PR creation if recompile fails
- Exit successfully if no updates found
- Document incomplete research if rate-limited
