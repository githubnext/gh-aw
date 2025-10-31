---
title: Custom Agent Files
description: Create specialized AI agents with custom instructions and behavior for GitHub Agentic Workflows
sidebar:
  order: 650
---

All AI engines (Copilot, Claude, Codex) support custom agent files that provide specialized instructions and behavior. Agent files are markdown documents stored in the `.github/agents/` directory and imported via the `imports` field.

## Creating a Custom Agent

Create a markdown file in `.github/agents/` with agent-specific instructions:

```markdown title=".github/agents/my-agent.md"
---
name: My Custom Agent
description: Specialized agent for code review tasks
tools:
  edit:
  bash: ["git", "gh"]
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
- **Import syntax**: Use `imports` field (not `engine.custom-agent`)

## Migration from engine.custom-agent

The `engine.custom-agent` field has been removed. Migrate to the `imports` approach:

**Before (deprecated):**
```yaml
engine:
  id: copilot
  custom-agent: .github/agents/my-agent.md
```

**After (current):**
```yaml
engine: copilot
imports:
  - .github/agents/my-agent.md
```

No backward compatibility - update workflows to use the new `imports` syntax.
