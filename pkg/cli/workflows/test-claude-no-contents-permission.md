---
on:
  workflow_dispatch:
permissions:
  issues: write
  pull-requests: write
engine: claude
tools:
  github:
    allowed: [list_issues, create_issue]
---

# Test Claude Without Contents Permission

This workflow tests that Claude works correctly when only given issue and pull request permissions, without any contents access. This workflow should NOT include a checkout step in the generated lock file.

Please:
1. List the current open issues in this repository
2. Create a new test issue with title "Test Issue - No Contents Access" and body "This issue was created by a workflow without contents permissions. No checkout step should be present in the workflow."

This test validates that:
- Workflows with only API permissions (issues, pull-requests) work correctly
- No unnecessary checkout step is added when contents access is not granted
- The workflow can still perform GitHub API operations without repository file access