---
on: workflow_dispatch
engine: codex
safe-inputs:
  test-tool:
    description: Test tool
    script: |
      return { result: "hello" };
---

Test safe-inputs with Codex
