---
# Serena MCP Server - Multi-Language Code Analysis
# Language Server Protocol (LSP)-based tool for deep semantic code analysis.
# Supports 30+ languages through per-language LSP integration.
#
# Documentation: https://github.com/oraios/serena
#
# Usage:
#   imports:
#     - uses: shared/mcp/serena.md
#       with:
#         languages: ["go"]                          # one language
#         languages: ["go", "typescript"]            # multiple languages
#         languages: ["typescript", "python"]        # with node/python runtimes
#
# The 'languages' input configures the Serena MCP server language list.

import-schema:
  languages:
    type: array
    items:
      type: string
    required: true
    description: >
      List of programming language identifiers to enable for Serena LSP analysis.
      Supported values include: go, typescript, javascript, python, rust, java,
      ruby, csharp, cpp, c, kotlin, scala, swift, php, and more.

mcp-servers:
  serena:
    container: "ghcr.io/github/serena-mcp-server:latest"
    args:
      - "--network"
      - "host"
    entrypoint: "serena"
    entrypointArgs:
      - "start-mcp-server"
      - "--context"
      - "codex"
      - "--project"
      - \${GITHUB_WORKSPACE}
    mounts:
      - \${GITHUB_WORKSPACE}:\${GITHUB_WORKSPACE}:rw
---

## Serena Code Analysis

The Serena MCP server is configured for **${{ github.aw.import-inputs.languages }}** analysis in this workspace:
- **Workspace**: `${{ github.workspace }}`
- **Memory**: `/tmp/gh-aw/cache-memory/serena/`

### Project Activation

Before analyzing code, activate the Serena project:
```
Tool: activate_project
Args: { "path": "${{ github.workspace }}" }
```

### Available Capabilities

Serena provides IDE-grade Language Server Protocol (LSP) tools including:
- **Symbol search**: `find_symbol` — locate functions, types, interfaces by name
- **Navigation**: `find_referencing_symbols` — find all callers/usages of a symbol
- **Type info**: `get_symbol_documentation` — hover-level type and doc information
- **Code editing**: `replace_symbol_body`, `insert_after_symbol` — symbol-level edits
- **Diagnostics**: `get_diagnostics` — compiler errors and linter warnings

### Analysis Guidelines

1. **Use semantic tools over text search** — prefer Serena's LSP tools over `grep`
2. **Activate project first** — always call `activate_project` before other tools
3. **Cross-reference findings** — validate with multiple tools for accuracy
4. **Focus on the relevant language files** — ignore unrelated file types
