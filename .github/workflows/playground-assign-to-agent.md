---
on: 
  workflow_dispatch:
    inputs:
      issue_url:
        description: 'Issue URL to assign agent to (e.g., https://github.com/owner/repo/issues/123)'
        required: true
        type: string
name: Playground Assign to Agent
description: Test assign-to-agent safe output feature
permissions:
  contents: read
  issues: read
  pull-requests: read

# NOTE: Assigning agents requires:
# 1. A fine-grained Personal Access Token (PAT) with write access for:
#    - actions, contents, issues, pull-requests
#    - Store as PLAYGROUND_AGENT_TOKEN repository secret
# 2. The github-token configured below provides write access via the PAT
# 3. Repository Settings > Actions > General > Workflow permissions:
#    Must be set to "Read and write permissions"

safe-outputs:
  github-token: ${{ secrets.PLAYGROUND_AGENT_TOKEN }}
  assign-to-agent:
    max: 1
    name: copilot

timeout-minutes: 5
---

# Assign Agent Test Workflow

Test the `assign-to-agent` safe output feature by assigning the Copilot agent to an issue.

## Task

You have been provided with an issue URL: ${{ github.event.inputs.issue_url }}

1. Parse the issue URL to extract the owner, repo, and issue number
2. Validate that the URL is an issue URL (not a pull request URL)
3. Use the `assign_to_agent` tool from the `safeoutputs` MCP server to assign the Copilot agent to the issue
4. Pass the numeric issue_number (extracted from the URL) to the `assign_to_agent` tool

**Important**: Do not use GitHub tools directly for assignment. Only use the `assign_to_agent` safe output tool.
