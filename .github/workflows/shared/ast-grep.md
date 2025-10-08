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

The ast-grep MCP server provides tools for:
- Searching code patterns using tree-sitter grammars
- Structural code analysis
- Pattern-based code transformations

### Basic Usage

The MCP server exposes ast-grep functionality through its tools interface. Common operations include:

**Search for patterns:**
- Use the ast-grep MCP tools to search for patterns in code
- Supports multiple programming languages (Go, JavaScript, TypeScript, Python, etc.)
- Pattern matching based on code structure, not text

**Common Go patterns to detect:**

1. **Unmarshal with dash tag** (problematic pattern):
   - Pattern: `json:"-"`
   - See catalog: https://ast-grep.github.io/catalog/go/unmarshal-tag-is-dash.html

2. **Error handling issues:**
   - Pattern: `if err != nil { $$$A }`

3. **Finding specific function calls:**
   - Pattern: `functionName($$$ARGS)`

### More Information

- Documentation: https://ast-grep.github.io/
- Go patterns catalog: https://ast-grep.github.io/catalog/go/
- Pattern syntax guide: https://ast-grep.github.io/guide/pattern-syntax.html
- Docker image: https://hub.docker.com/r/mcp/ast-grep
