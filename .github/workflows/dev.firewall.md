---
on: 
  workflow_dispatch:
concurrency:
  group: dev-workflow-${{ github.ref }}
  cancel-in-progress: true
name: Dev Firewall
engine: copilot
permissions:
  contents: read
  issues: read
  pull-requests: read
network:
  firewall: true
tools:
  github:
---

# Test GitHub MCP Tools

Test each GitHub MCP tool with sensible arguments to verify they are configured properly.

**Goal**: Invoke each tool from the GitHub MCP server with reasonable arguments. Some tools may fail due to missing data or invalid arguments, but they should at least be callable. Fail if there are permission issues indicating the tools aren't properly configured.

## Instructions

**Discover and test all available GitHub MCP tools:**

1. First, explore and identify all tools available from the GitHub MCP server
2. For each discovered tool, invoke it with sensible arguments based on the repository context (${{ github.repository }})
3. Use appropriate parameters for each tool (e.g., repository name, issue numbers, PR numbers, etc.)

Example tools you should discover and test may include (but are not limited to):
- Context tools: `get_me`, etc.
- Repository tools: `get_file_contents`, `list_branches`, `list_commits`, `search_repositories`, etc.
- Issues tools: `list_issues`, `search_issues`, `issue_read`, etc.
- Pull Request tools: `list_pull_requests`, `get_pull_request`, `search_pull_requests`, etc.
- Actions tools: `list_workflows`, `list_workflow_runs`, etc.
- Release tools: `list_releases`, etc.
- And any other tools you discover from the GitHub MCP server

## Expected Behavior

- Each tool should be invoked successfully, even if it returns empty results or errors due to data not existing
- If a tool cannot be called due to **permission issues** (e.g., "tool not allowed", "permission denied", "unauthorized"), the task should **FAIL** 
- If a tool fails due to invalid arguments or missing data (e.g., "resource not found", "invalid parameters"), that's acceptable - continue to the next tool
- Log the results of each tool invocation (success or failure reason)

## Summary

After testing all tools, provide a summary:
- Total tools tested: [count]
- Successfully invoked: [count]
- Failed due to missing data/invalid args: [count]  
- Failed due to permission issues: [count] - **FAIL if > 0**

If any permission issues were encountered, clearly state which tools had permission problems and fail the workflow.
