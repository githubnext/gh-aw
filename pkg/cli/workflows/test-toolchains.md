---
name: Test Toolchains
on: workflow_dispatch
engine: copilot
sandbox:
  agent:
    id: awf
    toolchains:
      - go
      - python
network:
  allowed:
    - defaults
---

Test workflow to verify toolchains configuration. The Go and Python toolchains should be mounted and available in PATH.
