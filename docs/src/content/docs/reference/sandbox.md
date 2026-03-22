---
title: Sandbox Configuration
description: Configure sandbox environments for AI engines including AWF agent container, mounted tools, runtime environments, and MCP Gateway
sidebar:
  order: 1350
disable-agentic-editing: true
---

The `sandbox` field configures sandbox environments for AI engines (coding agents), providing two main capabilities:

1. **Coding Agent Sandbox** - Controls the agent runtime security using AWF (Agent Workflow Firewall)
2. **Model Context Protocol (MCP) Gateway** - Routes MCP server calls through a unified HTTP gateway

## Configuration

### Coding Agent Sandbox

Configure the coding agent sandbox type to control how the AI engine is isolated:

```yaml wrap
# Use AWF (Agent Workflow Firewall) - default
sandbox:
  agent: awf

# Disable coding agent sandbox (firewall only) - use with caution
sandbox:
  agent: false

# Or omit sandbox entirely to use the default (awf)
```

**Default Behavior**

If `sandbox` is not specified in your workflow, it defaults to `sandbox.agent: awf`. The coding agent sandbox is recommended for all workflows.

**Disabling Coding Agent Sandbox**

Setting `sandbox.agent: false` disables only the agent firewall while keeping the MCP gateway enabled. This reduces security isolation and should only be used when necessary. The MCP gateway cannot be disabled and remains active in all workflows.

### MCP Gateway (Experimental)

Route MCP server calls through a unified HTTP gateway:

```yaml wrap
features:
  mcp-gateway: true

sandbox:
  mcp:
    port: 8080
    api-key: "${{ secrets.MCP_GATEWAY_API_KEY }}"
```

### Combined Configuration

Use both coding agent sandbox and MCP gateway together:

```yaml wrap
features:
  mcp-gateway: true

sandbox:
  agent: awf
  mcp:
    port: 8080
```

## Coding Agent Sandbox Types

### AWF (Agent Workflow Firewall)

AWF is the default coding agent sandbox that provides network egress control through domain-based access controls. Network permissions are configured through the top-level [`network`](/gh-aw/reference/network/) field.

```yaml wrap
sandbox:
  agent: awf

network:
  firewall: true
  allowed:
    - defaults
    - python
    - "api.example.com"
```

#### Filesystem Access

AWF makes the host filesystem visible inside the container with appropriate permissions:

| Path Type | Mode | Examples |
|-----------|------|----------|
| User paths | Read-write | `$HOME`, `$GITHUB_WORKSPACE`, `/tmp` |
| System paths | Read-only | `/usr`, `/opt`, `/bin`, `/lib` |
| Docker socket | Hidden | `/var/run/docker.sock` (security) |

#### Host Binaries

All host binaries are available without explicit mounts: system utilities, `gh`, language runtimes, build tools, and anything installed via `apt-get` or setup actions. Verify with `which <tool>`.

> [!WARNING]
> Docker socket is hidden for security. Agents cannot spawn containers.

#### Environment Variables

AWF passes all environment variables via `--env-all`. The host `PATH` is captured as `AWF_HOST_PATH` and restored inside the container, preserving setup action tool paths.

> [!NOTE]
> Go's "trimmed" binaries require `GOROOT` - AWF automatically captures it after `actions/setup-go`.

#### Runtime Tools

Setup actions work transparently. Runtimes update `PATH`, which AWF captures and restores inside the container.

```yaml wrap
---
jobs:
  setup:
    steps:
      - uses: actions/setup-go@v5
        with:
          go-version: '1.25'
      - uses: actions/setup-python@v5
        with:
          python-version: '3.12'
---

Use `go build` or `python3` - both are available.
```

## MCP Gateway

The MCP Gateway routes all MCP server calls through a unified HTTP gateway, enabling centralized management, logging, and authentication for MCP tools.

## Feature Flags

Some sandbox features require feature flags:

| Feature | Flag | Description |
|---------|------|-------------|
| MCP Gateway | `mcp-gateway` | Enable MCP gateway routing |

Enable feature flags in your workflow:

```yaml wrap
features:
  mcp-gateway: true
```

## Related Documentation

- [Network Permissions](/gh-aw/reference/network/) - Configure network access controls
- [AI Engines](/gh-aw/reference/engines/) - Engine-specific configuration
- [Tools](/gh-aw/reference/tools/) - Configure MCP tools and servers
