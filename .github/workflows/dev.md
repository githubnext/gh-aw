---
on: 
  workflow_dispatch:
    inputs:
      issue_number:
        description: 'Specific issue number to assign (optional - if empty, will search for issues)'
        required: false
        type: string
      base_branch:
        description: 'Base branch for Copilot to work from (optional)'
        required: false
        type: string
name: Dev
description: Test assign-to-agent with REST API (December 2025)
timeout-minutes: 10
strict: false
engine: copilot
permissions:
  contents: read
  issues: write
  pull-requests: read
github-token: ${{ secrets.COPILOT_GITHUB_TOKEN }}
tools:
  github:
    toolsets: [repos, issues]
safe-outputs:
  assign-to-agent:
    max: 3
---
# Test Assign to Copilot Agent (REST API)

This workflow tests the assign-to-agent safe output using the December 2025 REST API.

## Current Context

- **Repository**: ${{ github.repository }}
- **Actor**: @${{ github.actor }}
- **Run**: ${{ github.run_id }}
- **Input Issue**: ${{ github.event.inputs.issue_number }}
- **Input Base Branch**: ${{ github.event.inputs.base_branch }}

## Task

### If a specific issue number was provided:

If the input issue_number `${{ github.event.inputs.issue_number }}` is not empty, assign Copilot to that specific issue:

```
assign_to_agent(
  issue_number=${{ github.event.inputs.issue_number }},
  base_branch="${{ github.event.inputs.base_branch }}"
)
```

### If no issue number was provided:

1. **Search for assignable issues**: Use GitHub tools to find open issues that are good candidates for Copilot:
   - Issues with clear, actionable requirements
   - Issues that describe a specific code change needed
   - Issues NOT already assigned to someone
   - Prefer issues with labels like "bug", "enhancement", or "good first issue"

2. **Select up to 3 candidates**: Pick issues that Copilot can realistically work on.

3. **Assign to Copilot**: For each selected issue, use the `assign_to_agent` tool:

```
assign_to_agent(
  issue_number=<issue_number>
)
```

If a base_branch was specified in the inputs, include it:
```
assign_to_agent(
  issue_number=<issue_number>,
  base_branch="${{ github.event.inputs.base_branch }}"
)
```

## Notes

- This uses the REST API (December 2025) for basic assignment
- If you specify `base_branch`, it will use GraphQL with the copilotAssignmentOptions
- The workflow requires `COPILOT_GITHUB_TOKEN` secret with `repo` scope
