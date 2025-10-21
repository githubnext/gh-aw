---
# Test workflow demonstrating array format import for steps
# This tests importing a shared workflow that uses array format (maps to pre steps)
on: workflow_dispatch
permissions:
  contents: read
engine: copilot
imports:
  - shared/test-array-steps.md
---

# Test Array Format Import

Test that importing a shared workflow with array format steps works correctly.
The test-array-steps shared workflow uses `steps: [...]` which should be imported as `pre` steps.

List the current directory and confirm the array format step ran before this AI execution.
