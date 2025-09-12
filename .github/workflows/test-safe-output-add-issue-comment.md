---
on:
  workflow_dispatch:
  issues:
    types: [opened]
  pull_request:
    types: [opened]

safe-outputs:
  add-issue-comment:
    max: 1
    target: "*"
  staged: true

engine:
  id: custom
  steps:
    - name: Generate Add Issue Comment Safe Output
      run: |
        echo '{"type": "add-issue-comment", "body": "## Test Comment from Safe Output\n\nThis comment was automatically posted by the test-safe-output-add-issue-comment workflow to validate the add-issue-comment safe output functionality.\n\n**Test Information:**\n- Safe Output Type: add-issue-comment\n- Engine Type: Custom (GitHub Actions steps)\n- Execution Time: '"$(date)"'\n- Event: ${{ github.event_name }}\n- Staged Mode: true\n\n✅ Safe output testing in progress...\n\nThis comment should not appear in actual GitHub interactions due to staged mode."}' >> $GITHUB_AW_SAFE_OUTPUTS
        
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

# Test Safe Output - Add Issue Comment

This workflow tests the `add-issue-comment` safe output functionality using a custom engine that directly writes to the safe output file.

## Purpose

This workflow validates the add-issue-comment safe output type by:
- Generating a JSON entry with the `add-issue-comment` type
- Including the required body field with formatted markdown
- Using staged mode to prevent actual GitHub interactions
- Demonstrating custom engine safe output writing for comments

## Trigger Events

- **workflow_dispatch**: Manual execution for testing
- **issues.opened**: Responds to new issues being created
- **pull_request.opened**: Responds to new pull requests being created

## Safe Output Configuration

- **staged: true**: Prevents real GitHub interactions
- **target: "*"**: Allows comments on any issue (with issue_number in output)
- **max: 1**: Limits to one comment per workflow run

## Custom Engine Implementation

The workflow uses a custom engine with GitHub Actions steps to:
1. Generate the appropriate add-issue-comment JSON output
2. Include formatted markdown content with test information
3. Append it to the $GITHUB_AW_SAFE_OUTPUTS file
4. Verify the output was generated correctly

This demonstrates how custom engines can leverage the safe output system for commenting on issues and pull requests.