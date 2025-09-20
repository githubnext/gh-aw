---
on:
  workflow_dispatch:
permissions: {}
engine: claude
tools:
  github:
    allowed: [get_repository]
---

# Test Claude With Empty Permissions

This workflow tests that Claude works correctly when given empty permissions. This workflow should NOT include a checkout step in the generated lock file.

Please:
1. Get basic repository information using GitHub API (this should work with minimal permissions)
2. Report what you were able to access and what limitations you encountered

This test validates that:
- Workflows with empty permissions ({}) do not include unnecessary checkout steps
- The workflow can still perform basic GitHub API operations that don't require special permissions
- The system properly handles minimal permission scenarios
- No file access is attempted when no contents permission is granted

Note: This workflow intentionally has very limited permissions to test the minimal access scenario.