---
on:
  workflow_dispatch:
permissions:
  contents: read
  issues: read
engine: claude
tools:
  github:
    allowed: [get_repository, list_issues]
    args: ["--verbose"]
---

# Test GitHub Args Configuration

This workflow tests the `args` field for GitHub MCP server configuration.

The workflow is configured with:
- `args: ["--verbose"]` for the GitHub tool

Please perform the following tasks:

1. Get information about the current repository using the GitHub tool
2. List the first 5 issues in the repository
3. Confirm that the GitHub MCP server is working correctly with the custom args

The args field allows passing additional command-line arguments to the GitHub MCP server for debugging or customization purposes.
