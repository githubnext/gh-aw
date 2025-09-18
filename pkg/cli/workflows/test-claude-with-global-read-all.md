---
on:
  workflow_dispatch:
permissions: read-all
engine: claude
tools:
  github:
    allowed: [get_repository, list_issues]
  edit:
---

# Test Claude With Global Read-All Permission

This workflow tests that Claude works correctly when given global read-all permissions. This workflow SHOULD include a checkout step in the generated lock file.

Please:
1. Check out the repository (this should happen automatically via checkout step)
2. Get basic repository information using GitHub API
3. Read and summarize the project structure by examining key files like:
   - README.md
   - go.mod
   - Makefile
4. List the open issues in the repository
5. Provide a comprehensive summary of the repository

This test validates that:
- Workflows with global read-all permission include the checkout step
- Global permissions provide access to both repository contents and GitHub APIs
- The workflow can perform comprehensive repository analysis
- File reading capabilities work with global permissions