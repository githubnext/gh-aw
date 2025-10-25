---
title: VS Code Integration
description: Learn about the gh aw CLI tools for improving your developer experience in VS Code and other IDEs, including Copilot instructions.
sidebar:
  order: 300
---

The `gh aw` CLI provides tools to improve your developer experience in VS Code and other IDEs.

## Setup

Run `gh aw init` once in your repository to configure Copilot integration:

```sh
gh aw init
```

This creates two files:
- **Copilot instructions**: `.github/instructions/github-agentic-workflows.instructions.md` - Automatically imported when authoring markdown files under `.github/workflows/`, making Copilot Chat more efficient at generating Agentic Workflows.
- **Prompt template**: `.github/prompts/create-agentic-workflow.prompt.md` - Enables the `/create-agentic-workflow` command in Copilot Chat.

## /create-agentic-workflow command <a id="create-agentic-workflow"></a>

Use `/create-agentic-workflow` in Copilot Chat to interactively design workflows with guided assistance for trigger selection, permissions, security settings, tool configuration, and best practices. The conversational interface helps you build secure workflows without memorizing syntax.

## Background Compilation

Configure VS Code to automatically compile workflows when files change. Create or update `.vscode/tasks.json` with:

```json
{
  "version": "2.0.0",
  "tasks": [
    {
      "label": "Compile Github Agentic Workflows",
      "dependsOn": ["Compile gh-aw"],
      "type": "shell",
      "command": "./gh-aw",
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
