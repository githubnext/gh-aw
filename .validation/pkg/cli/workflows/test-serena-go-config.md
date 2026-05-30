---
on: workflow_dispatch
engine: copilot
permissions:
  contents: read
tools:
  serena:
    languages:
      go:
        version: "1.21"
        go-mod-file: "go.mod"
        gopls-version: "v0.14.2"
strict: false
---

# Test Serena Go Configuration

Test workflow to verify Serena Go configuration with custom go version, go.mod file location, and gopls version.
