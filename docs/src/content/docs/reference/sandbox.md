---
title: Sandbox Configuration
description: Configure sandbox environments for AI engines including Sandbox Runtime (SRT)
sidebar:
  order: 1350
---

The `sandbox` field configures sandbox environments for AI engines, controlling the agent runtime security (AWF or Sandbox Runtime).

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

# Disable firewall for the agent
sandbox:
  agent: false
```

#### Disabling the Firewall

To disable the firewall for the agent while keeping network permissions for content sanitization:

```yaml wrap
engine: copilot
network:
  allowed:
    - defaults
    - python
sandbox:
  agent: false
```

When `sandbox.agent: false`:
- The agent runs without firewall enforcement
- Network permissions still apply for content sanitization
- Useful during development or when the firewall is incompatible with your workflow
- For production workflows, enabling the firewall is recommended for better security

:::note
Setting `sandbox.agent: false` replaces the deprecated `network.firewall: false` configuration.
:::

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

#### Custom AWF Configuration

Use custom commands, arguments, and environment variables to replace the standard AWF installation with a custom setup:

```yaml wrap
sandbox:
  agent:
    id: awf
    command: "docker run --rm my-custom-awf-image"
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
| `id` | `string` | Agent identifier: `awf` or `srt` |
| `command` | `string` | Custom command to replace AWF binary installation |
| `args` | `string[]` | Additional arguments appended to the command |
| `env` | `object` | Environment variables set on the execution step |
| `mounts` | `string[]` | Container mounts using syntax `source:destination:mode` |

When `command` is specified, the standard AWF installation is skipped and your custom command is used instead.

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

#### Custom SRT Configuration

Similar to AWF, SRT supports custom commands, arguments, and environment variables:

```yaml wrap
features:
  sandbox-runtime: true

sandbox:
  agent:
    id: srt
    command: "custom-srt-wrapper"
    args:
      - "--custom-arg"
      - "--debug"
    env:
      SRT_DEBUG: "true"
      SRT_CUSTOM_VAR: "test_value"
    config:
      filesystem:
        allowWrite: [".", "/tmp"]
```

When `command` is specified, the standard SRT installation is skipped. The `config` field can still be used for filesystem configuration.

## Legacy Format

For backward compatibility, legacy formats are still supported:

```yaml wrap
# Legacy string format (deprecated)
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

## Feature Flags

Some sandbox features require feature flags:

| Feature | Flag | Description |
|---------|------|-------------|
| Sandbox Runtime | `sandbox-runtime` | Enable SRT agent sandbox |

Enable feature flags in your workflow:

```yaml wrap
features:
  sandbox-runtime: true
```

## Related Documentation

- [Network Permissions](/gh-aw/reference/network/) - Configure network access controls
- [AI Engines](/gh-aw/reference/engines/) - Engine-specific configuration
- [Tools](/gh-aw/reference/tools/) - Configure MCP tools and servers
