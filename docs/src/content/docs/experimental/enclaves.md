---
title: Private repository enclaves
description: Configure unified AWF script and agent enclaves through the trusted MCP gateway.
---

The top-level `enclaves` array enables finite-disclosure access to approved private repositories. The compiler registers `enclave_run_script` or `enclave_run_agent` from the keyed entries present on the `awf-enclave` MCP route. Omit the array to disable enclaves.

Enclaves require AWF network isolation, which every supported `sandbox.agent.runtime` profile provides, so the compiler launches mcpg in bridge mode and AWF can attach it to the isolated topology.

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

Each type can appear at most once. When the same repository appears in both entries, its sensitivity must match because its information budget is shared across executor types. AWF fixes the script enclave network and interpreter and the agent enclave network internally; workflows cannot override those security invariants.

The generated gateway upstream uses a fresh masked capability for each workflow run. That capability is passed only to mcpg and AWF and is excluded from the primary agent environment. The gateway allows 120 seconds for the AWF-owned HTTP upstream to become available. It enforces a 4,860-second tool timeout, covering AWF's maximum 4,800-second finite-disclosure timing bucket plus a 60-second transport allowance. Executor timeouts are capped at 4,740 seconds because AWF reserves 60 seconds in the final bucket for processing and cleanup. The gateway timeout is an enforcement bound, not an absolute AWF wall-clock guarantee under pathological host cleanup or scheduler stalls.

This compiler contract depends on the unified enclave implementation from `github/gh-aw-firewall#6992`. Until that change is available in an AWF release, pinning an older AWF version will not provide the enclave server.

## GitHub Issues access from agent enclaves

Agent enclaves can opt into the closed `issues-read-v1` profile:

```yaml
sandbox:
  agent:
    id: awf
  mcp:
    version: v0.4.13
enclaves:
  - agent:
      model: gpt-5
      github:
        cli: issues-read-v1
    repos:
      - repo: octo-org/private-service
        sensitivity: confidential
    timeout: 180
```

`issues-read-v1` is the only accepted `agent.github.cli` value. Script
enclaves cannot configure `github`. The first profile version accepts at most
one repository whose sensitivity is not `public`; additional assigned
repositories must declare `sensitivity: public`.

The profile permits only these paginated REST reads:

- `GET /repos/{owner}/{repo}/issues`
- `GET /repos/{owner}/{repo}/issues/{number}`
- `GET /repos/{owner}/{repo}/issues/{number}/comments`

Use carefully formed `gh api --method GET ...` requests for these routes. Stock
`gh issue` commands are not guaranteed because they commonly use GraphQL.
GraphQL, search, writes, and every other REST path are denied.

Public issue data uses the primary GitHub source's effective
`min-integrity`. An explicit `tools.github.min-integrity` is inherited;
otherwise the compiler uses the primary-agent default, `approved`. The
enclave entry cannot weaken this floor. The assigned repository is available
to its invocation. Other repositories are available only when an exact
visibility check reports that they are public; all other failures receive the
same denial. Private repository responses carry the
`private:<owner>/<repo>` DIFC secrecy label.

The compiler starts a dedicated mcpg proxy in Docker bridge mode. The PAT
remains in that proxy. AWF attaches it to a private control network, mints a
short-lived `awf-egh1` capability into a mode-`0600` file, and exposes only an
AWF-owned PAT-free local CLI proxy to the enclave. Neither the primary agent
nor the enclave receives the PAT, mcpg address, root key, container identity,
CA path, or repository catalog.

Provide `GH_AW_GITHUB_MCP_SERVER_TOKEN` or `GH_AW_GITHUB_TOKEN` with read access
to the assigned repository's Issues. The fallback `GITHUB_TOKEN` can only
access repositories that token can already read (typically just the current
repository in Actions).

The minimum supported versions are AWF `v0.28.9` and mcpg `v0.4.13`. The
compiler does not fall back to older versions.
