---
on:
  workflow_dispatch:
  push:
    branches: [main, develop]

safe-outputs:
  create-pull-request:
    title-prefix: "[Test] "
    labels: [test-safe-output, automation]
    draft: true
    if-no-changes: "warn"
  staged: true

engine:
  id: custom
  steps:
    - name: Create Test File Changes
      run: |
        # Create a test file to demonstrate PR creation
        echo "# Test file created by safe output test" > test-pr-creation-$(date +%Y%m%d-%H%M%S).md
        echo "This file was created to test the create-pull-request safe output." >> test-pr-creation-$(date +%Y%m%d-%H%M%S).md
        echo "Generated at: $(date)" >> test-pr-creation-$(date +%Y%m%d-%H%M%S).md
        echo "Event: ${{ github.event_name }}" >> test-pr-creation-$(date +%Y%m%d-%H%M%S).md
        echo "Repository: ${{ github.repository }}" >> test-pr-creation-$(date +%Y%m%d-%H%M%S).md
        echo "Run ID: ${{ github.run_id }}" >> test-pr-creation-$(date +%Y%m%d-%H%M%S).md
        
    - name: Generate Create Pull Request Safe Output
      run: |
        echo '{"type": "create-pull-request", "title": "Test Pull Request - Safe Output Validation", "body": "# Test Pull Request - create-pull-request Safe Output\n\nThis pull request was automatically created by the test-safe-output-create-pull-request workflow to validate the create-pull-request safe output functionality.\n\n## Changes Made\n- Created test file with timestamp\n- Demonstrates custom engine file creation capabilities\n- Tests safe output PR creation functionality\n\n## Test Information\n- Safe Output Type: create-pull-request\n- Engine: Custom (GitHub Actions steps)\n- Workflow: test-safe-output-create-pull-request\n- Trigger Event: ${{ github.event_name }}\n- Run ID: ${{ github.run_id }}\n- Staged Mode: true\n\nThis PR should not create actual GitHub interactions due to staged mode.\n\n## Validation\n- ✅ File changes created\n- ✅ JSON output generated\n- ✅ Safe output functionality tested\n\nThis PR can be closed after verification of the safe output functionality.", "labels": ["test-safe-output", "automation"], "draft": true}' >> $GITHUB_AW_SAFE_OUTPUTS
        
    - name: Verify Safe Output File
      run: |
        echo "Generated safe output entries:"
        if [ -f "$GITHUB_AW_SAFE_OUTPUTS" ]; then
          cat "$GITHUB_AW_SAFE_OUTPUTS"
        else
          echo "No safe outputs file found"
        fi
        
        echo "Test files created:"
        ls -la *.md 2>/dev/null || echo "No .md files found"

permissions: read-all
---

# Test Safe Output - Create Pull Request

This workflow tests the `create-pull-request` safe output functionality using a custom engine that creates file changes and writes to the safe output file.

## Purpose

This workflow validates the create-pull-request safe output type by:
- Creating actual file changes to include in the PR
- Generating a JSON entry with the `create-pull-request` type
- Including all required fields: title, body, labels, draft status
- Using staged mode to prevent actual GitHub interactions
- Demonstrating custom engine safe output writing for PR creation

## Trigger Events

- **workflow_dispatch**: Manual execution for testing
- **push**: Responds to pushes on main and develop branches

## Safe Output Configuration

- **staged: true**: Prevents real GitHub interactions
- **title-prefix**: Adds "[Test] " prefix to PR titles
- **labels**: Automatically adds test-safe-output and automation labels
- **draft: true**: Creates PR as draft
- **if-no-changes: "warn"**: Warns but succeeds if no changes are detected

## Custom Engine Implementation

The workflow uses a custom engine with GitHub Actions steps to:
1. Create test file changes to include in the PR
2. Generate the appropriate create-pull-request JSON output
3. Include comprehensive PR description with test information
4. Append it to the $GITHUB_AW_SAFE_OUTPUTS file
5. Verify both the output and created files

This demonstrates how custom engines can leverage the safe output system for pull request creation with actual file changes.