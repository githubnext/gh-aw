---
name: Test Custom Mounts
on: workflow_dispatch
engine: copilot
sandbox:
  agent:
    id: awf
    mounts:
      - "/host/data:/data:ro"
      - "/usr/local/bin/custom-tool:/usr/local/bin/custom-tool:ro"
network:
  allowed:
    - defaults
---

Test workflow to verify custom mounts configuration.
