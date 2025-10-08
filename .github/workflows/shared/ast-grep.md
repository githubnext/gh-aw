---
mcp-servers:
  ast-grep:
    container: "mcp/ast-grep"
    version: "latest"
    allowed: ["*"]
---

## ast-grep MCP Server

ast-grep is a powerful structural search and replace tool for code. It uses tree-sitter grammars to parse and search code based on its structure rather than just text patterns.

### Available Tools

The ast-grep MCP server provides MCP tools for structural code analysis. The specific tools exposed by the server can be discovered using the MCP protocol. This server enables:
- Searching code patterns using tree-sitter grammars
- Structural code analysis
- Pattern-based code transformations

### Basic Usage

The MCP server exposes ast-grep functionality through its MCP tools interface. When using ast-grep in your workflow, you can perform structural searches across multiple programming languages (Go, JavaScript, TypeScript, Python, etc.) with pattern matching based on code structure rather than text.

**Example patterns that can be searched:**

1. **Unmarshal with dash tag** (problematic Go pattern):
   - Pattern: `json:"-"`
   - Reference: https://ast-grep.github.io/catalog/go/unmarshal-tag-is-dash.html

2. **Error handling patterns:**
   - Pattern: `if err != nil { $$$A }`

3. **Function call patterns:**
   - Pattern: `functionName($$$ARGS)`

### More Information

- Documentation: https://ast-grep.github.io/
- Go patterns catalog: https://ast-grep.github.io/catalog/go/
- Pattern syntax guide: https://ast-grep.github.io/guide/pattern-syntax.html
- Docker image: https://hub.docker.com/r/mcp/ast-grep
