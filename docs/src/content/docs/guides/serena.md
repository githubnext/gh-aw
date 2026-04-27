---
title: Using Serena
description: Configure the Serena MCP server for semantic code analysis and intelligent code editing in your agentic workflows.
sidebar:
  order: 5
---

[Serena](https://github.com/oraios/serena) is an MCP server that enhances AI agents with IDE-like tools for semantic code analysis and manipulation. It supports **30+ programming languages** through Language Server Protocol (LSP) integration, enabling agents to find symbols, navigate code relationships, and edit at the symbol level — ideal for navigating and editing large, well-structured codebases.

> [!CAUTION]
> `tools.serena` has been removed. Use the `shared/mcp/serena.md` shared workflow instead (see [Migration](#migration-from-toolsserena) below). Workflows that still use `tools.serena` will fail to compile.

## Quick Start

### Recommended: Import shared workflow

The preferred way to add Serena is to copy the file [`shared/mcp/serena.md`](https://github.com/github/gh-aw/blob/main/.github/workflows/shared/mcp/serena.md) into your repo and import it into your workflow, which configures the complete MCP server automatically:

```aw wrap
---
on: issues
engine: copilot
permissions:
  contents: read
# NOTE: first copy `shared/mcp/serena.md` into your repository before importing it
imports:
  - uses: shared/mcp/serena.md
    with:
      languages: ["go", "typescript"]
---
```

For Go-only workflows, use the convenience wrapper (copy [`shared/mcp/serena-go.md`](https://github.com/github/gh-aw/blob/main/.github/workflows/shared/mcp/serena-go.md) into your repository before importing it):

```aw wrap
---
on: issues
engine: copilot
permissions:
  contents: read
# NOTE: first copy `shared/mcp/serena-go.md` into your repository before importing it
imports:
  - shared/mcp/serena-go.md
---
```

### Example: Code Analysis

```aw wrap
---
engine: copilot
permissions:
  contents: read
imports:
  - uses: shared/mcp/serena.md
    with:
      languages: ["go"]
tools:
  github:
    toolsets: [default]
---

# Code Quality Analyzer

Analyze Go code for quality improvements:
1. Find all exported functions and check for missing documentation
2. Identify code patterns and suggest improvements
```

## Migration from `tools.serena`

Replace `tools.serena` with the equivalent import:

```yaml title="Before (removed)"
tools:
  serena: ["go", "typescript"]
```

```aw wrap title="After (recommended)"
imports:
  - uses: shared/mcp/serena.md
    with:
      languages: ["go", "typescript"]
```

The shared workflow configures the full Serena MCP server (container image, entrypoint, workspace mount) explicitly.

## Language Support

Serena supports **30+ programming languages** through Language Server Protocol (LSP):

| Category | Languages |
| ---------- | ----------- |
| **Systems** | C, C++, Rust, Go, Zig |
| **JVM** | Java, Kotlin, Scala, Groovy (partial) |
| **Web** | JavaScript, TypeScript, Dart, Elm |
| **Dynamic** | Python, Ruby, PHP, Perl, Lua |
| **Functional** | Haskell, Elixir, Erlang, Clojure, OCaml |
| **Scientific** | R, Julia, MATLAB, Fortran |
| **Shell** | Bash, PowerShell |
| **Other** | C#, Swift, Nix, Markdown, YAML, TOML |

> [!NOTE]
> Some language servers require additional dependencies. Most are automatically installed by Serena, but check the [Language Support](https://oraios.github.io/serena/01-about/020_programming-languages.html) documentation for specific requirements.

## Available Tools

Serena provides semantic code tools organized into three categories:

| Category | Tools |
| ---------- | ------- |
| **Symbol Navigation** | `find_symbol`, `find_referencing_symbols`, `get_symbol_definition`, `list_symbols_in_file` |
| **Code Editing** | `replace_symbol_body`, `insert_after_symbol`, `insert_before_symbol`, `delete_symbol` |
| **Project Analysis** | `find_files`, `get_project_structure`, `analyze_imports` |

These tools enable agents to work at the **symbol level** rather than the file level, making code operations more precise and context-aware.

## Usage Examples

### Find Unused Functions

```aw wrap
---
engine: copilot
imports:
  - shared/mcp/serena-go.md
tools:
  github:
    toolsets: [default]
---

# Find Unused Code

1. Configure memory: `mkdir -p /tmp/gh-aw/cache-memory/serena`
2. Use `find_symbol` and `find_referencing_symbols` to identify unused exports
3. Report findings
```

## Best Practices

Pre-create the cache directory (`mkdir -p /tmp/gh-aw/cache-memory/serena`) for faster operations — Serena reuses language server indexes across runs. Pin the key with `tools.cache-memory.key: serena-analysis` in frontmatter to persist it. Prefer symbol-level operations (`replace_symbol_body`) over file-level edits. Combine Serena with other tools like `github`, `edit`, and `bash` for complete workflows. For large codebases, start with targeted analysis of specific packages before expanding scope.

## Troubleshooting

**Language server not found:** Install required dependencies (e.g., `go install golang.org/x/tools/gopls@latest` for Go).

**Memory permission issues:** Ensure cache directory exists with proper permissions: `mkdir -p /tmp/gh-aw/cache-memory/serena && chmod 755 /tmp/gh-aw/cache-memory/serena`

**Slow initial analysis:** Expected behavior as language servers build indexes. Subsequent runs use cached data.

## Related Documentation

- [Imports Reference](/gh-aw/reference/imports/) - Full imports and `import-schema` syntax
- [Using MCPs](/gh-aw/guides/mcps/) - General MCP server configuration
- [Tools Reference](/gh-aw/reference/tools/) - Complete tools configuration
- [Getting Started with MCPs](/gh-aw/guides/getting-started-mcp/) - MCP introduction
- [Serena GitHub Repository](https://github.com/oraios/serena) — official repo and [documentation](https://oraios.github.io/serena/)
- [Language Support](https://oraios.github.io/serena/01-about/020_programming-languages.html) - Supported languages and dependencies
- [Serena Tools Reference](https://oraios.github.io/serena/01-about/035_tools.html) - Complete tool documentation
