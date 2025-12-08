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
# 1. A Personal Access Token (PAT) with appropriate permissions:
#    - Classic PAT: 'repo' scope
#    - Fine-grained PAT: Issues and Contents write permissions
#    - The standard GITHUB_TOKEN does NOT have permission to assign Copilot
#    - Create a PAT at: https://github.com/settings/tokens
#    - Store it as COPILOT_GITHUB_TOKEN secret (recommended)
#
# 2. Token precedence (December 2025 REST API):
#    - COPILOT_GITHUB_TOKEN (recommended)
#    - COPILOT_CLI_TOKEN (alternative)
#    - GH_AW_AGENT_TOKEN (legacy)
#    - GH_AW_GITHUB_TOKEN (legacy fallback)
#
# 3. Repository Settings:
#    - Copilot coding agent must be enabled
#    - Settings > Copilot > Policies > Coding agent

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

This workflow tests the `assign_to_agent` safe output feature, which allows AI agents to assign GitHub Copilot agents to issues using the REST API (December 2025).

## Task

**For workflow_dispatch:**
Assign the Copilot agent to issue #${{ github.event.inputs.issue_number }} using the `assign_to_agent` tool from the `safeoutputs` MCP server.

You can optionally provide additional options:
- `base_branch`: Specify the branch Copilot should work from
- `custom_instructions`: Provide custom instructions for Copilot
- `target_repository`: Specify a different repository for Copilot to work in
- `custom_agent`: Use a custom agent from the repository's .github/agents directory

Do not use GitHub tools. The assign_to_agent tool will handle the actual assignment.
