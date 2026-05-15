---
mcp-servers:
  grafana:
    container: "grafana/mcp-grafana"
    entrypointArgs:
      - "-t"
      - "stdio"
      - "--disable-write"
    env:
      GRAFANA_URL: "${{ secrets.GRAFANA_URL }}"
      GRAFANA_SERVICE_ACCOUNT_TOKEN: "${{ secrets.GRAFANA_SERVICE_ACCOUNT_TOKEN }}"
---

<!--

https://github.com/grafana/mcp-grafana

Required secrets:
- GRAFANA_URL
- GRAFANA_SERVICE_ACCOUNT_TOKEN

This shared component runs the Grafana MCP server in stdio mode with write
operations disabled.

Usage:
  imports:
    - shared/mcp/grafana.md
-->
