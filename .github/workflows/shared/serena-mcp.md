---
# Serena MCP Server
# Container-based MCP server for semantic code analysis and duplicate detection
#
# Serena provides powerful semantic code analysis capabilities for detecting
# code duplication, finding symbols, and analyzing code structure.
#
# Documentation: https://github.com/oraios/serena
#
# Available tools:
#   - activate_project: Initialize semantic analysis environment
#   - find_symbol: Search for code symbols
#   - find_referencing_symbols: Find symbol references
#   - get_symbols_overview: Get file structure overview
#   - read_file: Read file contents
#   - search_for_pattern: Search for code patterns
#   - list_dir: List directory contents
#   - find_file: Find files by name/path
#
# Usage:
#   imports:
#     - shared/serena-mcp.md

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
