---
on:
  workflow_dispatch:
permissions:
  contents: read
  actions: read
  issues: write
engine: claude
safe-outputs:
  assign-to-agent:
    max: 2
    default-agent: copilot
timeout-minutes: 5
---

# Test Claude Assign to Agent

This workflow tests the assign-to-agent safe output type with Claude engine.

Please assign the copilot agent to issue #1 and a custom agent called "research-assistant" to issue #2.

Output as JSONL format using the assign_to_agent tool.
