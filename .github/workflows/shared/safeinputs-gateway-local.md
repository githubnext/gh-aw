---
sandbox:
  safe-inputs:
    command: ./gh-aw mcp-gateway
    steps:
      - name: Build local gh-aw binary
        run: |
          echo "Building local gh-aw binary for testing..."
          make build
          ./gh-aw --version
---
<!--
## Safe-Inputs Gateway Configuration (Local Binary)

This shared workflow configures the safe-inputs MCP gateway to use the local compiled `./gh-aw` binary instead of downloading from releases.

**Use this for testing only** - workflows in production should use the default configuration which downloads a pinned version from releases.

### Usage

```yaml
imports:
  - shared/safeinputs-gateway-local.md
safe-inputs:
  my-tool:
    description: "A custom safe-input tool"
    inputs:
      param:
        type: string
        description: "A parameter"
        required: true
    script: |
      console.log("Running with param:", param);
      return { result: "success" };
```

### What This Does

1. **Builds the local binary**: Runs `make build` to compile `./gh-aw`
2. **Starts the gateway**: Uses `./gh-aw mcp-gateway` to start the safe-inputs MCP gateway
3. **Configures the agent**: The agent's MCP configuration will point to the gateway's HTTP endpoint instead of running the safe-inputs server directly

### Default Configuration

By default (without importing this shared workflow), the compiler:
- Downloads a pinned version of gh-aw from GitHub releases
- Starts the gateway using `/tmp/gh-aw-cli mcp-gateway`
- Uses port 8088 for the gateway HTTP server

### Customization

You can override the configuration in your workflow:

```yaml
imports:
  - shared/safeinputs-gateway-local.md
sandbox:
  safe-inputs:
    command: ./gh-aw mcp-gateway
    port: 9000
    args:
      - --logs-dir
      - /tmp/gateway-logs
```

### Port Configuration

The gateway defaults to port 8088. You can customize it:

```yaml
sandbox:
  safe-inputs:
    port: 9000
```
-->

