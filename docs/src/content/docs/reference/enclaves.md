---
title: Private repository enclaves
description: Configure unified AWF script and agent enclaves through the trusted MCP gateway.
---

The top-level `enclaves` field enables finite-disclosure access to approved private repositories. The compiler registers only the enabled `enclave_run_script` and `enclave_run_agent` tools on the `awf-enclave` MCP route.

Enclaves require AWF network isolation. Configure `sandbox.agent.sudo: false` (or the `docker-sbx` runtime) so the compiler launches mcpg in bridge mode and AWF can attach it to the isolated topology.

```yaml
enclaves:
  enabled: true
  private-repos:
    - repo: octo-org/private-service
      sensitivity: confidential
  executors:
    script:
      enabled: true
      timeout: 45
    agent:
      enabled: true
      model: gpt-5
      timeout: 180

sandbox:
  agent:
    id: awf
    sudo: false
```

The generated gateway upstream uses a fresh masked capability for each workflow run. That capability is passed only to mcpg and AWF and is excluded from the primary agent environment. The gateway allows 120 seconds for the AWF-owned HTTP upstream to become available and sets its tool timeout to the longest enabled executor timeout plus 30 seconds.

This compiler contract depends on the unified enclave implementation from `github/gh-aw-firewall#6992`. Until that change is available in an AWF release, pinning an older AWF version will not provide the enclave server.
