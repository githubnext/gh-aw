---
description: Smoke test workflow that validates Codex engine functionality by reviewing recent PRs twice daily
on: 
  schedule: every 12h
  workflow_dispatch:
  pull_request:
    types: [labeled]
    names: ["smoke"]
  reaction: "hooray"
permissions:
  contents: read
  issues: read
  pull-requests: read
name: Smoke Codex
engine: codex
strict: true
imports:
  - shared/gh.md
  - shared/mcp/tavily.md
  - shared/reporting.md
network:
  allowed:
    - defaults
    - github
    - playwright
tools:
  cache-memory: true
  github:
  playwright:
    allowed_domains:
      - github.com
  edit:
  bash:
    - "*"
  serena:
    languages:
      go: {}
runtimes:
  go:
    version: "1.25"
  node:
    version: "22"
steps:
  - name: Setup Node.js
    uses: actions/setup-node@v4
    with:
      node-version: "22"
sandbox:
  mcp:
    container: "ghcr.io/githubnext/gh-aw-mcpg"
safe-outputs:
    add-comment:
      hide-older-comments: true
    create-issue:
      expires: 2h
      close-older-issues: true
    add-labels:
      allowed: [smoke-codex]
    remove-labels:
      allowed: [smoke]
    hide-comment:
    messages:
      footer: "> 🔮 *The oracle has spoken through [{workflow_name}]({run_url})*"
      run-started: "🔮 The ancient spirits stir... [{workflow_name}]({run_url}) awakens to divine this {event_type}..."
      run-success: "✨ The prophecy is fulfilled... [{workflow_name}]({run_url}) has completed its mystical journey. The stars align. 🌟"
      run-failure: "🌑 The shadows whisper... [{workflow_name}]({run_url}) {status}. The oracle requires further meditation..."
timeout-minutes: 10
---

# Smoke Test: Codex Engine Validation

**IMPORTANT: Keep all outputs extremely short and concise. Use single-line responses where possible. No verbose explanations.**

## Test Requirements

1. **GitHub MCP Testing**: Review the last 2 merged pull requests in ${{ github.repository }}
2. **Safe Inputs GH CLI Testing**: Use the `safeinputs-gh` tool to query 2 pull requests from ${{ github.repository }} (use args: "pr list --repo ${{ github.repository }} --limit 2 --json number,title,author")
3. **Serena MCP Testing**: Use the Serena MCP server tool `activate_project` to initialize the workspace at `${{ github.workspace }}` and verify it succeeds (do NOT use bash to run go commands - use Serena's MCP tools)
4. **Playwright Testing**: Use playwright to navigate to https://github.com and verify the page title contains "GitHub"
5. **Tavily Web Search Testing**: Use the Tavily MCP server to perform a web search for "GitHub Agentic Workflows" and verify that results are returned with at least one item
6. **Runtime Mount Testing - npm**: Execute `npm ls` command to verify that npm is available in the container (this tests that the runtime manager properly mounted Node.js folders)
7. **Runtime Mount Testing - Docs Build**: Build the documentation by running `cd docs && npm ci && npm run build` to verify Node.js runtime is fully functional with all dependencies and build tools
8. **Runtime Mount Testing - JS Tests**: Run JavaScript tests with `cd actions/setup/js && npm ci && npm run test:js` to verify Node.js can execute tests and all mounts are working correctly
9. **File Writing Testing**: Create a test file `/tmp/gh-aw/agent/smoke-test-codex-${{ github.run_id }}.txt` with content "Smoke test passed for Codex at $(date)" (create the directory if it doesn't exist)
10. **Bash Tool Testing**: Execute bash commands to verify file creation was successful (use `cat` to read the file back)

## Output

Add a **very brief** comment (max 5-10 lines) to the current pull request with:
- PR titles only (no descriptions)
- ✅ or ❌ for each test result
- Overall status: PASS or FAIL

If all tests pass, add the label `smoke-codex` to the pull request.
