---
on:
  workflow_dispatch:
  issues:
    types: [opened, labeled]
  pull_request:
    types: [opened, labeled]

safe-outputs:
  add-issue-label:
    allowed: [test-safe-output, automation, bug, enhancement, documentation, question]
    max: 3
  staged: true

engine:
  id: custom
  steps:
    - name: Generate Add Issue Label Safe Output
      run: |
        echo '{"type": "add-issue-label", "labels": ["test-safe-output", "automation"]}' >> $GITHUB_AW_SAFE_OUTPUTS
        
    - name: Verify Safe Output File
      run: |
        echo "Generated safe output entries:"
        if [ -f "$GITHUB_AW_SAFE_OUTPUTS" ]; then
          cat "$GITHUB_AW_SAFE_OUTPUTS"
        else
          echo "No safe outputs file found"
        fi

permissions: read-all
---

# Test Safe Output - Add Issue Label

This workflow tests the `add-issue-label` safe output functionality using a custom engine that directly writes to the safe output file.

## Purpose

This workflow validates the add-issue-label safe output type by:
- Generating a JSON entry with the `add-issue-label` type
- Including the required labels array
- Using staged mode to prevent actual GitHub interactions
- Demonstrating custom engine safe output writing for label addition

## Trigger Events

- **workflow_dispatch**: Manual execution for testing
- **issues.opened**: Responds to new issues being created
- **issues.labeled**: Responds to issues being labeled
- **pull_request.opened**: Responds to new pull requests being created
- **pull_request.labeled**: Responds to pull requests being labeled

## Safe Output Configuration

- **staged: true**: Prevents real GitHub interactions
- **allowed**: Restricts labels to a predefined allowlist for security
- **max: 3**: Limits to three labels per workflow run

## Custom Engine Implementation

The workflow uses a custom engine with GitHub Actions steps to:
1. Generate the appropriate add-issue-label JSON output
2. Include labels that are within the allowed list
3. Append it to the $GITHUB_AW_SAFE_OUTPUTS file
4. Verify the output was generated correctly

This demonstrates how custom engines can leverage the safe output system for adding labels to issues and pull requests while respecting security constraints.