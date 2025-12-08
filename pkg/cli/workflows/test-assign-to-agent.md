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
  issues: write

# NOTE: Assigning Copilot agents requires:
# 1. A Personal Access Token (PAT) with issues write scope
#    - The standard GITHUB_TOKEN does NOT have permission to assign bot agents
#    - Create a fine-grained PAT at: https://github.com/settings/tokens?type=beta
#    - Add it as a repository secret named COPILOT_GITHUB_TOKEN or GH_AW_AGENT_TOKEN
#    - Required scope: Issues (Write)
# 
# 2. Copilot coding agent must be enabled for the repository
#
# See: https://github.blog/changelog/2025-12-03-assign-issues-to-copilot-using-the-api/

engine: copilot
timeout-minutes: 5
github-token: ${{ secrets.COPILOT_GITHUB_TOKEN }}

safe-outputs:
  assign-to-agent:
    max: 5
    name: copilot
strict: false
---

# Assign to Agent Test Workflow

This workflow tests the `assign_to_agent` safe output feature, which allows AI agents to assign GitHub Copilot agents to issues.

## Task

**For workflow_dispatch:**
Assign the Copilot agent to issue #${{ github.event.inputs.issue_number }} using the `assign_to_agent` tool from the `safeoutputs` MCP server.

Do not use GitHub tools. The assign_to_agent tool will handle the actual assignment.
