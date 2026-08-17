---
description: Private-repository enclaves (preview) — finite-disclosure access to approved private repos via the MCP gateway.
---

# Private Repository Enclaves

Use these instructions when a workflow needs bounded, auditable access to a private repository other than the one the workflow runs in.

## What it is

- The top-level `enclaves:` array (1-2 entries) enables finite-disclosure access to approved private repositories through the compiler-launched MCP gateway.
- Each entry is either a **script enclave** (`script:` + `repos:`) registering `enclave_run_script`, or an **agent enclave** (`agent:` + `repos:`) registering `enclave_run_agent`.
- Omit `enclaves:` entirely to disable the feature — this is the default.
- This is a preview feature gated on `github/gh-aw-firewall#6992`; an older pinned AWF version will not provide the enclave server.

## Prerequisites

- Enclaves require AWF network isolation, which every supported `sandbox.agent.runtime` profile provides, so the compiler launches the MCP gateway in bridge mode and AWF can attach it to the isolated topology.
- Each `repos:` entry needs `repo:` (`owner/name`) and `sensitivity:` (`public`, `internal`, `confidential`, or `sealed`).

## Example

```yaml
sandbox:
  agent:
    id: awf
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

## Rules

- Each enclave type (`script`, `agent`) can appear at most once.
- If the same repository appears in both entries, its `sensitivity` must match — the information budget is shared across executor types.
- AWF fixes the script enclave's network and interpreter, and the agent enclave's network, internally; do not attempt to override these in workflow frontmatter.
- A fresh masked capability is generated per workflow run and passed only to the MCP gateway and AWF, never to the primary agent environment.
- `timeout:` per enclave entry is capped at 540 seconds (AWF reserves the final 60 seconds of its 600-second finite-disclosure bucket for cleanup). The gateway itself enforces a 630-second tool timeout (600s AWF bucket + 30s transport allowance) — treat this as an enforcement bound, not a wall-clock guarantee.

See also: [agent-runtime-instructions.md](agent-runtime-instructions.md) for `sandbox.agent` fields, and [network.md](network.md) for network isolation defaults.
