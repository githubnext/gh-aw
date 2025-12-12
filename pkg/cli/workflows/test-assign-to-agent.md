---
name: Test Assign to Agent
description: Test workflow for assign_to_agent safe output feature
on:
  issues:
    types: [labeled]
  workflow_dispatch:
    inputs:
      issue_number:
        description: 'Issue number to test with'
        required: true
        type: string

permissions:
  actions: write
  contents: write
  issues: write
  pull-requests: write

# NOTE: Assigning Copilot agents requires:
# 1. A Personal Access Token (PAT) with repo scope
#    - The standard GITHUB_TOKEN does NOT have permission to assign bot agents
#    - Create a PAT at: https://github.com/settings/tokens
#    - Add it as a repository secret named COPILOT_GITHUB_TOKEN
#    - Required scopes: repo (full control)
# 
# 2. All four workflow permissions declared above (for the safe output job)
#
# 3. Repository Settings > Actions > General > Workflow permissions:
#    Must be set to "Read and write permissions"

engine: copilot
timeout-minutes: 5
github-token: ${{ secrets.COPILOT_GITHUB_TOKEN }}

safe-outputs:
  assign-to-agent:
    max: 5
    name: copilot
    branch-prefix: "test-workflow/"
    allowed-agents: ["copilot"]
strict: false
---

# Assign to Agent Test Workflow

This workflow tests the `assign_to_agent` safe output feature, which allows AI agents to assign GitHub Copilot agents to issues.

## Task

**For workflow_dispatch:**
Assign the Copilot agent to issue #${{ github.event.inputs.issue_number }} using the `assign_to_agent` tool from the `safeoutputs` MCP server.

The tool now supports:
- `base_branch` parameter to specify which branch the agent should start from (any branch name is accepted)
- `agent` parameter to specify a custom agent (must be "copilot" in this workflow due to allowed-agents constraint)

Example: Call assign_to_agent with issue_number, base_branch="feature-branch", and agent="copilot"

Do not use GitHub tools. The assign_to_agent tool will handle the actual assignment.
