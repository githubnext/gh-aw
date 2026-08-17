---
title: Security Profile Selection
description: Select compatible sandbox runtime, GitHub access, and MCP exposure profiles for an agentic workflow.
sidebar:
  order: 1330
---

Three independent selectors define how an agentic workflow is isolated and how tools reach the agent:

| Selector | Controls | Does not control |
| --- | --- | --- |
| `sandbox.agent.runtime` | Agent isolation, AWF privileges, and host/service connectivity | How GitHub APIs are exposed |
| `tools.github.mode` | GitHub access through the `gh` CLI or a GitHub MCP server | How non-GitHub MCP servers are exposed |
| `tools.mcp-mode` | Whether user-facing non-GitHub MCP servers are exposed as native MCP tools or CLI wrappers; Copilot also wraps a selected GitHub MCP server | GitHub transport or sandbox isolation |

The policy proxy used by `tools.github.mode: cli` is an internal implementation detail. Do not use `gh-proxy` in new frontmatter.

## Sandbox runtime profiles

| `sandbox.agent.runtime` | Effective isolation and host access | Runner prerequisites | `runtime-install` | Services and host ports |
| --- | --- | --- | --- | --- |
| Omitted or `docker` | Docker container, rootless AWF, strict network isolation, no host access | Linux runner with an accessible Docker daemon | Not supported | Not supported |
| `docker-sudo-iptables` | Docker container, privileged AWF, legacy iptables networking, host access | Linux runner with Docker and passwordless host `sudo` | Not supported | Supported |
| `gvisor` | `runsc` user-space kernel, rootless AWF, strict network isolation | Local Docker; generated install also needs passwordless `sudo`, systemd, and download access | Supported; defaults to `true` | Not supported |
| `docker-sbx` | KVM microVM, rootless AWF, strict network isolation | KVM, local Docker, and `DOCKER_USERNAME`/`DOCKER_PAT`; generated installation also needs apt and passwordless `sudo` | Supported; defaults to `true` | Not supported |
| `cloud-hypervisor` | Preview KVM microVM, privileged launcher, strict network isolation | GitHub-hosted Ubuntu x86_64 with `/dev/kvm` | Not supported; release assets are always provisioned | Not supported |

Omitting `runtime` and setting `runtime: docker` are both valid and have the same effect. Omission is preferred when no explicit profile selection is needed.

`sandbox.agent.runtime-install: false` is valid only with `gvisor` or `docker-sbx`. It skips generated runtime provisioning and preflight checks; the runner must already provide the selected runtime. Docker sbx still validates and refreshes `DOCKER_USERNAME` and `DOCKER_PAT` before agent execution.

GitHub Actions `services:` with published ports and `sandbox.agent.allow-host-ports` both require `docker-sudo-iptables`. `runner.topology: arc-dind` is a runner topology, not a runtime, and is incompatible with `gvisor`, `docker-sbx`, and `cloud-hypervisor`.

## GitHub access profiles

| `tools.github.mode` | Effective access | Use when | Compatibility |
| --- | --- | --- | --- |
| `cli` | Pre-authenticated `gh` CLI protected by the host policy proxy; no GitHub MCP server | Shell-based GitHub access with the smallest tool schema; required by integrity reactions | Not supported by `cloud-hypervisor` |
| `mcp-local` | Local Docker GitHub MCP server | GitHub MCP tools or MCP-only fields are required | Historical effective default when mode is omitted |
| `mcp-remote` | Hosted GitHub MCP service | Hosted-only toolsets are required and suitable additional authentication is available | Do not use the GitHub Actions token as remote MCP authentication |

For MCP-capable engines, an omitted mode resolves to `mcp-local` for backward compatibility. New workflows should select `cli` explicitly unless they need GitHub MCP tools. `features.integrity-reactions: true` resolves an omitted mode to `cli`. Engines without native MCP support, including Pi, automatically derive both `tools.github.mode: cli` and `tools.mcp-mode: cli`; do not set an MCP GitHub mode for them.

The fields `toolsets` and `allowed` configure either GitHub MCP mode; `version` and `args` configure only `mcp-local`. All four are ignored with an explicit `tools.github.mode: cli` and produce a compiler warning. Policy fields such as `allowed-repos`, `min-integrity`, and `github-token` remain meaningful in every GitHub access mode. `github-app` configures GitHub MCP authentication and applies only to `mcp-local` and `mcp-remote`.

## MCP exposure profile

`tools.mcp-mode: cli` exposes user-facing non-GitHub MCP servers as CLI wrappers on `PATH`. With the Copilot engine, it also wraps the GitHub MCP server when `mcp-local` or `mcp-remote` is selected; other MCP-capable engines keep the selected GitHub server as a native MCP server. It does not select `tools.github.mode: cli` and does not turn the GitHub MCP server into the authenticated `gh` CLI. Leave `tools.mcp-mode` omitted, or set it to `default`, for native MCP exposure.

The legacy `tools.cli-proxy: true` maps to `tools.mcp-mode: cli`. This field is unrelated to the internal host policy proxy used for CLI GitHub access.

## Minimal configurations

Use default Docker isolation with CLI GitHub access:

```aw wrap
---
tools:
  github:
    mode: cli
---
```

Use GitHub MCP tools with the historical local transport:

```aw wrap
---
tools:
  github:
    mode: mcp-local
    toolsets: [issues, pull_requests]
---
```

Reach a declared service from the agent:

```aw wrap
---
sandbox:
  agent:
    runtime: docker-sudo-iptables

services:
  postgres:
    image: postgres:18
    ports:
      - 5432:5432
---
```

Expose non-GitHub MCP servers as CLI wrappers:

```aw wrap
---
tools:
  github:
    mode: cli
  mcp-mode: cli
---
```

## Invalid combinations

The compiler rejects these combinations:

| Invalid configuration | Use instead |
| --- | --- |
| `runtime-install` with Docker, `docker-sudo-iptables`, or `cloud-hypervisor` | Remove it, or select `gvisor` or `docker-sbx` |
| `services:` ports or `allow-host-ports` outside `docker-sudo-iptables` | Select `docker-sudo-iptables`, or remove host connectivity |
| `arc-dind` with `gvisor`, `docker-sbx`, or `cloud-hypervisor` | Use the Docker runtime profile |
| `cloud-hypervisor` with `tools.github.mode: cli` or integrity reactions | Use `mcp-local`, or select another runtime |
| Integrity reactions with `mcp-local` or `mcp-remote` | Use `tools.github.mode: cli` |
| A non-MCP engine with `mcp-local` or `mcp-remote` | Omit the mode and let the compiler derive CLI access |
| Both `tools.cli-proxy` and `tools.mcp-mode` | Keep only `tools.mcp-mode: cli` |

Run `gh aw fix --write` to migrate removed `sandbox.agent.sudo`, `sandbox.agent.legacy-security`, legacy GitHub modes, `features.cli-proxy`, and `tools.cli-proxy`.
