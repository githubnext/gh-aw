---
mcp-servers:
  datadog:
    container: "mcp/datadog"
    env:
      DD_API_KEY: "${{ secrets.DD_API_KEY }}"
      DD_APP_KEY: "${{ secrets.DD_APP_KEY }}"
      DD_SITE: "${{ secrets.DD_SITE || 'datadoghq.com' }}"
    network:
      allowed:
        - datadoghq.com
        - datadoghq.eu
        - ddog-gov.com
        - us5.datadoghq.com
        - ap1.datadoghq.com
    allowed:
      - get-monitors
      - get-monitor
      - get-dashboards
      - get-dashboard
      - get-metrics
      - get-metric-metadata
      - get-events
      - get-incidents
      - search-logs
      - aggregate-logs
---

## Datadog MCP Server

This shared configuration provides Datadog MCP server integration for monitoring, observability, and log analysis.

### Available Tools

- **get-monitors**: Fetch monitors with optional filtering by group states and tags
- **get-monitor**: Get details of a specific monitor by ID
- **get-dashboards**: List all dashboards in your Datadog account
- **get-dashboard**: Get a specific dashboard by ID with its full definition
- **get-metrics**: List available metrics in your Datadog account
- **get-metric-metadata**: Get metadata for a specific metric (unit, type, description)
- **get-events**: Fetch events within a specified time range
- **get-incidents**: List incidents with optional filtering and pagination
- **search-logs**: Search logs with advanced query filtering, time ranges, and sorting
- **aggregate-logs**: Perform analytics and aggregations on log data with grouping

### Setup

1. **Create Datadog API Keys**:
   - Log in to your Datadog account
   - Go to Organization Settings > API Keys to create an API key
   - Go to Organization Settings > Application Keys to create an application key

2. **Add Repository Secrets**:
   - `DD_API_KEY`: Your Datadog API key (required)
   - `DD_APP_KEY`: Your Datadog Application key (required)
   - `DD_SITE`: Your Datadog site domain (optional, defaults to `datadoghq.com`)

3. **Include in Your Workflow**:
   ```yaml
   imports:
     - shared/mcp/datadog.md
   ```

### Regional Endpoints

The `DD_SITE` secret should match your Datadog region:

- **US (Default)**: `datadoghq.com`
- **EU**: `datadoghq.eu`
- **US3 (GovCloud)**: `ddog-gov.com`
- **US5**: `us5.datadoghq.com`
- **AP1**: `ap1.datadoghq.com`

### Example Usage

```markdown
Search for error logs in the web-app service from the last hour and summarize the most common errors.
```

The AI agent will automatically use the appropriate Datadog tools (like `search-logs` or `aggregate-logs`) to fetch and analyze the data.

### Permissions

This configuration requires network access to Datadog API endpoints. The network allowlist includes all major Datadog regional domains.

### Troubleshooting

**403 Forbidden Errors**: Verify that:
- Your API key and Application key are correct
- The keys have necessary permissions to access requested resources
- You're using the correct endpoint for your region
- Your Datadog account has access to the requested data

**Reference**: [Datadog MCP Server Documentation](https://github.com/GeLi2001/datadog-mcp-server)
