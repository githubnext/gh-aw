---
mcp-servers:
  grafana:
    container: "grafana/mcp-grafana"
    entrypointArgs:
      - "-t"
      - "stdio"
      - "--disable-write"
    allowed:
      - list_datasources
      - get_datasource
      - tempo_traceql-search
      - tempo_get-trace
      - tempo_get-attribute-names
      - tempo_get-attribute-values
      - tempo_docs-traceql
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

Allowed tools:
- list_datasources
- get_datasource
- tempo_traceql-search
- tempo_get-trace
- tempo_get-attribute-names
- tempo_get-attribute-values
- tempo_docs-traceql

Usage:
  imports:
    - shared/mcp/grafana.md
-->
