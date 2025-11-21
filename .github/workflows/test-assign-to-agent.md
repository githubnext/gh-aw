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
  contents: read
  issues: write
  pull-requests: read

engine: copilot
tools:
  github:
    mode: remote
    toolsets: [default]

safe-outputs:
  github-token: ${{ secrets.COPILOT_GITHUB_TOKEN || secrets.GH_AW_GITHUB_TOKEN || secrets.GITHUB_TOKEN }}
  assign-to-agent:
    max: 5
    default-agent: copilot
---

# Assign to Agent Test Workflow

This workflow tests the `assign_to_agent` safe output feature, which allows AI agents to assign GitHub Copilot agents to issues.

## Task

You are testing the assign_to_agent feature. Based on the trigger:

**For workflow_dispatch:**
- Assign agent "copilot" to issue #${{ github.event.inputs.issue_number }}

**For issue labeled events:**
- If the issue has label "needs-agent", assign agent "copilot" to issue #${{ github.event.issue.number }}
- If the issue has label "needs-review", assign agent "code-reviewer" to issue #${{ github.event.issue.number }}

Use the `assign_to_agent` safe output with the following structure:
```json
{
  "type": "assign_to_agent",
  "issue_number": <number>,
  "agent": "<agent-name>"
}
```

**Important notes:**
- `issue_number` is REQUIRED (will fail validation if missing)
- `agent` is optional (defaults to "copilot" if not provided)
- Only provide valid issue numbers from this repository

After assigning, create a comment summarizing what you did.
