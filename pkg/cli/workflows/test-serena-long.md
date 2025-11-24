---
on: workflow_dispatch
engine: copilot
permissions:
  contents: read
tools:
  serena:
    version: latest
    args: ["--verbose"]
    languages:
      go:
        version: "1.21"
        go-mod-file: "go.mod"
        gopls-version: "v0.14.2"
      typescript:
      python:
        version: "3.12"
strict: false
---

# Test Serena Long Syntax

Test workflow to verify Serena MCP with long syntax (detailed configuration including Go version, go.mod path, and gopls version).
