---
on:
  workflow_dispatch:
  issues:
    types: [opened]

safe-outputs:
  create-issue:
    title-prefix: "[Test] "
    labels: [test-safe-output, automation]
    max: 1
  staged: true

engine:
  id: custom
  steps:
    - name: Generate Create Issue Safe Output
      run: |
        echo '{"type": "create-issue", "title": "Test Issue Created by Safe Output", "body": "# Test Issue for create-issue Safe Output\n\nThis issue was automatically created by the test-safe-output-create-issue workflow to validate the create-issue safe output functionality.\n\n**Test Details:**\n- Safe Output Type: create-issue\n- Engine: Custom\n- Trigger: ${{ github.event_name }}\n- Repository: ${{ github.repository }}\n- Run ID: ${{ github.run_id }}\n- Staged Mode: true\n\nThis is a test issue and should not create actual GitHub interactions due to staged mode.", "labels": ["test-safe-output", "automation"]}' >> $GITHUB_AW_SAFE_OUTPUTS
        
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

# Test Safe Output - Create Issue

This workflow tests the `create-issue` safe output functionality using a custom engine that directly writes to the safe output file.

## Purpose

This workflow validates the create-issue safe output type by:
- Generating a JSON entry with the `create-issue` type
- Including all required fields: title, body, labels
- Using staged mode to prevent actual GitHub interactions
- Demonstrating custom engine safe output writing

## Trigger Events

- **workflow_dispatch**: Manual execution for testing
- **issues.opened**: Responds to new issues being created

## Safe Output Configuration

- **staged: true**: Prevents real GitHub interactions
- **title-prefix**: Adds "[Test] " prefix to issue titles
- **labels**: Automatically adds test-safe-output and automation labels
- **max: 1**: Limits to one issue creation per workflow run

## Custom Engine Implementation

The workflow uses a custom engine with GitHub Actions steps to:
1. Generate the appropriate create-issue JSON output
2. Append it to the $GITHUB_AW_SAFE_OUTPUTS file
3. Verify the output was generated correctly

This demonstrates how custom engines can leverage the safe output system for issue creation.