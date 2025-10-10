---
title: VS Code Integration
description: Learn about the gh aw CLI tools for improving your developer experience in VS Code and other IDEs, including Copilot instructions.
---

The `gh aw` cli provides a few tools to improve your developer experience in VS Code (or other IDEs).

## Copilot instructions <a id="copilot-instructions"></a>

The `gh aw init` command writes a [custom Copilot instructions file](https://code.visualstudio.com/docs/copilot/copilot-customization) at `.github/instructions/github-agentic-workflows.instructions.md`.

:::tip[Initialize your repository]
Run `gh aw init` once in your repository to set up Copilot instructions and prompt files:

```sh
gh aw init
```
:::

The instructions will automatically be imported by Copilot when authoring markdown
files under the `.github/workflows` folder.

Once configured, you will notice that Copilot Chat will be much more efficient at
generating Agentic Workflows.

## /create-agentic-workflow command <a id="create-agentic-workflow"></a>

The `gh aw init` command also creates a [prompt template](https://code.visualstudio.com/docs/copilot/copilot-customization#_prompt-templates) at `.github/prompts/create-agentic-workflow.prompt.md` that enables the `/create-agentic-workflow` command in GitHub Copilot Chat.

:::tip[Initialize your repository]
If you haven't already, run `gh aw init` to set up the prompt files:

```sh
gh aw init
```
:::

Once the prompt file is created, you can use `/create-agentic-workflow` in Copilot Chat to interactively design and create agentic workflows with guided assistance for:

- Choosing appropriate triggers (`on:` events)
- Configuring permissions and security settings
- Selecting tools and MCP servers
- Setting up safe outputs and network permissions
- Following best practices for workflow design

The command provides a conversational interface that helps you build secure, well-structured agentic workflows without needing to memorize the full syntax.

## Background Compilation

You can leverage tasks in VS Code to configure a background compilation of Agentic Workflows.

- open or create `.vscode/tasks.json`
- add or merge the following JSON:

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

The background compilation should start as soon as you open a Markdown file under `.github/workflows/`. If it does not start, 

- open the command palette (`Ctrl + Shift + P`)
- type `Tasks: Run Task` to start the task once
- or type `Tasks: Managed Automatic Tasks` and select `Allow Automatic Tasks` to start it automatically.
