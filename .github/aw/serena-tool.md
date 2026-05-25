# Serena Language Server Tool

Serena is a **language service protocol (LSP) MCP server** for semantic code analysis. Use it ONLY when you need deep code understanding beyond text manipulation.

## Quick Decision: Should I Use Serena?

**✅ YES - Use Serena when you need:**
- Symbol navigation (find all usages of a function/type)
- Call graph analysis across files
- Semantic duplicate detection (not just text matching)
- Refactoring analysis (functions in wrong files, extraction opportunities)
- Type relationships and interface implementations

**❌ NO - Use simpler tools when:**
- Searching text patterns → Use `grep`
- Editing files → Use `edit` tool
- Running commands → Use `bash`
- Working with YAML/JSON/Markdown → Use `edit` tool
- Simple file operations → Use `bash` or `create`

**Rule of thumb**: If `grep` or `bash` can solve it in 1-2 commands, don't use Serena.

## Configuration

Import the shared Serena workflow via the `imports:` field — `tools.serena` has been removed. For multi-language, first entry is the default fallback:

```yaml
imports:
  - uses: shared/mcp/serena.md
    with:
      languages: ["go", "typescript"]  # go, typescript, python, ruby, rust, java, cpp, csharp
```

## Available Serena Tools

### Navigation & Analysis
- `find_symbol` - Search for symbols by name
- `get_symbols_overview` - List all symbols in a file
- `find_referencing_symbols` - Find where a symbol is used
- `find_referencing_code_snippets` - Find code snippets using a symbol
- `search_for_pattern` - Search for code patterns (regex)

### Code Editing
- `read_file` - Read file with semantic context
- `create_text_file` - Create/overwrite files
- `insert_at_line` - Insert content at line number
- `insert_before_symbol` / `insert_after_symbol` - Insert near symbols
- `replace_lines` - Replace line range
- `replace_symbol_body` - Replace symbol definition
- `delete_lines` - Delete line range

### Project Management
- `activate_project` - **REQUIRED** - Activate Serena for workspace
- `onboarding` - Analyze project structure
- `restart_language_server` - Restart LSP if needed
- `get_current_config` - View Serena configuration
- `list_dir` - List directory contents

## Usage Workflow

### 1. Activate Serena First

Always call `activate_project` before other Serena tools, passing the workspace path.

### 2. Combine with Other Tools

Use `bash` for file discovery, Serena for semantic analysis, `edit` for changes.

```yaml
imports:
  - uses: shared/mcp/serena.md
    with:
      languages: ["go"]
tools:
  bash:
    - "find pkg -name '*.go' ! -name '*_test.go'"
    - "cat go.mod"
  github:
    toolsets: [default]
```

### 3. Use Cache for Recurring Analysis

```yaml
imports:
  - uses: shared/mcp/serena.md
    with:
      languages: ["go"]
cache-memory: true  # Store analysis history
```

## Common Pitfalls

❌ **Using Serena for non-code files** - Use `edit` for YAML/JSON/Markdown
❌ **Forgetting activate_project** - Always call first
❌ **Not combining with bash** - Use bash for file discovery
❌ **Missing language configuration** - Must specify language(s)

## Supported Languages

Full LSP: `go` (gopls), `typescript`, `python` (jedi/pyright), `ruby` (solargraph), `rust` (rust-analyzer), `java`, `cpp`, `csharp`. See `.serena/project.yml` for full list (25+).
