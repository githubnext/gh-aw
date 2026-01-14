---
on: 
  workflow_dispatch:
    inputs:
      issue_number:
        description: Issue number to assign agent to
        required: true
        type: string
name: Dev
description: Test assign-to-agent safe output feature
timeout-minutes: 5
strict: false
sandbox: false
engine: copilot

permissions:
  contents: read
  issues: read
  pull-requests: read

# NOTE: Assigning agents requires:
# 1. A fine-grained Personal Access Token (PAT) with write access for:
#    - actions, contents, issues, pull-requests
#    - Store as GH_AW_AGENT_TOKEN repository secret
# 2. The github-token configured below provides write access via the PAT
# 3. Repository Settings > Actions > General > Workflow permissions:
#    Must be set to "Read and write permissions"

github-token: ${{ secrets.GH_AW_AGENT_TOKEN }}

safe-outputs:
  assign-to-agent:
    max: 1
    name: copilot
---

# Assign Agent Test Workflow

Test the `assign-to-agent` safe output feature by assigning the Copilot agent to an issue.

## Task

Assign the Copilot agent to issue #${{ github.event.inputs.issue_number }} using the `assign_to_agent` tool from the `safeoutputs` MCP server.

The `assign_to_agent` tool will handle the actual assignment. Do not use GitHub tools directly for assignment.
