---
name: Test Import Tools
on: issues
engine: copilot
imports:
  - shared/test/importable-tools.md
permissions:
  actions: read
  contents: read
---

# Test Workflow Using Imported Tools

This workflow tests that agentic-workflows, serena, and playwright tools can be imported from a shared workflow file.

The tools should be available:
- agentic-workflows for workflow introspection
- serena for Go and TypeScript code intelligence
- playwright for browser automation on example.com and github.com
