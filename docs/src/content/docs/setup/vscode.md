---
title: VS Code Integration
description: Learn about the gh aw CLI tools for improving your developer experience in VS Code and other IDEs, including Copilot instructions.
sidebar:
  order: 300
---

The `gh aw` CLI provides tools to improve your developer experience in VS Code and other IDEs.

## Setup

Run `gh aw init` once in your repository to configure Copilot integration:

```sh wrap
gh aw init
```

This creates files including:
- **Copilot instructions**: `.github/aw/github-agentic-workflows.md` - Automatically imported when authoring markdown files under `.github/workflows/`, making Copilot Chat more efficient at generating Agentic Workflows.
- **Agents**: `.github/agents/agentic-workflows.agent.md` - A unified dispatcher agent you can reference in Copilot Chat to create, debug, update, or upgrade workflows.

## agentic-workflows agent <a id="agentic-workflows"></a>

Use the `/agent` command in Copilot Chat with `agentic-workflows` to work with workflows. The agent intelligently routes your request based on your intent:

```sh wrap
# Create a new workflow
/agent agentic-workflows create a workflow that triages issues

# Debug a workflow
/agent agentic-workflows debug why is my workflow failing?

# Update an existing workflow
/agent agentic-workflows update add web-fetch tool to my-workflow

# Upgrade workflows
/agent agentic-workflows upgrade all workflows to latest version
```

The agent provides guided assistance for trigger selection, permissions, security settings, tool configuration, and best practices. The conversational interface helps you build secure workflows without memorizing syntax.

## Background Compilation

Configure VS Code to automatically compile workflows when files change. Create or update `.vscode/tasks.json` with:

```json wrap
{
  "version": "2.0.0",
  "tasks": [
    {
      "label": "Compile Github Agentic Workflows",
      "dependsOn": ["Compile gh-aw"],
      "type": "shell",
      "command": "gh aw",
      "args": ["compile", "--watch"],
      "isBackground": true,
      "problemMatcher": {
        "owner": "gh-aw",
        "fileLocation": "relative",
        "pattern": {
          "regexp": "^(.*?):(\\d+):(\\d+):\\s(error|warning):\\s(.+)$",
          "file": 1,
          "line": 2,
          "column": 3,
          "severity": 4,
          "message": 5
        },
        "background": {
          "activeOnStart": true,
          "beginsPattern": "Watching for file changes",
          "endsPattern": "Recompiled"
        }
      },
      "group": { "kind": "build", "isDefault": true },
      "runOptions": { "runOn": "folderOpen" }
    }
  ]
}
```

The task starts automatically when you open a Markdown file under `.github/workflows/`. If it doesn't start, use the command palette (`Ctrl + Shift + P`) and run `Tasks: Manage Automatic Tasks` to enable automatic task execution.

## YAML Schema Validation

The repository is configured to provide real-time validation and autocomplete for workflow frontmatter using the [RedHat YAML extension](https://marketplace.visualstudio.com/items?itemName=redhat.vscode-yaml).

### Features

- **Real-time validation**: Frontmatter fields are validated as you type against the comprehensive workflow schema
- **Autocomplete**: Get intelligent suggestions for field names and values
- **Hover documentation**: See descriptions and examples for each field by hovering over it
- **Error highlighting**: Invalid configurations are highlighted with descriptive error messages

### Setup

1. **Install the extension**: When you open the repository, VS Code will recommend installing the RedHat YAML extension. Accept the recommendation or install it manually from the Extensions marketplace.

2. **Automatic configuration**: The schema is automatically applied to all `.md` files in `.github/workflows/` via `.vscode/settings.json`:

```json
{
  "yaml.schemas": {
    "./pkg/parser/schemas/main_workflow_schema.json": ".github/workflows/*.md"
  }
}
```

### Usage

When editing workflow markdown files (`.md` files in `.github/workflows/`), the YAML frontmatter block is validated:

```yaml
---
name: My Workflow  # ✓ Valid string
on: push           # ✓ Valid trigger
permissions:
  contents: read   # ✓ Valid permission level
engine: copilot    # ✓ Valid engine
invalid-field: x   # ✗ Error: Unknown property
---
```

The extension validates:
- Required fields (e.g., `on` trigger is required)
- Field types (strings, numbers, booleans, objects, arrays)
- Enum values (e.g., `engine` must be one of: `copilot`, `claude`, `codex`, `custom`)
- Nested object structures
- Complex conditional logic

### Troubleshooting

If schema validation isn't working:

1. **Check extension is installed**: Look for "YAML" in your installed extensions list
2. **Reload VS Code**: Use `Ctrl+Shift+P` → "Developer: Reload Window"
3. **Verify file pattern**: Schema only applies to `.md` files in `.github/workflows/`
4. **Check YAML frontmatter**: Ensure your frontmatter is wrapped in `---` delimiters
