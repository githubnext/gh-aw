---
name: Test GitHub MCP Feature Flags
description: Test workflow demonstrating GitHub MCP Server feature flag support (v0.26.0+)
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
    mode: remote
    toolsets: [default, actions]
---

# Test GitHub MCP Feature Flags

This test workflow demonstrates GitHub MCP Server v0.26.0+ feature flag support.

## Context

GitHub MCP Server v0.26.0+ introduced feature flag infrastructure. Feature flags can be enabled via:
1. `--features` CLI flag (when running server directly or via Docker args)
2. `GITHUB_MCP_FEATURES` environment variable (when server supports it)

**Note**: In remote mode, feature flags are typically not needed as the hosted server has all features enabled. This workflow demonstrates that toolsets work correctly regardless of feature flag configuration.

## Task

Please use the GitHub MCP server to demonstrate that feature flags work correctly:

1. **Verify GitHub MCP Server Version**:
   - The workflow uses the default GitHub MCP Server (v0.26.3)
   - Feature flags were introduced in v0.26.0

2. **Test Actions Toolset**:
   - List workflows in this repository (githubnext/gh-aw)
   - Get the 3 most recent workflow runs
   - Use the actions toolset to access workflow information
   
3. **Test Basic Repository Access**:
   - Get repository information for githubnext/gh-aw
   - List the 3 most recent commits on main branch
   - Show that default toolsets work alongside feature flags

4. **Verify Feature Flag Configuration**:
   - Confirm that `GITHUB_MCP_FEATURES` is set in the environment
   - Demonstrate that both standard and feature-flagged functionality works

## Expected Behavior

- Actions toolset tools (list_workflows, list_workflow_runs) should be available
- Default toolset tools (get_repository, list_commits) should work normally
- Feature flags should not break backward compatibility
- Octicon icons should be present for supporting clients (transparent to workflow execution)

## Success Criteria

- Successfully list workflows using actions toolset
- Successfully access repository data using default toolset
- No errors related to feature flag configuration
- Workflow completes successfully demonstrating both standard and feature-flagged tools
