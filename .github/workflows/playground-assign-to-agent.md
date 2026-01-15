---
name: "Playground: assign-to-agent"
on: 
  workflow_dispatch:
    inputs:
      item_url:
        description: 'Issue or PR URL to assign agent to (e.g., https://github.com/owner/repo/issues/123 or https://github.com/owner/repo/pull/456)'
        required: true
        type: string
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

Test the `assign-to-agent` safe output feature by assigning the Copilot agent to an issue or pull request.

## Task

You have been provided with a GitHub URL: ${{ github.event.inputs.item_url }}

1. Parse the URL to extract the owner, repo, and number
2. Determine if the URL is an issue URL (contains `/issues/`) or a pull request URL (contains `/pull/`)
3. Use the `assign_to_agent` tool from the `safeoutputs` MCP server to assign the Copilot agent
4. Pass the appropriate parameter to the tool:
   - For issues: pass `issue_number` (the numeric ID extracted from the URL)
   - For pull requests: pass `pull_number` (the numeric ID extracted from the URL)

**Important**: Do not use GitHub tools directly for assignment. Only use the `assign_to_agent` safe output tool with the correct parameter based on the URL type.
