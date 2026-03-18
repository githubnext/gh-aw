---
on:
  issues:
    types: [opened]
permissions:
  contents: read

github-app:
  app-id: ${{ vars.APP_ID }}
  private-key: ${{ secrets.APP_PRIVATE_KEY }}
tools:
  github:
    mode: remote
    toolsets: [default]
safe-outputs:
  create-issue:
    title-prefix: "[automated] "
engine: copilot
---

# Top-Level GitHub App Fallback for GitHub MCP Tool

This workflow demonstrates using a top-level github-app as a fallback for tools.github
token minting operations.

The top-level `github-app` is automatically applied to the GitHub MCP tool configuration
when no `tools.github.github-app` is defined. This mints a GitHub App installation access
token for the GitHub MCP server to use when making API calls.
