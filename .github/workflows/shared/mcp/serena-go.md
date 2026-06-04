---
# Serena MCP Server - Go Code Analysis
# Language Server Protocol (LSP)-based tool for deep Go code analysis
#
# Documentation: https://github.com/oraios/serena
#
# Capabilities:
#   - Semantic code analysis using LSP (go to definition, find references, etc.)
#   - Symbol lookup and cross-file navigation
#   - Type inference and structural analysis
#   - Deeper insights than text-based grep approaches
#
# Usage:
#   imports:
#     - shared/mcp/serena-go.md

imports:
  - uses: ./serena.md
    with:
      languages: ["go"]
---

## Serena Go Code Analysis

### Analysis Constraints

1. **Only analyze `.go` files** — Ignore all other file types
2. **Skip test files** — Never analyze files ending in `_test.go`
3. **Focus on `pkg/` directory** — Primary analysis area
4. **Use Serena for semantic analysis** — Leverage LSP capabilities for deeper insights
