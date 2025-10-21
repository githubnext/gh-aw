---
# Test workflow demonstrating mixed format imports for steps
# This tests importing multiple shared workflows with different step formats
on: workflow_dispatch
permissions:
  contents: read
engine: copilot
imports:
  - shared/test-array-steps.md
  - shared/test-object-pre.md
  - shared/test-object-post-redaction.md
  - shared/test-object-post.md
steps:
  post:
    - name: Cleanup Test Files
      run: |
        echo "Main workflow post step executed"
        echo "Cleaning up test files..."
        rm -rf /tmp/gh-aw/test-*
---

# Test Mixed Format Import

Test that importing multiple shared workflows with different step formats works correctly:
1. test-array-steps uses `steps: [...]` - should run before AI (maps to pre)
2. test-object-pre uses `steps: { pre: [...] }` - should run before AI (after array steps)
3. test-object-post-redaction uses `steps: { post-redaction: [...] }` - should run after secret redaction
4. test-object-post uses `steps: { post: [...] }` - should run after AI execution (before main workflow post)
5. Main workflow uses `steps: { post: [...] }` - should run last

Create a simple message and list the execution order based on the logs.
