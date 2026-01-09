---
title: Integrating gh-aw-mcpg with gh-aw-firewall
description: Run the MCP gateway on the host and connect to it from an AWF (Agentic Workflow Firewall) container using Copilot CLI.
sidebar:
  order: 280
---

This guide shows you how to run the **gh-aw-mcpg** gateway on the host and let **gh-aw-firewall (AWF)** agents reach it over HTTP from inside the firewall, with an external view of the security boundaries.

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

## Prerequisites

- Docker with access to the Docker socket
- `gh-aw-firewall` v0.8.2 or later
- `gh-aw-mcpg` image `ghcr.io/githubnext/gh-aw-mcpg:v0.0.10` or later

## Step 1: Start the MCP gateway container

The gateway ships with default MCP servers (GitHub, fetch, memory).

```bash
export GITHUB_PERSONAL_ACCESS_TOKEN="YOUR_GITHUB_PERSONAL_ACCESS_TOKEN"

docker run -d --name mcpg-gateway \
  --restart unless-stopped \
  -p 80:8000 \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -e "GITHUB_PERSONAL_ACCESS_TOKEN=${GITHUB_PERSONAL_ACCESS_TOKEN}" \
  ghcr.io/githubnext/gh-aw-mcpg:v0.0.10
```

Check readiness:

```bash
curl http://127.0.0.1:80/health
# Returns: OK
```

:::note
The container listens on port 8000 internally and is published on port 80 on the host.
:::

:::tip
If port 80 requires elevated privileges on your host, change the mapping (for example `-p 8080:8000`) and update the MCP client URL to `http://host.docker.internal:8080/mcp/github`.
:::

### Optional: custom gateway configuration

Create `/tmp/mcpg-config.json`:

```json
{
  "mcpServers": {
    "github": {
      "type": "local",
      "container": "ghcr.io/github/github-mcp-server:latest",
      "env": {
        "GITHUB_PERSONAL_ACCESS_TOKEN": ""
      }
    }
  }
}
```

Mount it when starting the container:

```bash
docker run -d --name mcpg-gateway \
  --restart unless-stopped \
  -p 80:8000 \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -v /tmp/mcpg-config.json:/config.json:ro \
  -e "GITHUB_PERSONAL_ACCESS_TOKEN=${GITHUB_PERSONAL_ACCESS_TOKEN}" \
  -e "CONFIG_FILE=/config.json" \
  ghcr.io/githubnext/gh-aw-mcpg:v0.0.10
```

## Step 2: Create MCP client config for Copilot

Save `/tmp/mcp-gateway-config.json`:

```json
{
  "mcpServers": {
    "github-gateway": {
      "type": "http",
      "url": "http://host.docker.internal/mcp/github",
      "headers": {
      "Authorization": "Bearer SESSION_TOKEN"
      },
      "tools": ["*"]
    }
  }
}
```

Key settings:

- `type: "http"` connects over HTTP instead of spawning a local process.
- `host.docker.internal` reaches the host from inside AWF containers.
- `headers` carries a Bearer token for gateway authentication; use any opaque string to identify the session (for example, `Bearer awf-session`).

## Step 3: Run AWF with Copilot CLI

```bash
export GITHUB_TOKEN="YOUR_COPILOT_TOKEN"
export GITHUB_PERSONAL_ACCESS_TOKEN="YOUR_GITHUB_MCP_TOKEN"

sudo -E awf \
  --build-local \
  --env-all \
  --enable-host-access \
  --env "GITHUB_TOKEN=${GITHUB_TOKEN}" \
  --env "GITHUB_PERSONAL_ACCESS_TOKEN=${GITHUB_PERSONAL_ACCESS_TOKEN}" \
  --mount /tmp:/tmp:rw \
  --allow-domains 'host.docker.internal,api.github.com,api.enterprise.githubcopilot.com,*.githubusercontent.com,github.com,registry.npmjs.org,registry.npmjs.com' \
  -- npx -y @github/copilot@0.0.365 \
    --disable-builtin-mcps \
    --additional-mcp-config @/tmp/mcp-gateway-config.json \
    --allow-all-tools \
    --allow-all-paths \
    --prompt "List the 3 most recent open issues from containerd/runwasi"
```

### Command breakdown

| Flag | Purpose |
|------|---------|
| `--build-local` | Build AWF containers from source |
| `--env-all` | Pass all host environment variables |
| `--enable-host-access` | Adds `host.docker.internal` to agent and Squid |
| `--mount /tmp:/tmp:rw` | Shares MCP config into the agent container |
| `--allow-domains ...` | Whitelists gateway and GitHub endpoints |
| `--disable-builtin-mcps` | Prevents spawning the built-in GitHub MCP |
| `--additional-mcp-config` | Points Copilot CLI to the gateway config |

## Security model

- **Controlled egress:** All agent traffic exits through the AWF Squid proxy. The domain allowlist you pass to `--allow-domains` is the enforcement point.
- **Explicit host reachability:** `--enable-host-access` is required for agents to talk to `host.docker.internal`; without it, host services remain unreachable.
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
