---
description: Smoke test workflow for Sandbox Runtime (SRT) with custom configuration
features:
  sandbox-runtime: true
on:
  workflow_dispatch:
permissions:
  contents: read
name: Smoke SRT Custom Config
engine: copilot
network:
  allowed:
    - defaults
    - github
    - node
    - python
    - java
    - go
    - ruby
    - rust
    - "*.githubcopilot.com"
    - "example.com"
sandbox:
  type: sandbox-runtime
  config:
    filesystem:
      denyRead: []
      allowWrite:
        - "."
        - "/tmp"
        - "/home/runner/.copilot"
        - "/home/runner"
      denyWrite: []
    ignoreViolations: {}
    enableWeakerNestedSandbox: true
tools:
  bash:
  github:
timeout-minutes: 5
strict: true
---

**IMPORTANT: Keep all outputs extremely short and concise. Use single-line responses where possible. No verbose explanations.**

Test the Sandbox Runtime (SRT) with custom configuration:

1. Run `echo "Testing SRT with custom config"` using bash
2. Verify you can access GitHub by running a simple github tool operation
3. Check the environment with `env | grep -i copilot`

Output a **very brief** summary (max 3-5 lines): ✅ or ❌ for each test, overall status.
