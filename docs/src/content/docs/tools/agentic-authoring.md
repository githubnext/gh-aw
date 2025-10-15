---
title: Authoring Agentic Workflows
description: Learn how to use the create-agentic-workflow.prompt.md file in VS Code to author agentic workflows in natural language.
sidebar:
  order: 100
---

GitHub Agentic Workflows provide a prompt file that turns your favorite agent into 
a powerful workflow authoring tool. This guide explains how to use this mode to author agentic workflows in natural language.

## Quick Start

Initialize your repository to set up GitHub Copilot instructions and prompt files for authoring agentic workflows:

```bash wrap
gh aw init
```

This creates:
- `.github/instructions/github-agentic-workflows.instructions.md` - Custom Copilot instructions for better workflow authoring
- `.github/prompts/create-agentic-workflow.prompt.md` - The `/create-agentic-workflow` command for Copilot Chat
- `.github/prompts/create-shared-agentic-workflow.prompt.md` - The `/create-shared-agentic-workflow` command for creating reusable workflow components

:::tip[VS Code integration]
Once initialized, GitHub Copilot will automatically use these instructions when you edit workflow files in VS Code. See [VS Code Integration](/gh-aw/tools/vscode/) for more details.
:::

## What is the `/create-agentic-workflow` prompt?

`.github/prompts/create-agentic-workflow.prompt.md` is a prompt file that contains the structure and instructions the Copilot-style assistant will use to generate a workflow markdown file that the `gh aw` CLI understands.

Use the prompt file when you want to:
- Draft a new agentic workflow using natural language
- Iterate on workflow steps with AI assistance inside your editor

The prompt contains instructions and toolset to enable efficient workflow authoring.

To get this file in your repository, run the init command:

```bash
gh aw init
```

## How to use the `/create-agentic-workflow` prompt

### GitHub Copilot Chat in Visual Studio Code

In Visual Studio Code and the GitHub Copilot Chat, you can load it using the `/create-agentic-workflow` command.

```sh
/create-agentic-workflow
```

This will start the agentic workflow authoring process.

### GitHub Copilot CLI

Assuming you have the GitHub Copilot CLI installed, you can load the file in a session using the `@` syntax:

```bash
load @.github/prompts/create-agentic-workflow.prompt.md
```

## Other Agents and chats

Load the prompt file into your preferred AI chat or agent interface that supports loading from files. The prompt is designed to be compatible with various AI tools, although the tools might not be completely configured and you'll need to allow running the compiler.

## After compiling

Commit the generated files (`.lock.yml`) if they are part of the project's tracked artifacts. The project uses compiled workflows in version control.
