---
on:
  issues:
    types: [opened]

permissions:
  contents: write
  issues: write
  pull-requests: write
  
engine: copilot

features:
  mcp-gateway: true

sandbox:
  agent: awf
  mcp:
    container: "ghcr.io/lacox_microsoft/flowguard:stdin-config"
    port: 8000
    api-key: "${{ secrets.MCP_GATEWAY_API_KEY }}"

tools:
  github:
    github-token: "${{ secrets.GH_TOKEN }}"
    read-only: false
    toolsets: [repos, issues, pull_requests]

  playwright:

safe-outputs:
  add-comment:
    max: 3
---

# Test MCP Gateway Integration

This workflow tests the MCP Gateway feature by:
1. Starting an MCP Gateway container
2. Health checking the gateway
3. Routing GitHub and Playwright MCP calls through the gateway
4. Adding a comment to the issue

Please analyze this issue and add a helpful comment.
