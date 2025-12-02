---
title: Sandbox Configuration
description: Configure sandbox environments for AI engines including MCP Gateway and Sandbox Runtime (SRT)
sidebar:
  order: 1350
---

The `sandbox` field configures sandbox environments for AI engines, providing two main capabilities:

1. **Agent Sandbox** - Controls the agent runtime security (AWF or Sandbox Runtime)
2. **MCP Gateway** - Routes MCP server calls through a unified HTTP gateway

## Configuration

### Agent Sandbox

Configure the agent sandbox type to control how the AI engine is isolated:

```yaml wrap
# Use AWF (Agent Workflow Firewall) - default
sandbox:
  agent: awf

# Use Sandbox Runtime (SRT) - experimental
sandbox:
  agent: srt
```

### MCP Gateway (Experimental)

Route MCP server calls through a unified HTTP gateway:

```yaml wrap
features:
  mcp-gateway: true

sandbox:
  mcp:
    container: "ghcr.io/your-org/mcp-gateway"
    port: 8080
    api-key: "${{ secrets.MCP_GATEWAY_API_KEY }}"
```

### Combined Configuration

Use both agent sandbox and MCP gateway together:

```yaml wrap
features:
  mcp-gateway: true

sandbox:
  agent: awf
  mcp:
    container: "ghcr.io/your-org/mcp-gateway"
    port: 8080
```

## Agent Sandbox Types

### AWF (Agent Workflow Firewall)

AWF is the default agent sandbox that provides network egress control through domain-based access controls. Network permissions are configured through the top-level [`network`](/gh-aw/reference/network/) field.

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

### Sandbox Runtime (SRT)

:::caution[Experimental]
Sandbox Runtime is experimental and requires the `sandbox-runtime` feature flag.
:::

Sandbox Runtime provides enhanced isolation using Anthropic's sandbox technology. It supports custom filesystem configuration while network permissions are controlled by the top-level `network` field.

```yaml wrap
features:
  sandbox-runtime: true

sandbox:
  agent:
    type: srt
    config:
      filesystem:
        allowWrite: [".", "/tmp", "/home/runner/.copilot"]
        denyRead: ["/etc/passwd"]
      enableWeakerNestedSandbox: true

network:
  allowed:
    - defaults
    - python
```

#### SRT Configuration Options

| Field | Type | Description |
|-------|------|-------------|
| `filesystem.allowWrite` | `string[]` | Paths allowed for write access |
| `filesystem.denyRead` | `string[]` | Paths denied for read access |
| `filesystem.denyWrite` | `string[]` | Paths denied for write access |
| `ignoreViolations` | `object` | Map of command patterns to paths that should ignore violations |
| `enableWeakerNestedSandbox` | `boolean` | Enable weaker nested sandbox mode (recommended for Docker access) |

:::note[Network Configuration]
Network configuration for SRT is controlled by the top-level `network` field, not the sandbox config. This ensures consistent network policy across all sandbox types.
:::

## MCP Gateway

The MCP Gateway routes all MCP server calls through a unified HTTP gateway, enabling centralized management, logging, and authentication for MCP tools.

### Configuration Options

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `container` | `string` | Yes | Container image for the MCP gateway |
| `version` | `string` | No | Version tag for the container image |
| `port` | `integer` | No | HTTP server port (default: 8080) |
| `api-key` | `string` | No | API key for gateway authentication |
| `args` | `string[]` | No | Container execution arguments |
| `entrypointArgs` | `string[]` | No | Container entrypoint arguments |
| `env` | `object` | No | Environment variables for the gateway |

### How It Works

When MCP gateway is enabled:

1. A Docker container runs the gateway in the background
2. A health check verifies the gateway is ready
3. All MCP server configurations are transformed to route through the gateway
4. The gateway receives server configs via stdin in JSON format

### Example with Full Configuration

```yaml wrap
features:
  mcp-gateway: true

sandbox:
  mcp:
    container: "ghcr.io/githubnext/mcp-gateway"
    version: "v1.0.0"
    port: 9000
    api-key: "${{ secrets.MCP_GATEWAY_API_KEY }}"
    env:
      LOG_LEVEL: "debug"
    args: ["-v"]
```

## Legacy Format

For backward compatibility, the legacy string format is still supported:

```yaml wrap
# Legacy format (deprecated)
sandbox: sandbox-runtime

# Recommended format
sandbox:
  agent: srt
```

## Feature Flags

Some sandbox features require feature flags:

| Feature | Flag | Description |
|---------|------|-------------|
| Sandbox Runtime | `sandbox-runtime` | Enable SRT agent sandbox |
| MCP Gateway | `mcp-gateway` | Enable MCP gateway routing |

Enable feature flags in your workflow:

```yaml wrap
features:
  sandbox-runtime: true
  mcp-gateway: true
```

## Related Documentation

- [Network Permissions](/gh-aw/reference/network/) - Configure network access controls
- [AI Engines](/gh-aw/reference/engines/) - Engine-specific configuration
- [Tools](/gh-aw/reference/tools/) - Configure MCP tools and servers
