---
description: Example workflow demonstrating custom AWF configuration with command, args, and env
on:
  workflow_dispatch:
  
name: Custom AWF Example
engine: copilot
network:
  allowed:
    - example.com
  firewall: true

# Custom AWF Configuration
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

# Custom AWF Configuration Example

This workflow demonstrates the new custom AWF configuration capabilities:

1. **Custom Command**: Replace the standard AWF binary download with any command (e.g., Docker container, custom script)
2. **Custom Args**: Add additional arguments that are appended to the AWF command
3. **Custom Env**: Set environment variables on the execution step

## Use Cases

- **Custom AWF Image**: Run AWF from a custom Docker image with pre-configured settings
- **Custom Wrapper Script**: Use a shell script that sets up AWF with organization-specific configuration
- **Testing**: Use a modified AWF binary for testing new features
- **Debugging**: Add debug flags and environment variables for troubleshooting

## Example Configuration

The `sandbox.agent` object supports:
- `id`: Agent identifier ("awf" or "srt")
- `command`: Custom command to replace the AWF binary download (optional)
- `args`: Array of additional arguments to append (optional)
- `env`: Object with environment variables to set (optional)

When `command` is specified, the AWF installation step is skipped, and your custom command is used instead.

## Legacy Compatibility

The existing `type` field is still supported for backward compatibility:

```yaml
sandbox:
  agent:
    type: awf  # Still works!
```

Review the changes in this pull request.
