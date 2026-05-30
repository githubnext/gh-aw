---
description: Test workflow for validating post-steps functionality and proper compilation
# Test workflow for post-steps functionality
# This workflow validates that post-steps compile correctly and are properly indented

on:
  workflow_dispatch:

permissions:
  contents: read

engine: copilot

tools:
  github:
    toolsets: [repos]
    allowed: [get_repository]

# Steps that run after AI execution
post-steps:
  - name: Verify Post-Steps Execution
    run: |
      echo "✅ Post-steps are executing correctly"
      echo "This step runs after the AI agent completes"
  
  - name: Upload Test Results
    if: always()
    uses: actions/upload-artifact@v6
    with:
      name: post-steps-test-results
      path: /tmp/gh-aw/
      retention-days: 1
      if-no-files-found: ignore
  
  - name: Final Summary
    run: |
      {
        echo "## Post-Steps Test Summary"
        echo "✅ All post-steps executed successfully"
        echo "This validates the post-steps indentation fix"
      } >> "$GITHUB_STEP_SUMMARY"

timeout-minutes: 5
---

# Test Post-Steps Workflow

This is a test workflow to validate that post-steps compile correctly with proper YAML indentation.

## Your Task

Respond with a simple message acknowledging this test workflow.

**Repository**: ${{ github.repository }}
