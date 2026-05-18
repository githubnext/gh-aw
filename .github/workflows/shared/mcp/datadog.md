---
mcp-servers:
  datadog:
    url: "https://mcp.datadoghq.com/api/unstable/mcp-server/mcp?toolsets=core"
    headers:
      DD_API_KEY: "${{ secrets.DD_API_KEY }}"
      DD_APPLICATION_KEY: "${{ secrets.DD_APPLICATION_KEY || secrets.DD_APP_KEY }}"
      DD_SITE: "${{ secrets.DD_SITE || 'datadoghq.com' }}"
    allowed:
      - search_datadog_spans
      - get_datadog_trace
      - search_datadog_dashboards
      - search_datadog_slos
      - search_datadog_metrics
      - get_datadog_metric
---

<!--

Datadog MCP Server
Observability and monitoring platform integration

Provides access to the official Datadog MCP Server for observability data.
Documentation: https://docs.datadoghq.com/bits_ai/mcp_server/

This shared configuration provides Datadog MCP server integration for monitoring, 
observability, and log analysis via HTTP API.

Allowed tools in this shared import:
  - search_datadog_spans
  - get_datadog_trace
  - search_datadog_dashboards
  - search_datadog_slos
  - search_datadog_metrics
  - get_datadog_metric
#
Setup:
  1. Create Datadog API Keys:
     - Log in to your Datadog account
     - Go to Organization Settings > API Keys to create an API key
     - Go to Organization Settings > Application Keys to create an application key
#
  2. Add Repository Secrets:
     - DD_API_KEY: Your Datadog API key (required)
       - DD_APPLICATION_KEY: Your Datadog Application key (required)
       - DD_APP_KEY: Legacy alias for the Datadog Application key (optional fallback)
     - DD_SITE: Your Datadog site domain (optional, defaults to datadoghq.com)
#
  3. Include in Your Workflow:
     imports:
       - shared/mcp/datadog.md
#
Regional Sites:
  The DD_SITE secret should match your Datadog region:
  - US (Default): datadoghq.com
  - EU: datadoghq.eu
  - US3 (GovCloud): ddog-gov.com
  - US5: us5.datadoghq.com
  - AP1: ap1.datadoghq.com
#
Example Usage:
  Search for error logs in the web-app service from the last hour and 
  summarize the most common errors.
#
Connection Type:
  This configuration uses the official remote HTTP MCP server. Authentication is handled via HTTP headers, and the URL pins the generally available `core` toolset to keep the tool surface narrow.
#
Troubleshooting:
  403 Forbidden Errors - Verify that:
  - Your API key and Application key are correct
  - The keys have necessary permissions to access requested resources
  - You're using the correct endpoint for your region
  - Your Datadog account has access to the requested data
#
Usage:
  imports:
    - shared/mcp/datadog.md

-->
