---
private: true
emoji: "🔢"
description: Monitors and updates agentic CLI tools (Claude Code, GitHub Copilot CLI, OpenAI Codex, GitHub MCP Server, Playwright MCP, Playwright CLI, Playwright Browser, MCP Gateway, Pi) for new versions
on:
  schedule: daily
  workflow_dispatch:
max-daily-ai-credits: 10000
permissions:
  contents: read
  pull-requests: read
  issues: read
strict: false
engine: claude
network: 
   allowed: [defaults, node, go, "api.github.com", "ghcr.io"]
imports:
  - ../skills/jqschema/SKILL.md
  - shared/reporting.md
  - shared/otlp.md
sandbox:
  agent:
    sudo: false
tools:
  cli-proxy: true
  web-fetch:
  cache-memory: true
  bash:
    - "*"
  edit:
safe-outputs:
  create-issue:
    expires: 2d
    title-prefix: "[ca] "
    labels: [automation, dependencies, cookie]
    close-older-issues: true
timeout-minutes: 45
features:
  gh-aw-detection: true
---

# CLI Version Checker

Monitor and update agentic CLI tools: Claude Code, GitHub Copilot CLI, OpenAI Codex, GitHub MCP Server, Playwright MCP, Playwright CLI, Playwright Browser, MCP Gateway, and Pi.

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
  - No public GitHub repository
- **Copilot CLI**: Use `npm view @github/copilot version`
  - Repository: https://github.com/github/copilot-cli
  - **CRITICAL**: Always attempt to fetch and deeply analyze Copilot repository content
  - Release Notes: https://github.com/github/copilot-cli/releases
  - Changelog: https://github.com/github/copilot-cli/blob/main/CHANGELOG.md (or similar)
  - README: https://github.com/github/copilot-cli/blob/main/README.md
- **Codex**: Use `npm view @openai/codex version`
  - Repository: https://github.com/openai/codex
  - Release Notes: https://github.com/openai/codex/releases
- **GitHub MCP Server**: `https://api.github.com/repos/github/github-mcp-server/releases/latest`
  - Release Notes: https://github.com/github/github-mcp-server/releases
- **Playwright MCP**: Use `npm view @playwright/mcp version`
  - Repository: https://github.com/microsoft/playwright
  - Package: https://www.npmjs.com/package/@playwright/mcp
- **Playwright CLI**: Use `npm view @playwright/cli version`
  - Repository: https://github.com/microsoft/playwright-cli
  - Package: https://www.npmjs.com/package/@playwright/cli
- **Playwright Browser**: `https://api.github.com/repos/microsoft/playwright/releases/latest`
  - Release Notes: https://github.com/microsoft/playwright/releases
  - Docker Image: `mcr.microsoft.com/playwright:v{VERSION}`
- **MCP Gateway**: `https://api.github.com/repos/github/gh-aw-mcpg/releases/latest`
  - Repository: https://github.com/github/gh-aw-mcpg
  - Release Notes: https://github.com/github/gh-aw-mcpg/releases
  - Docker Image: `ghcr.io/github/gh-aw-mcpg:v{VERSION}`
  - Used as default sandbox.agent container (see `pkg/constants/constants.go`)
- **Pi**: Use `npm view @earendil-works/pi-coding-agent version`
  - Package: https://www.npmjs.com/package/@earendil-works/pi-coding-agent
  - Constant: `DefaultPiVersion` in `pkg/constants/version_constants.go`
**Optimization**: Fetch all versions in parallel using multiple npm view or WebFetch calls in a single turn.

### Research & Analysis
For each update, analyze intermediate versions:
- Categorize changes: Breaking, Features, Fixes, Security, Performance
- Assess impact on gh-aw workflows
- Document migration requirements
- Assign risk level (Low/Medium/High)

**GitHub Release Notes (when available)**:
- **Codex**: Fetch release notes from https://github.com/openai/codex/releases/tag/rust-v{VERSION}
  - Parse the "Highlights" section for key changes
  - Parse the "PRs merged" or "Merged PRs" section for detailed changes
  - **CRITICAL**: Convert PR/issue references (e.g., `#6211`) to full URLs since they refer to external repositories (e.g., `https://github.com/openai/codex/pull/6211`)
- **GitHub MCP Server**: Fetch release notes from https://github.com/github/github-mcp-server/releases/tag/v{VERSION}
  - Parse release body for changelog entries
  - **CRITICAL**: Convert PR/issue references (e.g., `#1105`) to full URLs since they refer to external repositories (e.g., `https://github.com/github/github-mcp-server/pull/1105`)
- **Playwright Browser**: Fetch release notes from https://github.com/microsoft/playwright/releases/tag/v{VERSION}
  - Parse release body for changelog entries
  - **CRITICAL**: Convert PR/issue references to full URLs (e.g., `https://github.com/microsoft/playwright/pull/12345`)
- **Copilot CLI**: **ALWAYS attempt deep analysis** - Repository: https://github.com/github/copilot-cli
  - **CRITICAL**: Thoroughly read and analyze all available documentation:
    1. **Release Notes**: Fetch from https://github.com/github/copilot-cli/releases/tag/v{VERSION}
       - Parse release highlights and feature descriptions
       - Extract breaking changes and deprecation notices
       - Note new commands, flags, and configuration options
    2. **CHANGELOG.md**: Read from https://github.com/github/copilot-cli/blob/main/CHANGELOG.md (or equivalent)
       - Compare versions to identify all changes between current and new version
       - Categorize changes: Breaking, Features, Fixes, Security, Performance
    3. **README.md**: Review https://github.com/github/copilot-cli/blob/main/README.md
       - Check for updated usage patterns and examples
       - Note new capabilities or configuration options
    4. **Documentation Changes**: Look for changes in documentation files that indicate new features
  - If repository is inaccessible (private), document the access limitation in the issue but still:
    - Use `npm view @github/copilot --json` for detailed package metadata
    - Compare CLI help output between versions (see "Tool Installation & Discovery" section)
    - Check for any publicly available release announcements or blog posts
  - **CRITICAL**: Convert PR/issue references to full URLs (e.g., `https://github.com/github/copilot-cli/pull/123`)
- **Claude Code**: No public repository, rely on NPM metadata and CLI help output
- **Playwright MCP**: Uses Playwright versioning, check NPM package metadata for changes
- **Playwright CLI**: Check NPM package metadata and GitHub releases for changes
  - Fetch release notes from https://github.com/microsoft/playwright-cli/releases/tag/v{VERSION}
  - **CRITICAL**: Convert PR/issue references to full URLs (e.g., `https://github.com/microsoft/playwright-cli/pull/123`)
- **MCP Gateway**: Fetch release notes from https://github.com/github/gh-aw-mcpg/releases/tag/{VERSION}
  - Parse release body for changelog entries
  - **CRITICAL**: Convert PR/issue references to full URLs (e.g., `https://github.com/github/gh-aw-mcpg/pull/123`)
  - Note: Used as default sandbox.agent container in MCP Gateway configuration
- **Pi**: No public GitHub repository; rely on NPM metadata and CLI help output
  - Use `npm view @earendil-works/pi-coding-agent --json` for package metadata
  - Compare CLI help output between versions
**NPM Metadata Fallback**: When GitHub release notes are unavailable, use:
- `npm view <package> --json` for package metadata
- Compare CLI help outputs between versions
- Check for version changelog in package description

### Tool Installation & Discovery
Check cache-memory first (`/tmp/gh-aw/cache-memory/`). Only install and run `--help` if the version changed; then save outputs to cache.

For each CLI tool update, install (`npm install -g <package>@<version>`), run `--help` on the main command and key subcommands (Copilot: `config`, `environment`), and compare with the cached output to identify new flags, removed features, or behavior changes.

### Update Process
1. Edit `./pkg/constants/constants.go` with new version(s)
2. **REQUIRED**: Run `make recompile` in the **foreground** — do NOT background it with `&` or follow it with `sleep`. Wait for it to finish completely before proceeding. Example: `make recompile && echo "done"`.
3. Verify changes with `git status`
4. **REQUIRED**: Create issue via safe-outputs with detailed analysis (do NOT skip this step)

## Issue Format

**Follow the Report Structure Pattern defined in `shared/reporting.md`.**

For each updated CLI, include: version old → new, release timeline, changes categorized as Breaking/Features/Fixes/Security/Performance, impact assessment, changelog links, and any CLI/subcommand changes discovered via help output.

**Important**: Use h3 (###) or lower for all headers. Wrap full changelogs in `<details>` tags. Use plain URLs (no backticks) and convert `#1234` PR references to full external URLs like `https://github.com/owner/repo/pull/1234`.

## Guidelines
- Only update stable versions (no pre-releases)
- Prioritize security updates
- Document all intermediate versions
- **USE NPM COMMANDS**: Use `npm view` instead of web-fetch for package metadata queries
- **CHECK CACHE FIRST**: Before re-analyzing versions, check cache-memory for recent results
- **PARALLEL FETCHING**: Fetch all versions in parallel using multiple npm/WebFetch calls in one turn
- **EARLY EXIT**: If no version changes detected, save check timestamp to cache and exit successfully
- **FETCH GITHUB RELEASE NOTES**: For tools with public GitHub repositories, fetch release notes to get detailed changelog information
  - Codex: Always fetch from https://github.com/openai/codex/releases
  - GitHub MCP Server: Always fetch from https://github.com/github/github-mcp-server/releases
  - Playwright Browser: Always fetch from https://github.com/microsoft/playwright/releases
  - MCP Gateway: Always fetch from https://github.com/github/gh-aw-mcpg/releases
  - Copilot CLI: Try to fetch, but may be inaccessible (private repo)
  - Playwright MCP: Check NPM metadata, uses Playwright versioning
  - Playwright CLI: Fetch from https://github.com/microsoft/playwright-cli/releases
  - Pi: No public GitHub repository; rely on NPM metadata (`npm view @earendil-works/pi-coding-agent --json`)
- **EXPLORE SUBCOMMANDS**: Install and test CLI tools to discover new features via `--help` and explore each subcommand
  - For Copilot CLI, explicitly check: `config`, `environment` and any other available subcommands
  - Use commands like `copilot help <subcommand>` or `<tool> <subcommand> --help`
- Compare help output between old and new versions (both main help and subcommand help)
- **SAVE TO CACHE**: Store help outputs (main and all subcommands) and version check results in cache-memory
- **REQUIRED**: Always run `make recompile` in the **foreground** (not backgrounded) after updating constants — wait for completion before proceeding
- **DO NOT COMMIT** `*.lock.yml` or `pkg/workflow/js/*.js` files directly

## JSON Parsing Tips

Filter stderr and use jq to avoid Unicode token errors from npm output:
```bash
npm view @github/copilot --json 2>/dev/null | jq -r '.version'
```

## Error Handling
- **SAVE PROGRESS**: Before exiting on errors, save current state to cache-memory
- **RESUME ON RESTART**: Check cache-memory on startup to resume from where you left off
- Retry NPM registry failures once after 30s
- Continue if individual changelog fetch fails
- Skip PR creation if recompile fails
- Exit successfully if no updates found
- Document incomplete research if rate-limited

{{#runtime-import shared/noop-reminder.md}}
