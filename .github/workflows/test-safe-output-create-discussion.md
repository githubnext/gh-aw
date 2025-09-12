---
on:
  workflow_dispatch:
  issues:
    types: [opened]
  push:
    branches: [main]

safe-outputs:
  create-discussion:
    title-prefix: "[Test] "
    max: 1
  staged: true

engine:
  id: custom
  steps:
    - name: Generate Create Discussion Safe Output
      run: |
        echo '{"type": "create-discussion", "title": "Test Discussion - Safe Output Validation", "body": "# Test Discussion - create-discussion Safe Output\n\nThis discussion was automatically created by the test-safe-output-create-discussion workflow to validate the create-discussion safe output functionality.\n\n## Purpose\nThis discussion serves as a test of the safe output systems ability to create GitHub discussions through custom engine workflows.\n\n## Test Details\n- **Safe Output Type:** create-discussion\n- **Engine Type:** Custom (GitHub Actions steps)\n- **Workflow:** test-safe-output-create-discussion\n- **Created:** '"$(date)"'\n- **Trigger:** ${{ github.event_name }}\n- **Repository:** ${{ github.repository }}\n- **Run ID:** ${{ github.run_id }}\n- **Staged Mode:** true\n\n## Discussion Points\n1. Custom engine successfully executed\n2. Safe output file generation completed\n3. Discussion creation triggered\n4. Staged mode prevents actual GitHub interactions\n\n## Validation Checklist\n- ✅ JSON output properly formatted\n- ✅ Required fields included (title, body)\n- ✅ Test information comprehensive\n- ✅ Safe output file appended correctly\n\nFeel free to participate in this test discussion or archive it after verification (though this should not create actual discussions due to staged mode)."}' >> $GITHUB_AW_SAFE_OUTPUTS
        
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

# Test Safe Output - Create Discussion

This workflow tests the `create-discussion` safe output functionality using a custom engine that directly writes to the safe output file.

## Purpose

This workflow validates the create-discussion safe output type by:
- Generating a JSON entry with the `create-discussion` type
- Including all required fields: title and body
- Using staged mode to prevent actual GitHub interactions
- Demonstrating custom engine safe output writing for discussion creation

## Trigger Events

- **workflow_dispatch**: Manual execution for testing
- **issues.opened**: Responds to new issues being created (discussions often relate to issues)
- **push**: Responds to pushes on main branch

## Safe Output Configuration

- **staged: true**: Prevents real GitHub interactions
- **title-prefix**: Adds "[Test] " prefix to discussion titles
- **max: 1**: Limits to one discussion creation per workflow run

## Custom Engine Implementation

The workflow uses a custom engine with GitHub Actions steps to:
1. Generate the appropriate create-discussion JSON output
2. Include comprehensive discussion content with test information
3. Add structured information about the test execution
4. Append it to the $GITHUB_AW_SAFE_OUTPUTS file
5. Verify the output was generated correctly

This demonstrates how custom engines can leverage the safe output system for creating repository discussions, which are useful for community engagement and project planning.