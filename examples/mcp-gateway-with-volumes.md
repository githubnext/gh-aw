---
on: workflow_dispatch
engine: copilot
features:
  mcp-gateway: true

# Example: MCP Server with Volume Mounts
# This example demonstrates how to configure volume mounts for individual MCP servers.

sandbox:
  agent: awf
  mcp:
    # Container image for the gateway
    container: ghcr.io/example/mcp-gateway
    version: latest
    
    # Environment variables for the gateway
    env:
      LOG_LEVEL: debug
      DEBUG: "true"

tools:
  bash: ["*"]
  custom-mcp-server:
    container: "ghcr.io/example/data-server:latest"
    # Volume mounts (format: "source:dest:mode")
    # - source: host path
    # - dest: container path
    # - mode: "ro" (read-only) or "rw" (read-write)
    mounts:
      - "/host/data:/data:ro"           # Read-only data mount
      - "/host/config:/config:rw"       # Read-write config mount
    env:
      DATA_PATH: "/data"
---

# MCP Server with Volume Mounts

This workflow demonstrates how to configure volume mounts for individual MCP servers.

## Task

Show the contents of the data directory that was mounted from the host.

```bash
ls -la /data
```
