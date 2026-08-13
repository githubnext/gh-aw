---
description: Configure private-repository enclaves through the trusted MCP gateway.
---

# Enclaves

The top-level `enclaves` array gives the agent finite-disclosure access to approved private repositories via an AWF-owned executor exposed only through the compiler-launched MCP gateway (`enclave_run_script` / `enclave_run_agent`). Omit the field to disable enclaves.

- Requires AWF network isolation: set `sandbox.agent.sudo: false` (or `runtime: docker-sbx`) so the compiler launches mcpg in bridge mode.
- At most one `script` entry and one `agent` entry (array length 1–2).
- When the same repository appears in both entries, its `sensitivity` must match — the information budget is shared across executor types.
- AWF fixes the script enclave's network/interpreter and the agent enclave's network internally; do not attempt to override these from workflow frontmatter.
- `timeout` is capped at 540s (script/agent executor bound); the gateway enforces a 630s tool timeout on top of that.

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

See [reference/enclaves](https://github.github.com/gh-aw/reference/enclaves/) for the full field list (`runtime`, `image`, `memory-limit`, `cpu-limit`, `pids-limit`, `tmpfs-limit`, `max-output-bytes`, `max-invocations`, `max-script-bytes`, `max-task-bytes`, `max-model-requests`, `max-model-tokens`).
