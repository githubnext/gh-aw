---
# Test workflow demonstrating object format import for steps
# This tests importing a shared workflow that uses object format with post-redaction steps
on: workflow_dispatch
permissions:
  contents: read
engine: copilot
imports:
  - shared/test-object-post-redaction.md
---

# Test Object Format Import

Test that importing a shared workflow with object format steps works correctly.
The test-object-post-redaction shared workflow uses `steps: { post-redaction: [...] }` which should be imported
and run after secret redaction but before artifacts.

Create a simple message confirming the test.
