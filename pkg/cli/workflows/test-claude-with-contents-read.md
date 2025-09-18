---
on:
  workflow_dispatch:
permissions:
  contents: read
  issues: write
engine: claude
tools:
  github:
    allowed: [list_issues, create_issue]
  edit:
---

# Test Claude With Contents Read Permission

This workflow tests that Claude works correctly when given contents read permission. This workflow SHOULD include a checkout step in the generated lock file.

Please:
1. Check out the repository (this should happen automatically via checkout step)
2. Read and analyze the README.md file 
3. List any other interesting files in the repository root
4. Create a new issue with title "Test Issue - With Contents Access" and include a summary of what you found in the repository files

This test validates that:
- Workflows with contents: read permission include the checkout step
- The workflow can access and read repository files
- File editing tools are available when contents permission is granted
- Both file operations and GitHub API operations work together