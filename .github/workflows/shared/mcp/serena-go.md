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

tools:
  serena: ["go"]
---

## Serena Go Code Analysis

The Serena MCP server is configured for Go code analysis in this workspace:
- **Workspace**: `${{ github.workspace }}`
- **Memory**: `/tmp/gh-aw/cache-memory/serena/`

### Project Activation

Before analyzing code, activate the Serena project:
```
Tool: activate_project
Args: { "path": "${{ github.workspace }}" }
```

### Analysis Constraints

1. **Only analyze `.go` files** — Ignore all other file types
2. **Skip test files** — Never analyze files ending in `_test.go`
3. **Focus on `pkg/` directory** — Primary analysis area
4. **Use Serena for semantic analysis** — Leverage LSP capabilities for deeper insights
