---
name: Test Sandbox Model Fallback
on:
  workflow_dispatch:
permissions:
  contents: read
engine: copilot
sandbox:
  agent:
    id: awf
    model-fallback: false
---

# Test Sandbox Model Fallback

Verify that sandbox.agent.model-fallback compiles into the AWF config JSON.
