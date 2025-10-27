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
safe-outputs:
  create-issue:
    title-prefix: "[dev] "
    labels: [automation, dev-workflow]
---

# Test GitHub MCP Tools and Create Poem Issue

Test each GitHub MCP tool with sensible arguments to verify they are configured properly, then create a child issue under issue #2439 containing a short poem.

**Goal**: 
1. Invoke each tool from the GitHub MCP server with reasonable arguments. Some tools may fail due to missing data or invalid arguments, but they should at least be callable. Fail if there are permission issues indicating the tools aren't properly configured.
2. Create a child issue under issue #2439 with a short poem about GitHub Agentic Workflows.

## Instructions

**Part 1: Discover and test all available GitHub MCP tools:**

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

## Expected Behavior for Part 1

- Each tool should be invoked successfully, even if it returns empty results or errors due to data not existing
- If a tool cannot be called due to **permission issues** (e.g., "tool not allowed", "permission denied", "unauthorized"), the task should **FAIL** 
- If a tool fails due to invalid arguments or missing data (e.g., "resource not found", "invalid parameters"), that's acceptable - continue to the next tool
- Log the results of each tool invocation (success or failure reason)

## Part 1 Summary

After testing all tools, provide a summary:
- Total tools tested: [count]
- Successfully invoked: [count]
- Failed due to missing data/invalid args: [count]  
- Failed due to permission issues: [count] - **FAIL if > 0**

If any permission issues were encountered, clearly state which tools had permission problems and fail the workflow.

## Part 2: Create Child Issue with Poem

After completing the tool testing, use the `create_issue` tool to create a child issue under issue #2439. The issue should:

1. **Be a child issue** - Set the parent field to 2439
2. **Contain a short poem** - Write a creative, original poem about GitHub Agentic Workflows (4-8 lines)
3. **Have an appropriate title** - Something like "A Poem for Agentic Workflows"
4. **Reference the parent** - The body should acknowledge it's related to issue #2439

**Important**: When calling `create_issue`, include the JSON field `"parent": 2439` to make this a child issue.

Example structure:
```json
{
  "type": "create_issue",
  "title": "A Poem for Agentic Workflows",
  "body": "[Your creative poem here]\n\nThis poem celebrates the magic of agentic workflows!",
  "parent": 2439,
  "labels": ["poetry", "automation"]
}
```
