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
- Each `repos:` entry needs `repo:` (`owner/name`) and `sensitivity:` (`public`, `trusted`, `internal`, `confidential`, or `sealed`).
- Choose `trusted` for repositories where the enclave may return free-form strings in a declared response schema. All other sensitivities are finite-schema-only; do not recommend free-form string schemas for them.

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
- `timeout:` per enclave entry is capped at 4,740 seconds (AWF reserves the final 60 seconds of its 4,800-second finite-disclosure bucket for cleanup). The gateway itself enforces a 4,860-second tool timeout (4,800s AWF bucket + 60s transport allowance) — treat this as an enforcement bound, not a wall-clock guarantee.

## Agent GitHub Issues profile

Use only this closed opt-in:

```yaml
sandbox:
  mcp:
    version: v0.4.15
enclaves:
  - agent:
      model: gpt-5
      github:
        cli: issues-read-v1
    repos:
      - repo: octo-org/private-service
        sensitivity: confidential
```

- `issues-read-v1` permits only the `list_issues` and `issue_read` GitHub MCP
  tools. GraphQL, search, writes, and all other GitHub tools fail closed.
- V1 allows at most one repository whose sensitivity is neither `public` nor `trusted` in the agent entry; `trusted` is public-equivalent for this limit.
- `trusted` repositories are public-equivalent for this limit, so multiple `trusted` and `public` repositories are allowed.
- The compiler generates separate primary and enclave identities for one shared
  mcpg gateway. The enclave identity is restricted to the GitHub server, those
  two tools, and the union of repositories declared in its trusted entry.
- AWF privately stages the enclave identity and connects the enclave directly
  to `/mcp/github`; the enclave has no `gh` executable or GitHub token.
- The primary agent receives neither the enclave identity nor the gateway
  configuration.
- Minimum versions are AWF `v0.28.9` and mcpg `v0.4.15`.

For a trusted repository, an `enclave_run_agent` response schema may contain strings while remaining structured and strict:

```json
{
  "type": "object",
  "fields": {
    "should_dispatch": { "type": "boolean" },
    "title": { "type": "string" },
    "problem": { "type": "string" },
    "root_cause": { "type": "string" },
    "proposed_solution": { "type": "string" }
  }
}
```

Responses must conform exactly to the declared schema: fields are required, extra properties are rejected, floats, `$ref`, recursion, regex schemas, and untagged unions are unsupported. Output remains subject to AWF's configured limit and the global 8 KiB ceiling.

See also: [agent-runtime-instructions.md](agent-runtime-instructions.md) for `sandbox.agent` fields, and [network.md](network.md) for network isolation defaults.
