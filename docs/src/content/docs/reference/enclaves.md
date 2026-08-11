---
title: Private repository enclaves
description: Configure unified AWF script and agent enclaves through the trusted MCP gateway.
---

The top-level `enclaves` array enables finite-disclosure access to approved private repositories. The compiler registers `enclave_run_script` or `enclave_run_agent` from the keyed entries present on the `awf-enclave` MCP route. Omit the array to disable enclaves.

Enclaves require AWF network isolation. Configure `sandbox.agent.sudo: false` (or the `docker-sbx` runtime) so the compiler launches mcpg in bridge mode and AWF can attach it to the isolated topology.

```yaml
sandbox:
  agent:
    id: awf
    sudo: false
enclaves:
  - script:
    repos:
      - repo: octo-org/private-service
        sensitivity: confidential
    timeout: 45
  - agent:
      model: gpt-5
    repos:
      - repo: octo-org/private-service
        sensitivity: confidential
    timeout: 180
```

Each type can appear at most once. When the same repository appears in both entries, its sensitivity must match because its information budget is shared across executor types. AWF fixes the script enclave network and interpreter and the agent enclave network internally; workflows cannot override those security invariants.

The generated gateway upstream uses a fresh masked capability for each workflow run. That capability is passed only to mcpg and AWF and is excluded from the primary agent environment. The gateway allows 120 seconds for the AWF-owned HTTP upstream to become available and sets its tool timeout to the longest configured or default executor timeout plus 30 seconds.

This compiler contract depends on the unified enclave implementation from `github/gh-aw-firewall#6992`. Until that change is available in an AWF release, pinning an older AWF version will not provide the enclave server.
