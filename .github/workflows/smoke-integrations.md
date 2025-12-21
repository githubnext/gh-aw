---
description: Integration smoke test workflow that validates Playwright, MCP servers, and browser automation
on: 
  schedule:
    - cron: "0 9,21 * * *"  # Twice daily at 9am and 9pm UTC
  workflow_dispatch:
    inputs:
      integration:
        description: 'Integration to test (all, playwright, mcp-servers)'
        required: false
        default: 'all'
        type: choice
        options:
          - all
          - playwright
          - mcp-servers
  pull_request:
    types: [labeled]
    names: ["smoke"]
permissions:
  contents: read
  pull-requests: read
  issues: read
name: Smoke Integrations
engine: 
  id: copilot
  env:
    DEBUG: "copilot:*"  # Enable copilot CLI debug logs
imports:
  - shared/gh.md
network:
  allowed:
    - defaults
    - node
    - github
    - playwright
    - clients2.google.com        # Chrome time sync
    - www.google.com             # Chrome services
    - accounts.google.com        # Chrome account checks
    - android.clients.google.com # Chrome internal
sandbox:
  agent: awf  # Firewall enabled
tools:
  cache-memory: true
  edit:
  bash:
    - "*"
  github:
    toolsets: [repos, pull_requests, issues]
  playwright:
    allowed_domains:
      - github.com
    args:
      - "--save-trace"  # Enable trace capture for debugging
  serena: ["go"]
safe-outputs:
  add-comment:
    hide-older-comments: true
  create-issue:
    expires: 1d
  add-labels:
    allowed: [smoke-integrations, smoke-playwright, smoke-mcp]
  messages:
    footer: "> 🔌 *Integration tested by [{workflow_name}]({run_url})*"
    run-started: "🔌 INTEGRATIONS: [{workflow_name}]({run_url}) testing integrations for {event_type}..."
    run-success: "✅ INTEGRATIONS: [{workflow_name}]({run_url}) PASSED. All integrations validated. 🎨"
    run-failure: "❌ INTEGRATIONS: [{workflow_name}]({run_url}) {status}. Integration issues detected..."
timeout-minutes: 15
strict: true
steps:
  # Pre-flight Docker container test for Playwright MCP
  - name: Pre-flight Playwright MCP Test
    run: |
      echo "🧪 Testing Playwright MCP Docker container startup..."
      
      # Pull the Playwright MCP Docker image
      echo "Pulling Playwright MCP Docker image..."
      docker pull mcr.microsoft.com/playwright/mcp
      
      # Test container startup with a simple healthcheck
      echo "Testing container startup..."
      timeout 30 docker run --rm -i mcr.microsoft.com/playwright/mcp --help || {
        echo "❌ Playwright MCP container failed to start"
        exit 1
      }
      
      echo "✅ Playwright MCP container pre-flight check passed"
post-steps:
  # Collect Playwright MCP logs after execution
  - name: Collect Playwright MCP Logs
    if: always()
    run: |
      echo "📋 Collecting Playwright MCP logs..."
      
      # Create logs directory
      mkdir -p /tmp/gh-aw/playwright-debug-logs
      
      # Copy any playwright logs from the MCP logs directory
      if [ -d "/tmp/gh-aw/mcp-logs/playwright" ]; then
        echo "Found Playwright MCP logs directory"
        cp -r /tmp/gh-aw/mcp-logs/playwright/* /tmp/gh-aw/playwright-debug-logs/ 2>/dev/null || true
        ls -la /tmp/gh-aw/playwright-debug-logs/
      else
        echo "No Playwright MCP logs directory found at /tmp/gh-aw/mcp-logs/playwright"
      fi
      
      # List all trace files if any
      echo "Looking for trace files..."
      find /tmp -name "*.zip" -o -name "trace*" 2>/dev/null | head -20 || true
      
      # Show docker container logs if any containers are still running
      echo "Checking for running Docker containers..."
      docker ps -a --format "table {{.Names}}\t{{.Status}}\t{{.Image}}" 2>/dev/null || true
  - name: Upload Playwright Debug Logs
    if: always()
    uses: actions/upload-artifact@v5
    with:
      name: playwright-debug-logs-${{ github.run_id }}
      path: /tmp/gh-aw/playwright-debug-logs/
      if-no-files-found: ignore
      retention-days: 7
---

# Smoke Test: Integration Validation

**IMPORTANT: Keep all outputs extremely short and concise. Use single-line responses where possible. No verbose explanations.**

This workflow validates integration functionality including Playwright browser automation, MCP servers, and external tool integration.

## Integration Selection

Current integration: Determined by workflow_dispatch input (default: all)

This workflow uses the Copilot engine with Playwright and MCP server integrations. When triggered via `workflow_dispatch` with the `integration` input parameter, the AI agent will be instructed to focus on the specified integration(s):
- `all` - Validate all integrations (Playwright + MCP servers)
- `playwright` - Validate Playwright browser automation
- `mcp-servers` - Validate MCP server integrations

**Note:** The workflow configuration includes all integration tools. The `integration` input controls which test scenarios to prioritize.

## Integration Test Requirements

### Playwright Integration Tests

When testing **playwright** integration:

1. **Browser Navigation**: Use playwright to navigate to https://github.com
2. **Page Verification**: Verify the page title contains "GitHub"
3. **Container Health**: Confirm Playwright Docker container started successfully (validated in pre-flight step)
4. **Trace Capture**: Verify trace capture is enabled with `--save-trace` arg
5. **Network Access**: Confirm allowed domains (github.com) are accessible
6. **Chrome Services**: Verify Chrome-related domains are accessible for browser functionality

### MCP Server Integration Tests

When testing **mcp-servers** integration:

1. **GitHub MCP Server**: Use the GitHub MCP server to list recent pull requests
2. **Toolsets Validation**: Confirm repos, pull_requests, and issues toolsets are available
3. **MCP Communication**: Verify MCP server communication works through Docker
4. **Tool Execution**: Execute at least one tool from each toolset successfully
5. **Error Handling**: Confirm proper error handling if a tool fails

### Common Integration Tests

1. **Cache Memory**: Write a test file to `/tmp/gh-aw/cache-memory/smoke-test-${{ github.run_id }}.txt` with content "Integration test for run ${{ github.run_id }}"
2. **Safe Input gh Tool**: Use the `safeinputs-gh` tool to run "gh issues list --limit 3" to verify GitHub CLI integration
3. **Bash Integration**: Execute multi-step bash commands to verify shell integration
4. **File Operations**: Create and read test files to verify filesystem access

## Multi-Integration Testing

If the `integration` input is set to `all` (or triggered without specifying an integration), perform tests for all integrations:

1. **Playwright Integration** - Test browser automation
2. **MCP Server Integration** - Test MCP communication

For each integration, report results separately.

## Output

Add a **very brief** comment (max 10-15 lines) to the current pull request with:

### Format:
```
## Smoke Integrations Test Results - Run ${{ github.run_id }}

**Integration(s) Tested:** [integration list]

| Integration | Component | Result |
|------------|-----------|--------|
| Playwright | Browser Nav | ✅/❌ |
| Playwright | Container | ✅/❌ |
| Playwright | Traces | ✅/❌ |
| MCP Servers | GitHub MCP | ✅/❌ |
| MCP Servers | Toolsets | ✅/❌ |

**Overall Status:** PASS/FAIL

**Artifacts:**
- Playwright logs: [uploaded/not found]
- Trace files: [available/not available]
```

If all tests pass, add the label `smoke-integrations` to the pull request. For specific integrations, also add integration-specific labels (`smoke-playwright` or `smoke-mcp`).
