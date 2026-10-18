---
title: Using Serena
description: Configure the Serena MCP server for semantic code analysis and intelligent code editing in your agentic workflows.
sidebar:
  order: 5
---

[Serena](https://github.com/oraios/serena) is an MCP server that enhances AI agents with IDE-like tools for semantic code analysis and manipulation. It supports **30+ programming languages** through Language Server Protocol (LSP) integration, enabling agents to find symbols, navigate code relationships, and edit at the symbol level — ideal for navigating and editing large, well-structured codebases.

## Quick Start

### Recommended: Import shared workflow

The preferred way to add Serena is to copy [`shared/mcp/serena.md`](https://github.com/github/gh-aw/blob/main/.github/workflows/shared/mcp/serena.md) into your repo and import it. For Go analysis, also copy [`shared/mcp/serena-go.md`](https://github.com/github/gh-aw/blob/main/.github/workflows/shared/mcp/serena-go.md) and import that wrapper to provision its language server:

```aw wrap
---
on: issues

engine: copilot

permissions:
  contents: read
# NOTE: first copy both shared Serena files into your repository

imports:
  - uses: shared/mcp/serena-go.md
    with:
      languages: ["go", "typescript"]
---
```

For workflows analyzing Go, copy both shared files and use the [`shared/mcp/serena-go.md`](https://github.com/github/gh-aw/blob/main/.github/workflows/shared/mcp/serena-go.md) wrapper. It sets up Go from `go.mod`, installs pinned `gopls` in the mounted runner tool cache, and checks semantic-symbol readiness before MCP startup. The container sees that cache read-only; no extra host mounts are added.

```aw wrap
---
on: issues

engine: copilot

permissions:
  contents: read
# NOTE: first copy both `shared/mcp/serena.md` and `shared/mcp/serena-go.md`

imports:
  - shared/mcp/serena-go.md
---
```

For mixed-language workflows, pass `languages: ["go", "typescript"]` to the Go wrapper. The default is `["go"]`; existing runtime PATH entries are preserved for other language servers.

### Example: Code Analysis

```aw wrap
---
engine: copilot

permissions:
  contents: read

imports:
  - shared/mcp/serena-go.md

tools:
  github:
    toolsets: [default]
---

# Code Quality Analyzer

Analyze Go code for quality improvements:
1. Find all exported functions and check for missing documentation
2. Identify code patterns and suggest improvements
```

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

**Go language server not found:** Use the Go wrapper above rather than the base Serena import. It provisions `gopls` before agent startup in a directory covered by Serena's mounts. A runner's default `GOPATH/bin` is not necessarily visible inside the container. If setup or a semantic request fails, report the error and stop semantic tasks; do not install tools ad hoc or substitute text-only analysis.

**Memory permission issues:** Ensure cache directory exists with proper permissions: `mkdir -p /tmp/gh-aw/cache-memory/serena && chmod 755 /tmp/gh-aw/cache-memory/serena`

**Slow initial analysis:** Expected behavior as language servers build indexes. Subsequent runs use cached data.

## Learn More

- [Imports Reference](/gh-aw/reference/imports/) - Full imports and `import-schema` syntax
- [Using MCPs](/gh-aw/guides/mcps/) - General MCP server configuration
- [Tools Reference](/gh-aw/reference/tools/) - Complete tools configuration
- [Serena GitHub Repository](https://github.com/oraios/serena) — official repo and [documentation](https://oraios.github.io/serena/)
- [Language Support](https://oraios.github.io/serena/01-about/020_programming-languages.html) - Supported languages and dependencies
- [Serena Tools Reference](https://oraios.github.io/serena/01-about/035_tools.html) - Complete tool documentation
