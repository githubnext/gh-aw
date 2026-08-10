---
mcp-servers:
  grafana:
    container: "grafana/mcp-grafana:1.0.0-alpine"
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

The Alpine-based image variant (`-alpine`) is used instead of the default
Debian bookworm-slim variant: the Debian base layer ships a large set of OS
packages (perl-base, libc-bin, util-linux, ...) that carry unpatched CVEs and
GPL/LGPL licenses rejected by the repository license policy.

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
