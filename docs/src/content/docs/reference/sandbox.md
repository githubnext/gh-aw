---
title: Sandbox Configuration
description: Configure sandbox environments for AI engines including AWF agent container, mounted tools, runtime environments, and MCP Gateway
sidebar:
  order: 1350
disable-agentic-editing: true
---

The `sandbox` field configures sandbox environments for AI engines, providing two main capabilities:

1. **Agent Sandbox** - Controls the agent runtime security using AWF (Agent Workflow Firewall)
2. **Model Context Protocol (MCP) Gateway** - Routes MCP server calls through a unified HTTP gateway

## Configuration

### Agent Sandbox

Configure the agent sandbox type to control how the AI engine is isolated:

```yaml wrap
# Use AWF (Agent Workflow Firewall) - default
sandbox:
  agent: awf

# Disable agent sandbox (firewall only) - use with caution
sandbox:
  agent: false

# Or omit sandbox entirely to use the default (awf)
```

> [!NOTE]
> Default Behavior
> If `sandbox` is not specified in your workflow, it defaults to `sandbox.agent: awf`. The agent sandbox is recommended for all workflows.

> [!CAUTION]
> Disabling Agent Sandbox
> Setting `sandbox.agent: false` disables only the agent firewall while keeping the MCP gateway enabled. This reduces security isolation and should only be used when necessary. The MCP gateway cannot be disabled and remains active in all workflows.

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

Use both agent sandbox and MCP gateway together:

```yaml wrap
features:
  mcp-gateway: true

sandbox:
  agent: awf
  mcp:
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

#### Chroot Mode

AWF v0.15.0+ uses **chroot mode by default** to provide transparent host filesystem access while maintaining network isolation via iptables. This eliminates explicit volume mounts and environment variable configuration.

```text
┌─────────────────────────────────────────────┐
│  AWF Container (chroot)                     │
│  • Full filesystem visibility               │
│  • All host binaries available              │
│  • Network: RESTRICTED via iptables/Squid   │
└─────────────────────┬───────────────────────┘
                      ▼
              Allowed domains only
```

#### Filesystem Access

AWF chroot mode makes the host filesystem visible inside the container with appropriate permissions:

| Path Type | Mode | Examples |
|-----------|------|----------|
| User paths | Read-write | `$HOME`, `$GITHUB_WORKSPACE`, `/tmp` |
| System paths | Read-only | `/usr`, `/opt`, `/bin`, `/lib` |
| Docker socket | Hidden | `/var/run/docker.sock` (security) |

Custom mounts can still be added via `sandbox.agent.mounts` for paths that need different permissions.

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

#### Custom AWF Configuration

Use custom commands, arguments, and environment variables to replace the standard AWF installation with a custom setup:

```yaml wrap
sandbox:
  agent:
    id: awf
    command: "/usr/local/bin/custom-awf-wrapper"
    args:
      - "--custom-logging"
      - "--debug-mode"
    env:
      AWF_CUSTOM_VAR: "custom_value"
      DEBUG_LEVEL: "verbose"
```

##### Custom Mounts

Add custom container mounts to make host paths available inside the AWF container:

```yaml wrap
sandbox:
  agent:
    id: awf
    mounts:
      - "/host/data:/data:ro"
      - "/usr/local/bin/custom-tool:/usr/local/bin/custom-tool:ro"
      - "/tmp/cache:/cache:rw"
```

Mount syntax follows Docker's format: `source:destination:mode`
- `source`: Path on the host system
- `destination`: Path inside the container
- `mode`: Either `ro` (read-only) or `rw` (read-write)

Custom mounts are useful for:
- Providing access to datasets or configuration files
- Making custom tools available in the container
- Sharing cache directories between host and container

| Field | Type | Description |
|-------|------|-------------|
| `id` | `string` | Agent identifier: `awf` |
| `command` | `string` | Custom command to replace AWF binary installation |
| `args` | `string[]` | Additional arguments appended to the command |
| `env` | `object` | Environment variables set on the execution step |
| `mounts` | `string[]` | Container mounts using syntax `source:destination:mode` |

When `command` is specified, the standard AWF installation is skipped and your custom command is used instead.

## Deprecated: Sandbox Runtime (SRT)

> [!CAUTION]
> Removed
> Sandbox Runtime (SRT) support has been removed. AWF is now the only supported sandbox implementation.

### Migration

Legacy workflows using `sandbox.agent: srt` or `sandbox: sandbox-runtime` are automatically migrated to AWF during workflow parsing. No manual changes are required.

**Before (automatically migrated):**
```yaml wrap
sandbox:
  agent: srt
```

**After (transparent conversion):**
```yaml wrap
sandbox:
  agent: awf
```

If your workflow previously used SRT, it will now use AWF with the same network permissions configured in the `network` field. AWF provides network egress control while maintaining compatibility with existing workflow configurations.

## MCP Gateway

The MCP Gateway routes all MCP server calls through a unified HTTP gateway, enabling centralized management, logging, and authentication for MCP tools.

### Configuration Options

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `command` | `string` | No | Custom command to execute (mutually exclusive with `container`) |
| `container` | `string` | No | Container image for the MCP gateway (mutually exclusive with `command`) |
| `version` | `string` | No | Version tag for the container image |
| `port` | `integer` | No | HTTP server port (default: 8080) |
| `api-key` | `string` | No | API key for gateway authentication |
| `args` | `string[]` | No | Command/container execution arguments |
| `entrypointArgs` | `string[]` | No | Container entrypoint arguments (only valid with `container`) |
| `env` | `object` | No | Environment variables for the gateway |

> [!NOTE]
> Execution Modes
> The MCP gateway supports two execution modes:
> 1. **Custom command** - Use `command` field to specify a custom binary or script
> 2. **Container** - Use `container` field for Docker-based execution
>
> The `command` and `container` fields are mutually exclusive - only one can be specified.
> You must specify either `command` or `container` to use the MCP gateway feature.

### How It Works

When MCP gateway is configured:

1. The gateway starts using the specified execution mode (command or container)
2. A health check verifies the gateway is ready
3. All MCP server configurations are transformed to route through the gateway
4. The gateway receives server configs via a configuration file

### Example: Custom Command Mode

```yaml wrap
features:
  mcp-gateway: true

sandbox:
  mcp:
    command: "/usr/local/bin/mcp-gateway"
    args: ["--port", "9000", "--verbose"]
    env:
      LOG_LEVEL: "debug"
```

### Example: Container Mode

```yaml wrap
features:
  mcp-gateway: true

sandbox:
  mcp:
    container: "ghcr.io/github/gh-aw-mcpg:latest"
    args: ["--rm", "-i"]
    entrypointArgs: ["--routed", "--listen", "0.0.0.0:8000", "--config-stdin"]
    port: 8000
    env:
      LOG_LEVEL: "info"
```

## Legacy Format

For backward compatibility, legacy formats are still supported:

```yaml wrap
# Legacy string format - automatically migrated to AWF
sandbox: sandbox-runtime

# Legacy object format with 'type' field (deprecated)
sandbox:
  agent:
    type: awf

# Recommended format with 'id' field
sandbox:
  agent:
    id: awf
```

The `id` field replaces the legacy `type` field in the object format. When both are present, `id` takes precedence.

> [!NOTE]
> SRT Migration
> The legacy string format `sandbox: sandbox-runtime` is automatically converted to `sandbox.agent: awf` during workflow parsing.

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

> [!NOTE]
> Removed Feature Flags
> The `sandbox-runtime` feature flag has been removed. It is no longer recognized and will be ignored if present in workflow configurations.

## Related Documentation

- [Network Permissions](/gh-aw/reference/network/) - Configure network access controls
- [AI Engines](/gh-aw/reference/engines/) - Engine-specific configuration
- [Tools](/gh-aw/reference/tools/) - Configure MCP tools and servers
