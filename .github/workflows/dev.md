---
on: 
  workflow_dispatch:
concurrency:
  group: dev-workflow-${{ github.ref }}
  cancel-in-progress: true
name: Dev
engine: copilot
permissions:
  contents: write
  actions: read
tools:
  github:
  edit:
---

# Test GitHub MCP Tools and Create Poem File

Test each GitHub MCP tool with sensible arguments to verify they are configured properly, then create or update a poem.md file with a new poem.

**Goal**: 
1. Invoke each tool from the GitHub MCP server with reasonable arguments. Some tools may fail due to missing data or invalid arguments, but they should at least be callable. Fail if there are permission issues indicating the tools aren't properly configured.
2. Create or update a `poem.md` file in the repository with a short poem about GitHub Agentic Workflows.

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

## Part 2: Create or Update Poem File

After completing the tool testing, create or update a `poem.md` file in the repository root. The file should:

1. **Contain a short poem** - Write a creative, original poem about GitHub Agentic Workflows (4-8 lines)
2. **Be in markdown format** - Use proper markdown formatting
3. **Include a title** - Use a markdown heading for the poem title

**Instructions**: Use the `edit` tool to either create a new `poem.md` file or update the existing one if it already exists. The poem should celebrate the capabilities and magic of agentic workflows.

**Example poem structure:**
```markdown
# A Poem for Agentic Workflows

In the realm of code where actions flow,
AI agents work their magic show.
With natural language as their guide,
They automate tasks far and wide.

GitHub workflows, smart and bright,
Transform our repos day and night!
```
