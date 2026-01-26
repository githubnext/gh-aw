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
