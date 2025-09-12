---
on:
  workflow_dispatch:
  issues:
    types: [opened, edited]

safe-outputs:
  update-issue:
    status:
    title:
    body:
    target: "*"
    max: 1
  staged: true

engine:
  id: custom
  steps:
    - name: Generate Update Issue Safe Output
      run: |
        echo '{"type": "update-issue", "title": "[UPDATED] Test Issue - Safe Output Update Test", "body": "# Updated Issue Body\n\nThis issue has been updated by the test-safe-output-update-issue workflow to validate the update-issue safe output functionality.\n\n**Update Details:**\n- Safe Output Type: update-issue\n- Updated by: Custom Engine\n- Update time: '"$(date)"'\n- Original trigger: ${{ github.event_name }}\n- Staged Mode: true\n\n**Test Status:** ✅ Update functionality verified\n\nThis update should not modify actual GitHub issues due to staged mode.\n\n## Fields Updated\n- Title: Added [UPDATED] prefix\n- Body: Completely replaced with test content\n- Status: Set to open (no change in this test)\n\n## Validation Points\n- Custom engine can generate update-issue outputs\n- All configurable fields (title, body, status) are supported\n- Staged mode prevents actual API calls", "status": "open"}' >> $GITHUB_AW_SAFE_OUTPUTS
        
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

# Test Safe Output - Update Issue

This workflow tests the `update-issue` safe output functionality using a custom engine that directly writes to the safe output file.

## Purpose

This workflow validates the update-issue safe output type by:
- Generating a JSON entry with the `update-issue` type
- Including updatable fields: title, body, and status
- Using staged mode to prevent actual GitHub interactions
- Demonstrating custom engine safe output writing for issue updates

## Trigger Events

- **workflow_dispatch**: Manual execution for testing
- **issues.opened**: Responds to new issues being created
- **issues.edited**: Responds to issues being edited

## Safe Output Configuration

- **staged: true**: Prevents real GitHub interactions
- **status**: Enables status updates (open/closed)
- **title**: Enables title updates
- **body**: Enables body content updates
- **target: "*"**: Allows updates to any issue (with issue_number in output)
- **max: 1**: Limits to one issue update per workflow run

## Custom Engine Implementation

The workflow uses a custom engine with GitHub Actions steps to:
1. Generate the appropriate update-issue JSON output
2. Include updates to title, body, and status fields
3. Add comprehensive test information to the updated content
4. Append it to the $GITHUB_AW_SAFE_OUTPUTS file
5. Verify the output was generated correctly

This demonstrates how custom engines can leverage the safe output system for updating existing issues while respecting the configured field permissions.