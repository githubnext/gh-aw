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

Serena is enabled for **${{ github.aw.import-inputs.languages }}** in `${{ github.workspace }}`. Start by calling `activate_project` with that workspace path, then prefer Serena semantic tools for symbol lookup, references, docs, diagnostics, and structured edits.
