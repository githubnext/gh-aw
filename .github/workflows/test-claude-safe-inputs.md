---
on: workflow_dispatch
engine: claude
safe-inputs:
  test-tool:
    description: Test tool
    script: |
      return { result: "hello" };
---

Test safe-inputs with Claude
