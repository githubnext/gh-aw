---
title: Using Serena
description: Configure the Serena MCP server for semantic code analysis and intelligent code editing in your agentic workflows.
sidebar:
  order: 5
---

This guide covers using [Serena](https://github.com/oraios/serena), a powerful coding agent toolkit that provides semantic code retrieval and editing capabilities to agentic workflows.

## What is Serena?

Serena is an MCP server that enhances AI agents with IDE-like tools for understanding and manipulating code. Instead of reading entire files or performing text-based searches, agents can use Serena to:

- **Find symbols** - Locate functions, classes, types, and variables by name
- **Navigate relationships** - Discover references, implementations, and dependencies
- **Edit at symbol level** - Insert, replace, or modify code entities precisely
- **Analyze semantically** - Understand code structure using language servers

Serena supports **30+ programming languages** through Language Server Protocol (LSP) integration, including Go, Python, TypeScript, JavaScript, Rust, Java, C/C++, and many more.

> [!TIP]
> Serena excels at navigating and manipulating complex codebases. It provides the greatest value when working with large, well-structured projects where precise code navigation and editing are essential.

## Quick Start

### Basic Configuration

Add Serena to your workflow using the short syntax with a list of languages:

```yaml wrap
---
engine: copilot
permissions:
  contents: read
tools:
  serena: ["go", "typescript", "python"]
---
```

This enables Serena for Go, TypeScript, and Python code analysis.

### Common Use Cases

**Code analysis workflow:**

```yaml wrap
---
engine: copilot
permissions:
  contents: read
tools:
  serena: ["go"]
  github:
    toolsets: [default]
---

# Code Quality Analyzer

Analyze Go code in this repository and provide recommendations:

1. Use Serena to find all exported functions
2. Check for missing documentation
3. Identify code patterns and suggest improvements
```

**Code refactoring workflow:**

```yaml wrap
---
engine: claude
permissions:
  contents: write
tools:
  serena: ["typescript", "javascript"]
  edit:
---

# Refactor TypeScript Code

Refactor the codebase to use modern TypeScript features:

1. Find all class declarations
2. Convert to functional components where appropriate
3. Update type definitions
```

## Configuration Options

### Short Syntax (Array)

The simplest way to configure Serena - just list the languages:

```yaml wrap
tools:
  serena: ["go", "typescript"]
```

This uses default settings for each language server.

### Long Syntax (Detailed Configuration)

For fine-grained control over language servers:

```yaml wrap
tools:
  serena:
    version: latest
    args: ["--verbose"]
    languages:
      go:
        version: "1.21"
        go-mod-file: "go.mod"
        gopls-version: "v0.14.2"
      typescript:
      python:
        version: "3.12"
```

**Configuration fields:**

- `version` - Serena version to use (default: `latest`)
- `args` - Additional command-line arguments (e.g., `["--verbose"]`)
- `languages` - Language-specific configuration

**Language-specific options:**

For **Go**:
- `version` - Go runtime version
- `go-mod-file` - Path to `go.mod` (default: `"go.mod"`)
- `gopls-version` - gopls language server version (default: `latest`)

For **Python**:
- `version` - Python runtime version

For **TypeScript/JavaScript**:
- No additional configuration required (uses default language server)

### Custom Go Module Path

For projects with `go.mod` in a subdirectory:

```yaml wrap
tools:
  serena:
    languages:
      go:
        go-mod-file: "backend/go.mod"
        gopls-version: "latest"
```

## Language Support

Serena supports **30+ programming languages** through Language Server Protocol (LSP):

| Category | Languages |
|----------|-----------|
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

When Serena is enabled, the agent has access to these semantic code tools:

### Symbol Navigation

- `find_symbol` - Search for functions, classes, types, and variables by name
- `find_referencing_symbols` - Find all references to a symbol
- `get_symbol_definition` - Get the full definition of a symbol
- `list_symbols_in_file` - List all symbols defined in a file

### Code Editing

- `replace_symbol_body` - Replace the implementation of a function or method
- `insert_after_symbol` - Add code after a specific symbol
- `insert_before_symbol` - Add code before a specific symbol
- `delete_symbol` - Remove a symbol definition

### Project Analysis

- `find_files` - Locate files matching patterns
- `get_project_structure` - Analyze directory structure
- `analyze_imports` - Examine import dependencies

> [!TIP]
> These tools enable agents to work at the **symbol level** rather than the file level, making code operations more precise and context-aware.

## Memory Configuration

Serena maintains analysis state in memory for faster operations. Configure memory location:

```yaml wrap
---
tools:
  serena: ["go"]
  cache-memory:
    key: serena-analysis
---

# In your workflow instructions:
# Memory location: /tmp/gh-aw/cache-memory/serena/
```

The agent should create this directory before using Serena:

```bash
mkdir -p /tmp/gh-aw/cache-memory/serena
```

This caches language server indexes and analysis results for improved performance.

## Practical Examples

### Example 1: Finding Unused Functions

```yaml wrap
---
engine: copilot
tools:
  serena: ["go"]
  github:
    toolsets: [default]
---

# Find Unused Code

Analyze the Go codebase for unused exported functions:

1. Configure Serena memory: `mkdir -p /tmp/gh-aw/cache-memory/serena`
2. Use `find_symbol` to list all exported functions
3. Use `find_referencing_symbols` to check usage
4. Report functions with no references
```

### Example 2: Automated Refactoring

```yaml wrap
---
engine: claude
permissions:
  contents: write
tools:
  serena: ["python"]
  edit:
---

# Modernize Python Code

Update Python code to use type hints:

1. Find all function definitions without type hints
2. Analyze function signatures and return types
3. Add appropriate type annotations using `replace_symbol_body`
4. Verify changes maintain correctness
```

### Example 3: Code Quality Analysis

```yaml wrap
---
engine: copilot
tools:
  serena: ["go"]
---

# Analyze Test Coverage

Review Go test files for completeness:

1. List all exported functions in source files
2. Check corresponding test files
3. Identify functions without test coverage
4. Generate report with missing tests
```

## Best Practices

### 1. Configure Memory Early

Always set up Serena's cache directory at the start of your workflow:

```bash
mkdir -p /tmp/gh-aw/cache-memory/serena
```

This enables faster analysis on subsequent operations.

### 2. Use Symbol-Level Operations

Prefer Serena's symbol tools over file-level edits when possible:

```
✅ Good: Use replace_symbol_body to update a function
❌ Avoid: Read entire file, modify text, write back
```

### 3. Specify Language Versions

For Go projects, explicitly configure `go-mod-file` location:

```yaml wrap
tools:
  serena:
    languages:
      go:
        go-mod-file: "go.mod"
        gopls-version: "latest"
```

### 4. Combine with Other Tools

Serena works well alongside other tools:

```yaml wrap
tools:
  serena: ["go"]        # Semantic analysis
  github:               # Repository access
    toolsets: [default]
  edit:                 # File operations
  bash:                 # Build and test
```

### 5. Start Small

For large codebases, begin with targeted analysis:

```
1. Focus on specific packages or modules
2. Use symbol search with filters
3. Gradually expand scope based on findings
```

## Common Workflows

### Code Analysis Workflow

```yaml wrap
---
name: Daily Code Quality Check
on:
  schedule:
    - cron: daily
permissions:
  contents: read
tools:
  serena: ["go"]
  github:
    toolsets: [default]
---

# Mission: Analyze code quality daily

1. Configure Serena cache: `/tmp/gh-aw/cache-memory/serena/`
2. Find all exported functions
3. Check for missing documentation
4. Identify complex functions (high cyclomatic complexity)
5. Create issue with findings
```

### Refactoring Workflow

```yaml wrap
---
name: Automated Refactoring
on: workflow_dispatch
permissions:
  contents: write
tools:
  serena: ["typescript"]
  edit:
safe-outputs:
  create-pull-request:
    title-prefix: "[refactor] "
---

# Mission: Refactor TypeScript code

1. Find deprecated API usage
2. Use Serena to locate all references
3. Replace with modern equivalents
4. Verify changes compile
5. Create pull request with changes
```

### Documentation Workflow

```yaml wrap
---
name: Documentation Check
on: issues
permissions:
  contents: read
  issues: write
tools:
  serena: ["go"]
  github:
    toolsets: [default]
---

# Mission: Verify function documentation

1. Parse issue for target file
2. Use Serena to list all exported symbols
3. Check for missing GoDoc comments
4. Report findings in issue comment
```

## Troubleshooting

### Language Server Not Found

**Problem:** Serena cannot find the language server for your language.

**Solution:** Check that dependencies are installed. For Go:

```bash
go install golang.org/x/tools/gopls@latest
```

### Memory Permission Issues

**Problem:** Serena cannot write to cache directory.

**Solution:** Ensure cache directory exists with proper permissions:

```bash
mkdir -p /tmp/gh-aw/cache-memory/serena
chmod 755 /tmp/gh-aw/cache-memory/serena
```

### Go Module Path Not Found

**Problem:** gopls cannot find `go.mod` file.

**Solution:** Explicitly configure the path:

```yaml wrap
tools:
  serena:
    languages:
      go:
        go-mod-file: "path/to/go.mod"
```

### Slow Initial Analysis

**Problem:** First run takes a long time to analyze code.

**Solution:** This is expected as language servers build indexes. Subsequent runs use cached data and are much faster. Consider:

- Enabling cache-memory for persistence
- Running analysis workflows on schedule (daily) to maintain warm cache
- Limiting scope to specific packages for large codebases

## Advanced Configuration

### Multiple Language Support

Enable Serena for multiple languages in one workflow:

```yaml wrap
tools:
  serena:
    languages:
      go:
        version: "1.21"
        go-mod-file: "backend/go.mod"
      typescript:
      python:
        version: "3.12"
```

### Custom Language Server Settings

For advanced users, pass custom arguments to Serena:

```yaml wrap
tools:
  serena:
    version: latest
    args: ["--verbose", "--log-level=debug"]
    languages:
      go:
        gopls-version: "v0.14.2"
```

### Integrated with Repository Memory

Combine Serena with repository memory for persistent state:

```yaml wrap
tools:
  serena: ["go"]
  repo-memory:
    branch-name: memory/serena-analysis
    description: "Serena analysis cache"
    file-glob: ["memory/serena/*.json"]
```

## Related Documentation

- [Using MCPs](/gh-aw/guides/mcps/) - General MCP server configuration
- [Tools Reference](/gh-aw/reference/tools/) - Complete tools configuration
- [Getting Started with MCPs](/gh-aw/guides/getting-started-mcp/) - MCP introduction
- [Safe Outputs](/gh-aw/reference/safe-outputs/) - Automated pull requests and issues
- [Frontmatter Reference](/gh-aw/reference/frontmatter/) - All configuration options

## External Resources

- [Serena GitHub Repository](https://github.com/oraios/serena) - Official repository
- [Serena Documentation](https://oraios.github.io/serena/) - Comprehensive user guide
- [Language Support](https://oraios.github.io/serena/01-about/020_programming-languages.html) - Supported languages and dependencies
- [Serena Tools Reference](https://oraios.github.io/serena/01-about/035_tools.html) - Complete tool documentation
