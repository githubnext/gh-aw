---
title: Workflow Structure
description: Learn how agentic workflows are organized and structured within your repository, including directory layout and file organization.
sidebar:
  order: 1
---

## Workflow File Format

Each workflow consists of:

1. **YAML Frontmatter**: Configuration options wrapped in `---`. See [Frontmatter Options](./frontmatter/) for details.
2. **Markdown Content**: Natural language instructions for the AI. See [Markdown Content](./markdown/).

The markdown content is where you write natural language instructions for the AI agent. 

Create a markdown file in `.github/workflows/` with the following structure:

```markdown
---
on:
  issues:
    types: [opened]

permissions:
  issues: write

tools:
  github:
    allowed: [add_issue_comment]
---

# Workflow Description

Read the issue #${{ github.event.issue.number }}. Add a comment to the issue listing useful resources and links.
```

This guide explains how agentic workflows are organized and structured within your repository.

## File Organization

Agentic workflows are stored in the `.github/workflows` folder as Markdown files (`*.md`)
and they are compiled to GitHub Actions Workflows files (`*.lock.yml`)

```
.github/
└── workflows/
  ├── weekly-research.md # Agentic Workflow
  └── weekly-research.lock.yml # Compiled GitHub Actions Workflow
```

When you run the `compile` command you generate the lock file.

```sh
gh aw compile
```

## Best Practices

- Use descriptive names: `issue-responder.md`, `pr-reviewer.md`
- Follow kebab-case convention: `weekly-summary.md`
- Avoid spaces and special characters
- **Commit source files**: Always commit `.md` files
- **Commit generated files**: Also commit `.lock.yml` files for transparency

## Related Documentation

- [Frontmatter Options](./frontmatter/) - Configuration options for workflows
- [Markdown Content](./markdown/) - The main markdown content of workflows
- [Include Directives](./include-directives/) - Modularizing workflows with includes
- [CLI Commands](../tools/cli/) - CLI commands for workflow management
- [MCPs](../guides/mcps/) - Model Context Protocol configuration
