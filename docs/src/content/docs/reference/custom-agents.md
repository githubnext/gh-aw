---
title: Custom Agent Files
description: Create specialized AI agents with custom instructions and behavior for GitHub Agentic Workflows
sidebar:
  order: 650
---

Custom agent files provide specialized instructions and behavior for AI engines. Agent files are markdown documents stored in the `.github/agents/` directory and imported via the `imports` field. Copilot supports agents natively, while other engines (Claude, Codex) inject the markdown body as a prompt.

## Creating a Custom Agent

Create a markdown file in `.github/agents/` with agent-specific instructions:

```markdown title=".github/agents/my-agent.md"
---
name: My Custom Agent
description: Specialized agent for code review tasks
---

# Agent Instructions

You are a specialized code review agent. Focus on:
- Code quality and best practices
- Security vulnerabilities
- Performance optimization
```

## Using Custom Agents

Import the agent file in your workflow using the `imports` field:

```yaml
---
on: pull_request
engine: copilot
imports:
  - .github/agents/my-agent.md
---

Review the pull request and provide feedback.
```

The agent instructions are merged with the workflow prompt, customizing the AI engine's behavior for specific tasks.

## Agent File Requirements

- **Location**: Must be in `.github/agents/` directory
- **Format**: Markdown with YAML frontmatter
- **Frontmatter**: Can include `name`, `description`, `tools`, and `mcp-servers`
- **One per workflow**: Only one agent file can be imported per workflow
