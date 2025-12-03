---
description: Example workflow demonstrating custom agent configuration with command, args, and env
on:
  workflow_dispatch:
  
name: Custom Agent Example
engine: copilot
network:
  allowed:
    - example.com
  firewall: true

# Custom Agent Configuration (works for both AWF and SRT)
# This example shows how to use a custom command to replace the standard AWF binary
sandbox:
  agent:
    id: awf  # Agent identifier (awf or srt)
    command: "docker run --rm -it my-custom-awf-image"  # Custom command replaces AWF binary download
    args:
      - "--custom-logging"  # Additional arguments appended to AWF command
      - "--debug-mode"
    env:
      AWF_CUSTOM_VAR: "custom_value"  # Environment variables set on the execution step
      DEBUG_LEVEL: "verbose"

permissions:
  contents: read
  
tools:
  github:
    toolsets: [repos]
---

# Custom Agent Configuration Example

This workflow demonstrates the custom agent configuration capabilities that work for **both AWF and SRT**:

1. **Custom Command**: Replace the standard AWF or SRT installation with any command (e.g., Docker container, custom script)
2. **Custom Args**: Add additional arguments that are appended to the command
3. **Custom Env**: Set environment variables on the execution step

## Use Cases

### For AWF (Agent Workflow Firewall)
- **Custom AWF Image**: Run AWF from a custom Docker image with pre-configured settings
- **Custom Wrapper Script**: Use a shell script that sets up AWF with organization-specific configuration
- **Testing**: Use a modified AWF binary for testing new features
- **Debugging**: Add debug flags and environment variables for troubleshooting

### For SRT (Sandbox Runtime)
- **Custom SRT Wrapper**: Use a custom wrapper around the Anthropic Sandbox Runtime
- **Pre-configured Container**: Run SRT from a Docker image with custom settings
- **Custom Isolation**: Implement custom sandboxing logic that wraps SRT
- **Testing & Development**: Use a modified SRT setup for testing

## Example Configurations

### AWF with Custom Command

```yaml
sandbox:
  agent:
    id: awf
    command: "docker run --rm my-custom-awf"
    args: ["--debug-mode", "--verbose"]
    env:
      AWF_LOG_LEVEL: "debug"
```

### SRT with Custom Command

```yaml
features:
  sandbox-runtime: true
sandbox:
  agent:
    id: srt
    command: "custom-srt-wrapper"
    args: ["--custom-arg"]
    env:
      SRT_DEBUG: "true"
```

## Configuration Reference

The `sandbox.agent` object supports:
- `id`: Agent identifier ("awf" or "srt")
- `command`: Custom command to replace the default installation (optional)
- `args`: Array of additional arguments to append (optional)
- `env`: Object with environment variables to set (optional)

When `command` is specified, the installation step is skipped, and your custom command is used instead.

## Legacy Compatibility

The existing `type` field is still supported for backward compatibility:

```yaml
sandbox:
  agent:
    type: awf  # Still works!
```

Review the changes in this pull request.
