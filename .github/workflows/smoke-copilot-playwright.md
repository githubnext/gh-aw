---
description: Smoke test workflow that validates Copilot engine functionality by reviewing recent PRs every 6 hours
on: 
  schedule:
    - cron: "0 0,6,12,18 * * *"  # Every 6 hours
  workflow_dispatch:
  pull_request:
    types: [labeled]
    names: ["smoke"]
  reaction: "eyes"
permissions:
  contents: read
  pull-requests: read
  issues: read
name: Smoke Copilot Playwright
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
  mcp:
    container: ghcr.io/githubnext/gh-aw-mcpg
    version: v0.0.9
    port: 8080
features:
  mcp-gateway: true
tools:
  cache-memory: true
  edit:
  bash:
    - "*"
  github:
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
      expires: 2h
    add-labels:
      allowed: [smoke-copilot]
    messages:
      footer: "> 📰 *BREAKING: Report filed by [{workflow_name}]({run_url})*"
      run-started: "📰 BREAKING: [{workflow_name}]({run_url}) is now investigating this {event_type}. Sources say the story is developing..."
      run-success: "📰 VERDICT: [{workflow_name}]({run_url}) has concluded. All systems operational. This is a developing story. 🎤"
      run-failure: "📰 DEVELOPING STORY: [{workflow_name}]({run_url}) reports {status}. Our correspondents are investigating the incident..."
timeout-minutes: 5
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
        find /tmp/gh-aw/playwright-debug-logs/ -maxdepth 1 -ls
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
    uses: actions/upload-artifact@v6
    with:
      name: playwright-debug-logs-${{ github.run_id }}
      path: /tmp/gh-aw/playwright-debug-logs/
      if-no-files-found: ignore
      retention-days: 7
---

# Smoke Test: Copilot Engine Validation

**IMPORTANT: Keep all outputs extremely short and concise. Use single-line responses where possible. No verbose explanations.**

## Test Requirements

1. **Playwright MCP Testing**: Use playwright to navigate to https://github.com and verify the page title contains "GitHub"
2. **Cache Memory Testing**: Write a test file to `/tmp/gh-aw/cache-memory/smoke-test-${{ github.run_id }}.txt` with content "Cache memory test for run ${{ github.run_id }}" and verify it was created successfully

**Safe Input gh Tool Testing**: Use the `safeinputs-gh` tool to run "gh issues list --limit 3" to verify the tool can access GitHub issues

## Output

Add a **very brief** comment (max 5-10 lines) to the current pull request with:
- ✅ or ❌ for each test result
- Overall status: PASS or FAIL

If all tests pass, add the label `smoke-copilot-playwright` to the pull request.