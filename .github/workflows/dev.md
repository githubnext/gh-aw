---
on: 
  workflow_dispatch:
concurrency:
  group: dev-workflow-${{ github.ref }}
  cancel-in-progress: true
name: Dev
engine: copilot
permissions:
  contents: read
  actions: read
tools:
  github:
---

# Test GitHub MCP Tools

Test each GitHub MCP tool with sensible arguments to verify they are configured properly.

**Goal**: Invoke each tool from the GitHub MCP server with reasonable arguments. Some tools may fail due to missing data or invalid arguments, but they should at least be callable. Fail if there are permission issues indicating the tools aren't properly configured.

## Instructions

Go through the following GitHub MCP tools and invoke each one with sensible arguments based on the repository context (${{ github.repository }}):

### Context Tools
1. `get_me` - Get information about the authenticated user

### Repository Tools  
2. `get_file_contents` - Get contents of README.md or another file from the repo
3. `list_branches` - List branches in the repository
4. `list_commits` - List recent commits on the main branch
5. `list_tags` - List tags in the repository
6. `search_repositories` - Search for repositories related to "github actions"

### Issues Tools
7. `list_issues` - List recent issues in the repository (state: all, per_page: 5)
8. `search_issues` - Search for issues with a keyword

### Pull Request Tools
9. `list_pull_requests` - List recent pull requests (state: all, per_page: 5)
10. `search_pull_requests` - Search for pull requests

### Actions Tools
11. `list_workflows` - List GitHub Actions workflows in the repository
12. `list_workflow_runs` - List recent workflow runs (per_page: 5)

### Release Tools
13. `list_releases` - List releases in the repository (per_page: 5)

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
