---
name: Test GitHub MCP Octicon Icons
description: Demonstrates Octicon icon support in GitHub MCP Server v0.26.0+
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
    toolsets: [default, issues, pull_requests, actions]
---

# Test GitHub MCP Octicon Icons

This test workflow demonstrates Octicon icon support introduced in GitHub MCP Server v0.26.0.

## About Octicon Icons

Starting with v0.26.0, the GitHub MCP server automatically adds Octicon icons to all tools, resources, and prompts. These icons:

- **Appear in supporting clients**: VS Code MCP extension, Claude Desktop (with latest MCP support)
- **Are invisible to older clients**: Backward compatible - clients that don't support icons simply ignore them
- **Require no configuration**: Icons are added automatically by the server
- **Improve UI clarity**: Make it easier to identify GitHub-related capabilities at a glance

## Icons by Toolset

Different toolsets provide tools with different icons:

- **Repos toolset**: Repository, code, branch, commit icons
- **Issues toolset**: Issue, comment, label icons  
- **Pull Requests toolset**: PR, review, merge icons
- **Actions toolset**: Workflow, run, artifact icons

## Task

Please use the GitHub MCP server to access various toolsets and demonstrate tool availability. While we cannot directly see icons in workflow execution logs, this workflow verifies that:

1. **All toolsets work correctly** with icon metadata present (transparent to execution)
2. **Tools function normally** regardless of client icon support
3. **Icons don't break backward compatibility** with clients that don't support them

### Repository Operations
- Get repository information for githubnext/gh-aw
- List branches in the repository
- Get the 3 most recent commits

### Issue Operations
- List the 3 most recent open issues
- Get details for issue #1 if it exists
- Show issue comments for any issue

### Pull Request Operations
- List the 3 most recent pull requests (any state)
- Get details for the most recent PR
- List files changed in a PR if available

### Actions Operations
- List workflows in the repository
- Get the 3 most recent workflow runs
- Show workflow run details

## Icon Rendering Notes

**For Users with Supporting Clients**:
If you're viewing this workflow in VS Code with the MCP extension (v0.26.0+), you should see:
- 🏢 Repository icon for repo tools
- 🔧 Issue icon for issue tools
- 🔀 PR icon for pull request tools
- ⚙️ Actions icon for workflow tools

**For Users with Non-Supporting Clients**:
Icons are gracefully ignored - all tools work normally without any visual icons.

## Success Criteria

- All toolsets function correctly
- Tools from different toolsets are accessible
- No errors related to icon metadata
- Backward compatibility maintained
- Workflow completes successfully demonstrating all toolsets

## Version Information

- **MCP Server Version**: v0.26.3 (includes Octicon icons from v0.26.0)
- **Hotfix Applied**: v0.26.1 icon size compatibility fix (backward compatible with older clients)
- **Feature**: Automatic Octicon icons for all tools, resources, and prompts
