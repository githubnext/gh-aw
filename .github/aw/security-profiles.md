---
description: Canonical selection rules for sandbox runtime, GitHub access, and MCP exposure profiles.
disable-model-invocation: true
---

# Security Profile Selection

Use this file as the canonical decision reference. These selectors are independent:

| Selector | Controls |
| --- | --- |
| `sandbox.agent.runtime` | Agent isolation, AWF privilege, and host/service access |
| `tools.github.mode` | GitHub access through `gh` or a GitHub MCP server |
| `tools.mcp-mode` | Native MCP exposure versus CLI wrappers for MCP servers, including a selected GitHub MCP server |

Never emit removed `sandbox.agent.sudo` or `sandbox.agent.legacy-security` fields. Do not emit legacy `gh-proxy`, `local`, or `remote` GitHub mode values, `features.cli-proxy`, or `tools.cli-proxy`.

## Sandbox runtime profiles

| Runtime | Isolation and host access | Runner requirements | `runtime-install` | Services/host ports |
| --- | --- | --- | --- | --- |
| Omitted or `docker` | Docker, rootless AWF, strict network isolation, no host access | Linux and accessible Docker | Invalid | Invalid |
| `docker-sudo-iptables` | Docker, privileged AWF, legacy iptables, host access | Linux, Docker, passwordless `sudo` | Invalid | Supported |
| `gvisor` | `runsc`, rootless AWF, strict network isolation | Local Docker; generated install needs `sudo`, systemd, downloads | Supported | Invalid |
| `docker-sbx` | KVM microVM, rootless AWF, strict network isolation | KVM, local Docker, and Docker credentials; generated install also needs apt and `sudo` | Supported | Invalid |
| `cloud-hypervisor` | Preview KVM microVM, privileged launcher, strict network isolation | GitHub-hosted Ubuntu x86_64 with `/dev/kvm` | Invalid | Invalid |

Omission and explicit `runtime: docker` are both valid. Prefer omission unless the explicit profile helps communicate intent.

Use `runtime-install: false` only with `gvisor` or `docker-sbx` on a pre-provisioned runner. Docker sbx credentials remain required. `runner.topology: arc-dind` is compatible only with the Docker profiles; do not combine it with `gvisor`, `docker-sbx`, or `cloud-hypervisor`.

GitHub Actions `services:` with published ports and `sandbox.agent.allow-host-ports` require `docker-sudo-iptables`. Use `allow-host-ports` only for host daemons that cannot be declared as services.

## GitHub access profiles

| Mode | Select when | Important constraints |
| --- | --- | --- |
| `cli` | The agent should use pre-authenticated `gh` commands | Recommended for new MCP-capable workflows; required by integrity reactions; incompatible with Cloud Hypervisor |
| `mcp-local` | The workflow needs GitHub MCP tools or MCP-only fields | Local Docker GitHub MCP server; effective omitted-mode default for backward compatibility |
| `mcp-remote` | The workflow needs the hosted GitHub MCP service | Requires suitable additional authentication; do not use the GitHub Actions token |

For MCP-capable engines, omitted `tools.github.mode` resolves to `mcp-local`. Select `cli` explicitly for new CLI-based workflows. Engines without MCP support, including Pi, automatically derive `tools.github.mode: cli` and `tools.mcp-mode: cli`; do not select an MCP mode for them.

`toolsets` and `allowed` apply to either GitHub MCP mode; `version` and `args` apply only to `mcp-local`. Do not combine them with explicit `mode: cli`; the compiler ignores them with a warning. `allowed-repos`, `min-integrity`, `github-token`, and `github-app` remain meaningful in all modes.

`features.integrity-reactions: true` requires CLI GitHub access and is incompatible with explicit `mcp-local`, explicit `mcp-remote`, and `cloud-hypervisor`.

## MCP exposure

`tools.mcp-mode: cli` mounts user-facing MCP servers as CLI wrappers, including the GitHub MCP server when an MCP GitHub mode is selected. It does not select GitHub access and does not provide the authenticated `gh` CLI; configure `tools.github.mode` separately. Omit `mcp-mode`, or use `default`, for native MCP exposure.

## Minimal valid examples

```yaml
# Default Docker isolation and CLI GitHub access
tools:
  github:
    mode: cli
```

```yaml
# GitHub MCP tools
tools:
  github:
    mode: mcp-local
    toolsets: [issues, pull_requests]
```

```yaml
# Host service access
sandbox:
  agent:
    runtime: docker-sudo-iptables
services:
  postgres:
    image: postgres:18
    ports: [5432]
```

```yaml
# CLI GitHub access plus non-GitHub MCP CLI wrappers
tools:
  github:
    mode: cli
  mcp-mode: cli
```

Run `gh aw fix --write` to migrate removed or legacy fields.
