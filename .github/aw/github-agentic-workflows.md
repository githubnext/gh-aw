---
description: GitHub Agentic Workflows
applyTo: ".github/workflows/*.md,.github/workflows/**/*.md"
---

# GitHub Agentic Workflows

## File Format

Agentic workflows are markdown files with YAML frontmatter.

```markdown
---
emoji: 🧠
name: My Workflow
description: Short description
on:
  issues:
    types: [opened]
permissions:
  contents: read
  actions: read
strict: true
network:
  allowed: [defaults, github]
tools:
  github:
    mode: gh-proxy
    toolsets: [default]
safe-outputs:
  add-comment:
---

# Workflow Title

Natural language instructions for the AI agent.
```

## Recompilation Rule

- Edit the **frontmatter** → run `gh aw compile <workflow-id>`.
- Edit the **markdown body** only → no recompilation required.

See also: [workflow-editing.md](workflow-editing.md)

## Core Rules

- Keep the main agent job read-only.
- Use `safe-outputs:` for GitHub writes.
- Prefer `tools.github.mode: gh-proxy` and use `gh` for GitHub reads.
- For non-GitHub MCP servers, prefer `tools.cli-proxy: true` and use mounted `mcp-clis` commands.
- Use `${{ steps.sanitized.outputs.text }}` for untrusted user content.
- Set `strict: true` for production workflows.
- Limit network and bash access to what the workflow actually needs.

See also: [workflow-constraints.md](workflow-constraints.md)

## Reference Files

| Topic | File |
|---|---|
| Editing and recompilation rules | [workflow-editing.md](workflow-editing.md) |
| Architectural and security constraints | [workflow-constraints.md](workflow-constraints.md) |
| Common design patterns | [workflow-patterns.md](workflow-patterns.md) |
| Frontmatter schema index | [syntax.md](syntax.md) |
| Safe outputs index | [safe-outputs.md](safe-outputs.md) |
| Trigger patterns | [triggers.md](triggers.md) |
| Context expressions and `{{#if}}` templates | [context.md](context.md) |
| CLI commands and MCP equivalents | [cli-commands.md](cli-commands.md) |
| Network configuration | [network.md](network.md) |
| Memory and persistence | [memory.md](memory.md) |
| Imports and shared components | [reuse.md](reuse.md) |
| Sub-agents | [subagents.md](subagents.md) |
| Skills | [skills.md](skills.md) |
| Token cost optimization | [token-optimization.md](token-optimization.md) |
| GitHub MCP server configuration | [github-mcp-server.md](github-mcp-server.md) |
| Campaign and KPI patterns | [campaign.md](campaign.md) |
| Experiments and A/B testing | [experiments.md](experiments.md) |
| Charts and Python data visualization | [charts.md](charts.md) |
| LLM API endpoint discovery | [llms.md](llms.md) |

## Compile Commands

```bash
gh aw compile
gh aw compile <workflow-id>
gh aw compile --purge
gh aw compile --strict
```
