---
name: http_mcp_headers
description: HTTP MCP Header Secret Support - Implementation Summary
---


# HTTP MCP Header Secret Support - Implementation Summary

This document demonstrates the complete implementation of HTTP MCP header secret support for the copilot engine.

## Problem Statement

When using HTTP MCP tools with headers containing GitHub Actions secrets, the generated mcp-config.json needs to:

1. Extract secrets from headers (e.g., `${{ secrets.DD_API_KEY }}`)
2. Declare those env variables in the execution step
3. Configure the MCP config's "env" section to passthrough those variables
4. Use the passed variables in the headers section

## Example Workflow

```markdown
on:
  workflow_dispatch:
permissions:
  contents: read
engine: copilot
mcp-servers:
  datadog:
    type: http
    url: "https://mcp.datadoghq.com/api/unstable/mcp-server/mcp"
    headers:
      DD_API_KEY: "${{ secrets.DD_API_KEY }}"
      DD_APPLICATION_KEY: "${{ secrets.DD_APPLICATION_KEY }}"
      DD_SITE: "${{ secrets.DD_SITE || 'datadoghq.com' }}"
    allowed:
      - search_datadog_dashboards
      - search_datadog_slos
      - search_datadog_metrics
      - get_datadog_metric

# Datadog Dashboard Search

Search for Datadog dashboards and provide a summary.
```

## Generated Output

### 1. MCP Config (mcp-config.json)

```json
{
  "mcpServers": {
    "datadog": {
      "type": "http",
      "url": "https://mcp.datadoghq.com/api/unstable/mcp-server/mcp",
      "headers": {
        "DD_API_KEY": "${DD_API_KEY}",
        "DD_APPLICATION_KEY": "${DD_APPLICATION_KEY}",
        "DD_SITE": "${DD_SITE}"
      },
      "tools": [
        "search_datadog_dashboards",
        "search_datadog_slos",
        "search_datadog_metrics",
        "get_datadog_metric"
      ],
      "env": {
        "DD_API_KEY": "\\${DD_API_KEY}",
        "DD_APPLICATION_KEY": "\\${DD_APPLICATION_KEY}",
        "DD_SITE": "\\${DD_SITE}"
      }
    }
  }
}
```

### 2. Execution Step Environment Variables

```yaml
env:
  DD_API_KEY: ${{ secrets.DD_API_KEY }}
  DD_APPLICATION_KEY: ${{ secrets.DD_APPLICATION_KEY }}
  DD_SITE: ${{ secrets.DD_SITE || 'datadoghq.com' }}
  GH_AW_MCP_CONFIG: /home/runner/.copilot/mcp-config.json
  COPILOT_GITHUB_TOKEN: ${{ secrets.COPILOT_GITHUB_TOKEN }}
  # ... other env vars
```

## Implementation Details

### Key Functions

1. **extractSecretsFromValue(value string)** - Extracts secret expressions from a string
   - Parses `${{ secrets.VAR_NAME }}` patterns
   - Handles default values: `${{ secrets.VAR || 'default' }}`
   - Returns map of variable names to full expressions

2. **extractSecretsFromHeaders(headers map[string]string)** - Extracts all secrets from HTTP headers
   - Iterates through all header values
   - Collects all unique secret expressions
   - Returns consolidated map of secrets

3. **replaceSecretsWithEnvVars(value string, secrets map[string]string)** - Replaces secret expressions with env var references
   - Transforms `${{ secrets.DD_API_KEY }}` to `${DD_API_KEY}`
   - Used in MCP config headers rendering

4. **collectHTTPMCPHeaderSecrets(tools map[string]any)** - Collects secrets from all HTTP MCP tools
   - Scans all tools for HTTP MCP configurations
   - Extracts secrets from each tool's headers
   - Returns consolidated map for execution step env

### Rendering Logic

#### In renderSharedMCPConfig (mcp-config.go):

1. **Extract secrets** when rendering HTTP MCP configs for copilot engine
2. **Add env section** to property order when secrets are found
3. **Render headers** with env var references instead of secret expressions
4. **Render env** with passthrough syntax (`\${VAR_NAME}`)

#### In GetExecutionSteps (copilot_engine.go):

1. **Collect all HTTP MCP header secrets** from workflow tools
2. **Add to execution step env map** with secret expressions

## Security Benefits

1. **Secrets never appear in MCP config** - Only env var references
2. **Proper GitHub Actions secret handling** - Uses `${{ secrets.* }}` syntax
3. **Environment isolation** - Each MCP server receives only its required secrets
4. **Consistent pattern** - Matches existing GitHub remote MCP server implementation

## Test Coverage

### Unit Tests (mcp_http_headers_test.go)
- extractSecretsFromValue
- extractSecretsFromHeaders
- replaceSecretsWithEnvVars
- collectHTTPMCPHeaderSecrets
- renderSharedMCPConfig with HTTP headers

### Integration Tests (copilot_mcp_http_integration_test.go)
- Single HTTP MCP tool with secrets
- Multiple HTTP MCP tools
- HTTP MCP without secrets
- Property ordering
- Env variable sorting

All tests pass ✓
