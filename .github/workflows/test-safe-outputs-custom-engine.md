---
on:
  workflow_dispatch:
  issues:
    types: [opened, reopened, closed]
  pull_request:
    types: [opened, reopened, synchronize, closed]
  push:
    branches: [main]
  schedule:
    - cron: "0 12 * * 1"  # Weekly on Mondays at noon

safe-outputs:
  create-issue:
    title-prefix: "[Custom Engine Test] "
    labels: [test-safe-outputs, automation, custom-engine]
    max: 1
  add-issue-comment:
    max: 1
    target: "*"
  create-pull-request:
    title-prefix: "[Custom Engine Test] "
    labels: [test-safe-outputs, automation, custom-engine]
    draft: true
  add-issue-label:
    allowed: [test-safe-outputs, automation, custom-engine, bug, enhancement, documentation]
    max: 3
  update-issue:
    status:
    title:
    body:
    target: "*"
    max: 1
  push-to-branch:
    target: "*"
  missing-tool:
    max: 5
  create-discussion:
    title-prefix: "[Custom Engine Test] "
    max: 1
  create-pull-request-review-comment:
    max: 1
    side: "RIGHT"

engine:
  id: custom
  steps:
    - name: Generate Create Issue Output
      run: |
        echo '{"type": "create-issue", "title": "[Custom Engine Test] Test Issue Created by Custom Engine", "body": "# Test Issue Created by Custom Engine\n\nThis issue was automatically created by the test-safe-outputs-custom-engine workflow to validate the create-issue safe output functionality.\n\n**Test Details:**\n- Engine: Custom\n- Trigger: ${{ github.event_name }}\n- Repository: ${{ github.repository }}\n- Run ID: ${{ github.run_id }}\n\nThis is a test issue and can be closed after verification.", "labels": ["test-safe-outputs", "automation", "custom-engine"]}' >> $GITHUB_AW_SAFE_OUTPUTS
        
    - name: Generate Add Issue Comment Output
      run: |
        echo '{"type": "add-issue-comment", "body": "## Test Comment from Custom Engine\n\nThis comment was automatically posted by the test-safe-outputs-custom-engine workflow to validate the add-issue-comment safe output functionality.\n\n**Test Information:**\n- Workflow: test-safe-outputs-custom-engine\n- Engine Type: Custom (GitHub Actions steps)\n- Execution Time: '"$(date)"'\n- Event: ${{ github.event_name }}\n\n✅ Safe output testing in progress..."}' >> $GITHUB_AW_SAFE_OUTPUTS
        
    - name: Generate Add Issue Labels Output
      run: |
        echo '{"type": "add-issue-label", "labels": ["test-safe-outputs", "automation", "custom-engine"]}' >> $GITHUB_AW_SAFE_OUTPUTS
        
    - name: Generate Update Issue Output
      run: |
        echo '{"type": "update-issue", "title": "[UPDATED] Test Issue - Custom Engine Safe Output Test", "body": "# Updated Issue Body\n\nThis issue has been updated by the test-safe-outputs-custom-engine workflow to validate the update-issue safe output functionality.\n\n**Update Details:**\n- Updated by: Custom Engine\n- Update time: '"$(date)"'\n- Original trigger: ${{ github.event_name }}\n\n**Test Status:** ✅ Update functionality verified", "status": "open"}' >> $GITHUB_AW_SAFE_OUTPUTS
        
    - name: Generate Create Pull Request Output
      run: |
        # Create a test file change
        echo "# Test file created by custom engine safe output test" > test-custom-engine-$(date +%Y%m%d-%H%M%S).md
        echo "This file was created to test the create-pull-request safe output." >> test-custom-engine-$(date +%Y%m%d-%H%M%S).md
        echo "Generated at: $(date)" >> test-custom-engine-$(date +%Y%m%d-%H%M%S).md
        
        # Create PR output
        echo '{"type": "create-pull-request", "title": "[Custom Engine Test] Test Pull Request - Custom Engine Safe Output", "body": "# Test Pull Request - Custom Engine Safe Output\n\nThis pull request was automatically created by the test-safe-outputs-custom-engine workflow to validate the create-pull-request safe output functionality.\n\n## Changes Made\n- Created test file with timestamp\n- Demonstrates custom engine file creation capabilities\n\n## Test Information\n- Engine: Custom (GitHub Actions steps)\n- Workflow: test-safe-outputs-custom-engine\n- Trigger Event: ${{ github.event_name }}\n- Run ID: ${{ github.run_id }}\n\nThis PR can be merged or closed after verification of the safe output functionality.", "labels": ["test-safe-outputs", "automation", "custom-engine"], "draft": true}' >> $GITHUB_AW_SAFE_OUTPUTS
        
    - name: Generate Create Discussion Output
      run: |
        echo '{"type": "create-discussion", "title": "[Custom Engine Test] Test Discussion - Custom Engine Safe Output", "body": "# Test Discussion - Custom Engine Safe Output\n\nThis discussion was automatically created by the test-safe-outputs-custom-engine workflow to validate the create-discussion safe output functionality.\n\n## Purpose\nThis discussion serves as a test of the safe output systems ability to create GitHub discussions through custom engine workflows.\n\n## Test Details\n- **Engine Type:** Custom (GitHub Actions steps)\n- **Workflow:** test-safe-outputs-custom-engine\n- **Created:** '"$(date)"'\n- **Trigger:** ${{ github.event_name }}\n- **Repository:** ${{ github.repository }}\n\n## Discussion Points\n1. Custom engine successfully executed\n2. Safe output file generation completed\n3. Discussion creation triggered\n\nFeel free to participate in this test discussion or archive it after verification."}' >> $GITHUB_AW_SAFE_OUTPUTS
        
    - name: Generate PR Review Comment Output
      run: |
        echo '{"type": "create-pull-request-review-comment", "path": "README.md", "line": 1, "body": "## Custom Engine Review Comment Test\n\nThis review comment was automatically created by the test-safe-outputs-custom-engine workflow to validate the create-pull-request-review-comment safe output functionality.\n\n**Review Details:**\n- Generated by: Custom Engine\n- Test time: '"$(date)"'\n- Workflow: test-safe-outputs-custom-engine\n\n✅ PR review comment safe output test completed."}' >> $GITHUB_AW_SAFE_OUTPUTS
        
    - name: Generate Push to Branch Output
      run: |
        # Create another test file for branch push
        echo "# Branch Push Test File" > branch-push-test-$(date +%Y%m%d-%H%M%S).md
        echo "This file tests the push-to-branch safe output functionality." >> branch-push-test-$(date +%Y%m%d-%H%M%S).md
        echo "Created by custom engine at: $(date)" >> branch-push-test-$(date +%Y%m%d-%H%M%S).md
        
        echo '{"type": "push-to-branch", "message": "Custom engine test: Push to branch functionality\n\nThis commit was generated by the test-safe-outputs-custom-engine workflow to validate the push-to-branch safe output functionality.\n\nFiles created:\n- branch-push-test-[timestamp].md\n\nTest executed at: '"$(date)"'"}' >> $GITHUB_AW_SAFE_OUTPUTS
        
    - name: Generate Missing Tool Output
      run: |
        echo '{"type": "missing-tool", "tool_name": "example-missing-tool", "reason": "This is a test of the missing-tool safe output functionality. No actual tool is missing.", "alternatives": "This is a simulated missing tool report generated by the custom engine test workflow.", "context": "test-safe-outputs-custom-engine workflow validation"}' >> $GITHUB_AW_SAFE_OUTPUTS
        
    - name: List generated outputs
      run: |
        echo "Generated safe output entries:"
        if [ -f "$GITHUB_AW_SAFE_OUTPUTS" ]; then
          cat "$GITHUB_AW_SAFE_OUTPUTS"
        else
          echo "No safe outputs file found"
        fi
        
        echo "Additional test files created:"
        ls -la *.md 2>/dev/null || echo "No additional .md files found"

permissions: read-all
---

# Test Safe Outputs - Custom Engine

This workflow validates all safe output types using the custom engine implementation. It demonstrates the ability to use GitHub Actions steps directly in agentic workflows while leveraging the safe output processing system.

## Purpose

This is a comprehensive test workflow that exercises every available safe output type:

- **create-issue**: Creates test issues with custom engine
- **add-issue-comment**: Posts comments on issues/PRs
- **create-pull-request**: Creates PRs with code changes
- **add-issue-label**: Adds labels to issues/PRs
- **update-issue**: Updates issue properties
- **push-to-branch**: Pushes changes to branches
- **missing-tool**: Reports missing functionality (test simulation)
- **create-discussion**: Creates repository discussions
- **create-pull-request-review-comment**: Creates PR review comments

## Custom Engine Implementation

The workflow uses the custom engine with GitHub Actions steps to generate all the required safe output files. Each step creates the appropriate output file with test content that demonstrates the functionality.

## Test Content

All generated content is clearly marked as test data and includes:
- Timestamp information
- Trigger event details
- Workflow identification
- Clear indication that it's test data

The content can be safely created and cleaned up as part of testing the safe output functionality.