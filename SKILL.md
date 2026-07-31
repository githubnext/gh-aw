---
name: agentic-workflows
description: Repo entrypoint skill for gh-aw workflow authoring, routing, and troubleshooting.
---

# gh-aw Prompt Surface

This repository publishes the `agentic-workflows` skill.

For gh-aw workflow design, creation, update, debug, or upgrade tasks, load and
follow `./skills/agentic-workflows/SKILL.md`.

## What this surface does

- Exposes the installable `agentic-workflows` router skill for this repository
- Summarizes the gh-aw repository purpose for first-turn ambient context
- Points task-specific work to the deeper skill files only when needed

## Key concepts

1. **Workflow compilation**: edit workflow markdown, then recompile lock files
2. **Engine selection**: set `engine` in frontmatter to control runtime agent behavior
3. **MCP tools**: configure GitHub/MCP toolsets in frontmatter for repository operations
4. **Safe outputs**: workflow-safe issue/comment output paths and constraints

## Representative usage examples

```bash
# Compile markdown workflows to lock files
gh aw compile

# Run a workflow manually
gh aw run .github/workflows/daily-skill-optimizer.md

# Inspect MCP server usage in workflows
gh aw mcp list
gh aw mcp inspect daily-skill-optimizer

# Audit a workflow run
gh aw audit 24814681146
```

## Where to learn more in this repo

- `/AGENTS.md` for development and agent workflow conventions
- `/skills/*/SKILL.md` for focused domain guidance
- `/.github/aw/*.md` for gh-aw workflow authoring and runtime reference material
