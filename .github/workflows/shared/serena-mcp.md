---
mcp-servers:
  serena:
    container: "ghcr.io/oraios/serena"
    version: "latest"
    args:
      - "-v"
      - "${{ github.workspace }}:/workspace:ro"
      - "-w"
      - "/workspace"
    env:
      SERENA_DOCKER: "1"
      SERENA_PORT: "9121"
      SERENA_DASHBOARD_PORT: "24282"
    network:
      allowed:
        - "github.com"
    allowed:
      - activate_project
      - find_symbol
      - find_referencing_symbols
      - get_symbols_overview
      - read_file
      - search_for_pattern
      - list_dir
      - find_file
---

### Serana Tools
Serana is enabled through MCP tools. **DO NOT ATTEMPT TO DOWNLOAD OR LAUNCH DOCKER CONTAINERS MANUALLY.**