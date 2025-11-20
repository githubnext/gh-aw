---
on:
  workflow_dispatch:
permissions:
  contents: read
  actions: read
  issues: write
engine: codex
safe-outputs:
  assign-to-agent:
    max: 2
    default-agent: copilot
timeout-minutes: 5
---

# Test Codex Assign to Agent

This workflow tests the assign-to-agent safe output type with Codex engine.

Please assign the copilot agent to issue #1 and a custom agent called "code-reviewer" to issue #2.

Output as JSONL format using the assign_to_agent tool.
