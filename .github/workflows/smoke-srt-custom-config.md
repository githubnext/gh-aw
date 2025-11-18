---
description: Smoke test workflow for Sandbox Runtime (SRT) with custom configuration
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
    - "*.npmjs.org"
sandbox:
  type: sandbox-runtime
  config:
    network:
      allowedDomains:
        - "github.com"
        - "*.github.com"
        - "api.github.com"
        - "npmjs.org"
        - "*.npmjs.org"
      deniedDomains: []
      allowUnixSockets:
        - "/var/run/docker.sock"
      allowLocalBinding: false
    filesystem:
      denyRead: []
      allowWrite:
        - "."
      denyWrite: []
    ignoreViolations: {}
    enableWeakerNestedSandbox: true
tools:
  bash:
  github:
timeout-minutes: 5
strict: true
---

You are testing the Sandbox Runtime (SRT) with custom configuration. Perform the following tasks:

1. Run `echo "Testing SRT with custom config"` using bash
2. Verify you can access GitHub by running a simple github tool operation
3. Check the environment with `env | grep -i copilot`

Report your findings. This validates that SRT works with custom configuration.
