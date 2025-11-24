---
on: workflow_dispatch
engine: copilot
permissions:
  contents: read
tools:
  serena:
    languages:
      go:
        go-mod-file: "backend/go.mod"
        gopls-version: "latest"
strict: false
---

# Test Serena Custom Go.mod Path

Test workflow to verify Serena with custom go.mod file path.
