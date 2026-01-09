---
title: Integrating gh-aw-mcpg with gh-aw-firewall
description: Run the MCP gateway on the host and connect to it from an AWF (Agentic Workflow Firewall) container using Copilot CLI.
sidebar:
  order: 280
---

This guide explains how the **gh-aw-mcpg** gateway and **gh-aw-firewall (AWF)** work together in compiled workflows. It is educational only—you normally do **not** start mcpg or AWF yourself. The compiler provisions and wires these components automatically.

## Tested versions

- `ghcr.io/githubnext/gh-aw-mcpg:v0.0.10`
- `gh-aw-firewall v0.8.2` or later (includes required fixes)

## Architecture

```
┌────────────────────────────────────────────────────────────────────┐
│                           HOST MACHINE                              │
│                                                                     │
│   ┌─────────────────────┐                                          │
│   │   gh-aw-mcpg        │◄──────────────────────┐                  │
│   │   (Docker container)│                       │                  │
│   │   Port 80:8000      │                       │                  │
│   └─────────────────────┘                       │                  │
│            │                                    │                  │
│            │ spawns (via Docker socket)         │                  │
│            ▼                                    │                  │
│   ┌─────────────────────┐                       │                  │
│   │   GitHub MCP Server │                       │                  │
│   │   (Docker container)│                       │                  │
│   └─────────────────────┘                       │                  │
│                                                 │                  │
│   ┌─────────────────────────────────────────────┼────────────────┐ │
│   │                    AWF Network              │                │ │
│   │                                             │                │ │
│   │   ┌─────────────────┐    ┌─────────────────┼──┐             │ │
│   │   │   Agent         │    │   Squid Proxy      │             │ │
│   │   │   Container     │───▶│   172.30.0.10      │             │ │
│   │   │   172.30.0.20   │    │                    │             │ │
│   │   │                 │    │   CONNECT to       │─────────────┘ │
│   │   │   Copilot CLI   │    │   host.docker.     │               │
│   │   │   + MCP Client  │    │   internal:80      │               │
│   │   └─────────────────┘    └────────────────────┘               │ │
│   │                                                               │
│   └───────────────────────────────────────────────────────────────┘
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

## How it is provisioned

When you compile a workflow with the Copilot engine and AWF sandbox, the compiler:

- Starts the **gh-aw-mcpg** gateway container (including default GitHub/fetch/memory servers).
- Launches the AWF Squid proxy and agent containers with wiring to reach the gateway.
- Generates the MCP client configuration that points the agent to the gateway and injects it into the run.

You do not need to start Docker containers or create MCP config files yourself.

## What you configure in workflows

- **Frontmatter:** Set `engine: copilot` and `sandbox.agent: awf`.
- **Network allowlist:** Keep domains minimal (gateway host plus required GitHub/Copilot endpoints).
- **Tokens:** Use separate PATs for Copilot (`GITHUB_TOKEN`) and the GitHub MCP server (`GITHUB_PERSONAL_ACCESS_TOKEN`) with least-privilege scopes.
- **Optional MCP servers:** Add additional entries under `mcp-servers:` if you need more than the default GitHub/fetch/memory servers.

## Security model

- **Controlled egress:** All agent traffic exits through the AWF Squid proxy. The domain allowlist you set in frontmatter is the enforcement point.
- **Explicit host reachability:** Host access is opt-in; without it, agents cannot reach host services.
- **Gateway auth header:** The MCP client sends an opaque bearer value (any unguessable string) in `headers.Authorization`, which the gateway treats as a session identifier.
- **Token separation:** Use separate PATs for Copilot (`GITHUB_TOKEN`) and the GitHub MCP server (`GITHUB_PERSONAL_ACCESS_TOKEN`) with least-privilege scopes.
- **Minimal allowlist:** Keep the allowlist limited to the gateway and required GitHub/Copilot endpoints.
- **Auditability:** AWF firewall logs capture CONNECT attempts and allowlist decisions; review them when tightening access.

## Required tokens

| Token | Env var | Purpose |
|-------|---------|---------|
| GitHub Copilot token | `GITHUB_TOKEN` | Authenticates Copilot CLI |
| GitHub MCP token | `GITHUB_PERSONAL_ACCESS_TOKEN` | Authenticates the GitHub MCP server |

## Troubleshooting

### "No Bearer token" from gateway
- Ensure `headers.Authorization` is set in `/tmp/mcp-gateway-config.json`.
- Verify the file is mounted at `/tmp` in AWF.

### "fetch failed" or connection errors
- Confirm the gateway is healthy: `curl http://127.0.0.1:80/health`.
- Check `--enable-host-access` is present.
- Verify `host.docker.internal` appears in `--allow-domains`.

### "TCP_DENIED" in Squid logs
- Use AWF v0.8.2+ for the Safe_ports CONNECT fix.
- Confirm the domain is in the allowed list.

### "sys___init must be called"
- Update to the latest `gh-aw-mcpg` image.

### Gateway container issues
- Review logs: `docker logs mcpg-gateway`.
- Ensure the Docker socket is mounted.
- Confirm the MCP token is set and valid.

## Security considerations

1. `--enable-host-access` allows access to host services; use only in trusted environments.
2. The bearer value is session identification, not hardened auth. Keep the gateway on trusted networks.
3. `--env-all` passes all host environment variables into the agent; avoid storing extra secrets in the shell.
4. Keep the domain allowlist as small as possible.

## Related docs

- [Using MCPs](/gh-aw/guides/mcps/)
- [Network Configuration Guide](/gh-aw/guides/network-configuration/)
- [Security Guide](/gh-aw/guides/security/)
