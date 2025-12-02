---
title: Custom Prompt Files
description: Create specialized AI prompts with custom instructions and behavior for GitHub Agentic Workflows
sidebar:
  order: 650
---

Custom prompt files provide specialized instructions and behavior for AI engines. Prompt files are markdown documents with `.prompt.md` extension stored in the `.github/prompts/` directory. They can be invoked as slash commands in GitHub Copilot Chat.

## Creating a Custom Prompt

Create a markdown file with `.prompt.md` extension in `.github/prompts/` with prompt-specific instructions:

```markdown title=".github/prompts/my-prompt.prompt.md"
---
description: Specialized prompt for code review tasks
name: my-custom-prompt
agent: agent
---

# Prompt Instructions

You are a specialized code review assistant. Focus on:
- Code quality and best practices
- Security vulnerabilities
- Performance optimization
```

## Using Custom Prompts

Invoke custom prompts directly in GitHub Copilot Chat using slash commands:

```
/my-custom-prompt
```

The prompt instructions guide the AI engine's behavior for specific tasks.

## Prompt File Requirements

- **Location**: Must be in `.github/prompts/` directory
- **Extension**: Must use `.prompt.md` extension
- **Format**: Markdown with YAML frontmatter
- **Frontmatter**: Must include `description`, `name`, and `agent` fields
  - `description`: Brief description of what the prompt does
  - `name`: Command name for invoking the prompt (must match filename without extension)
  - `agent`: Mode used for running the prompt (typically `agent`)

## Built-in Custom Prompts

The `gh aw init` command sets up several custom prompts:

- `/create-agentic-workflow` - Interactive workflow creation with guidance on triggers, tools, and security
- `/setup-agentic-workflows` - Setup guide for configuring workflow engines and secrets
- `/debug-agentic-workflow` - Debug and refine workflows using CLI tools (`gh aw logs`, `gh aw audit`, `gh aw compile`)
- `/create-shared-agentic-workflow` - Create reusable shared workflow components

These prompts provide conversational workflow creation, debugging, and performance analysis.
