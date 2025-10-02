---
title: Agentic Authoring
description: Learn how to use the create-agentic-workflow.prompt.md file in VS Code with GitHub Copilot to author agentic workflows in natural language.
---

GitHub Agentic Workflows provide a prompt file that turns your favorite agent into 
a powerful workflow authoring tool. This guide explains how to use this mode to author agentic workflows in natural language.

## What is the create-agentic-workflow prompt?

`.github/prompts/create-agentic-workflow.prompt.md` is a prompt file that contains the structure and instructions the Copilot-style assistant will use to generate a workflow markdown file that the `gh aw` CLI understands.

Use the prompt file when you want to:
- Draft a new agentic workflow using natural language
- Iterate on workflow steps with AI assistance inside your editor

The prompt contains instructions and toolset to enable efficient workflow authoring.

To get this file in your repository, run the compile command:

```bash
gh aw compile
```

## How to use the create-agentic-workflow prompt

### GitHub Copilot Chat in Visual Studio Code

In Visual Studio Code and the GitHub Copilot Chat, you can load it using the `create-agentic-workflow` command.

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

### After compiling

Commit the generated files (`.lock.yml`) if they are part of the project's tracked artifacts. The project uses compiled workflows in version control.

```sh
/create-agentic-workflow
```

This will start the agentic workflow authoring process.

### GitHub Copilot CLI

Assuming you have the GitHub Copilot CLI installed, you can load the file in a session using the `@` syntax:

```bash
load @.github/prompts/create-agentic-workflow.prompt.md
```


## After compiling

Commit the generated files (`.lock.yml`) if they are part of the project's tracked artifacts. The project uses compiled workflows in version control.
