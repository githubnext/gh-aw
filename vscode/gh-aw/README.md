# GitHub Agentic Workflows VSCode Extension

A minimalistic VSCode extension that provides syntax highlighting and schema validation for GitHub Agentic Workflow files.

## Features

- **Syntax Highlighting**: Rich syntax highlighting for agentic workflow files (`.md` files in `.github/workflows/`)
- **Schema Validation**: YAML frontmatter validation with IntelliSense
- **Auto-completion**: Smart completions for workflow properties and values
- **Hover Information**: Helpful tooltips explaining workflow properties
- **Language Support**: Proper language association for agentic workflow files

## Supported File Patterns

The extension automatically detects and provides features for:
- `**/.github/workflows/*.md` - Agentic workflow files

## YAML Frontmatter Schema

The extension provides validation and IntelliSense for the following frontmatter properties:

### Core Properties
- `on` - Workflow triggers (push, pull_request, schedule, etc.)
- `engine` - AI engine configuration (claude, codex, gpt-4)
- `permissions` - GitHub token permissions (read-all, write-all, or granular)
- `safe-outputs` - Configuration for allowed GitHub API actions
- `tools` - Available tools for the AI agent (web-fetch, web-search, playwright)

### Additional Properties
- `network` - Network access configuration
- `cache` - Caching configuration for persistent storage
- `timeout_minutes` - Maximum workflow runtime
- `if` - Conditional execution expressions

## Syntax Highlighting Features

- **YAML Frontmatter**: Syntax highlighting for configuration properties
- **Markdown Content**: Standard markdown highlighting with agentic-specific enhancements
- **GitHub Expressions**: Highlighting for `\${{ }}` expressions
- **Include Directives**: Special highlighting for `@include` statements
- **Code Blocks**: Language-specific highlighting within fenced code blocks

## Installation

1. Open VSCode
2. Go to Extensions (Ctrl+Shift+X)
3. Search for "GitHub Agentic Workflows"
4. Click Install

## Usage

1. Open any `.md` file in a `.github/workflows/` directory
2. The extension will automatically activate and provide syntax highlighting
3. Use Ctrl+Space for auto-completion in the YAML frontmatter
4. Hover over properties for documentation

## Example Workflow File

\`\`\`markdown
---
on:
  schedule:
    - cron: "0 9 * * 1"
engine: claude
permissions: read-all
safe-outputs:
  create-issue:
    title-prefix: "[Weekly] "
    labels: [automation]
tools:
  web-search:
---

# Weekly Research Report

Generate a weekly research report on the latest developments in AI and machine learning.
\`\`\`

## Contributing

This extension is part of the [GitHub Agentic Workflows](https://github.com/githubnext/gh-aw) project. Contributions are welcome!

## License

MIT License - see the [LICENSE](../../LICENSE) file for details.