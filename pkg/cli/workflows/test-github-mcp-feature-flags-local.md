---
name: Test GitHub MCP Feature Flags Local
description: Test GitHub MCP Server feature flags in local (Docker) mode
on:
  workflow_dispatch:
permissions:
  contents: read
  actions: read
  issues: read
  pull-requests: read
engine: copilot
tools:
  github:
    mode: local
    toolsets: [default, actions]
    args:
      - "--features=consolidated-actions"
---

# Test GitHub MCP Feature Flags (Local Mode)

This test workflow demonstrates GitHub MCP Server v0.26.0+ feature flags in **local (Docker) mode**.

## Context

This workflow is identical to test-github-mcp-feature-flags.md but uses `mode: local` instead of `mode: remote`. This verifies that feature flags work in both deployment modes:

- **Remote mode**: Hosted GitHub MCP server at api.githubcopilot.com
- **Local mode**: Docker container running on GitHub Actions runner

## Task

Please use the GitHub MCP server (running in Docker) to:

1. **Verify Container Environment**:
   - The MCP server runs in a Docker container (ghcr.io/github/github-mcp-server:v0.26.3)
   - Feature flag `--features=consolidated-actions` is passed via args field
   - This demonstrates using CLI flags with Docker mode

2. **Test Actions Toolset**:
   - List workflows in this repository (githubnext/gh-aw)
   - Get the 3 most recent workflow runs for any workflow
   - Demonstrate actions toolset functionality

3. **Test Repository Access**:
   - Get repository information for githubnext/gh-aw
   - List the 5 most recent commits
   - Use default toolset features

4. **Compare Local vs Remote**:
   - Behavior should be identical to remote mode
   - Only difference is the server runs in Docker instead of hosted service

## Expected Behavior

- Docker container starts successfully with feature flags
- Actions toolset works identically to remote mode
- Default toolsets function normally
- Feature flag configuration is properly passed to container
- No compatibility issues between local and remote modes

## Success Criteria

- Successfully list workflows using actions toolset
- Successfully access repository data
- Workflow completes demonstrating feature parity with remote mode
- No Docker container startup or configuration errors
