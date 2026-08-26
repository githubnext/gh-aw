---
description: Apple Container runtime smoke fixture
on:
  workflow_dispatch:
permissions:
  contents: read
  issues: read
name: Apple Container Smoke
engine: copilot
runs-on: [self-hosted, macOS, ARM64, apple-container]
sandbox:
  agent:
    id: awf
    runtime: apple-container
    version: "v0.28.9"
network:
  allowed:
    - defaults
    - github
tools:
  github:
    mode: remote
    toolsets: [repos]
timeout-minutes: 15
---

# Apple Container Runtime Smoke

This fixture exists to pin the **generated output** for `sandbox.agent.runtime: apple-container`.
It is compiled by the wasm golden tests; it is not scheduled and is never dispatched against a
real runner from this repository, because no bare-metal Apple Silicon runner is registered here.

The golden lock file is the regression guard for the properties that make this runtime safe:

- the Apple Container host preflight, CLI setup, and service start steps run before AWF,
- the MCP gateway is published on macOS loopback only and declared to AWF as
  `appleContainer.mcpGatewayUpstreamPort`,
- the agent addresses the gateway through the guest relay on loopback,
- `network.topologyAttach` is absent, and
- teardown runs with `if: always()`.

Report the repository description and stop.
