# 📋 Workflow Structure

This guide explains how agentic workflows are organized and structured within your repository.

## Directory Structure

Agentic workflows are stored in a unified location:

- **`.github/workflows/`**: Contains both your markdown workflow definitions (source files) and the generated GitHub Actions YAML files (.lock.yml files)
- **`.gitattributes`**: Automatically created/updated to mark `.lock.yml` files as generated code using `linguist-generated=true`

Create markdown files in `.github/workflows/` with the following structure:

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

## File Organization

Your repository structure will look like this:

```
your-repository/
├── .github/
│   └── workflows/
│       ├── issue-responder.md        # Your source workflow
│       ├── issue-responder.lock.yml  # Generated GitHub Actions file
│       ├── weekly-summary.md         # Another source workflow
│       └── weekly-summary.lock.yml   # Generated GitHub Actions file
├── .gitattributes                    # Marks .lock.yml as generated
└── ... (other repository files)
```

## Workflow File Format

Each workflow consists of:

1. **YAML Frontmatter**: Configuration options wrapped in `---`. See [Frontmatter Options](frontmatter.md) for details.
2. **Markdown Content**: Natural language instructions for the AI

### Example Workflow File

```markdown
---
name: Issue Auto-Responder
on:
  issues:
    types: [opened, labeled]

permissions:
  issues: write
  contents: read

engine: claude

tools:
  github:
    allowed: [get_issue, add_issue_comment, list_issue_comments]

cache:
  key: node-modules-${{ hashFiles('package-lock.json') }}
  path: node_modules

max-runs: 50
stop-time: "2025-12-31 23:59:59"
ai-reaction: "eyes"
---

# Issue Auto-Responder

When a new issue is opened, analyze the issue content and:

1. Determine if it's a bug report, feature request, or question
2. Add appropriate labels based on the content
3. Provide a helpful initial response with:
   - Acknowledgment of the issue
   - Request for additional information if needed
   - Links to relevant documentation

The issue details are: "${{ needs.task.outputs.text }}"
```

## Generated Files

When you run `gh aw compile`, the system:

1. **Reads** your `.md` files from `.github/workflows/`
2. **Processes** the frontmatter and markdown content
3. **Generates** corresponding `.lock.yml` GitHub Actions workflow files
4. **Updates** `.gitattributes` to mark generated files

### Lock File Characteristics

- **Automatic Generation**: Never edit `.lock.yml` files manually
- **Complete Workflows**: Contains full GitHub Actions YAML
- **Security**: Includes proper permissions and secret handling
- **MCP Integration**: Sets up Model Context Protocol servers (see [MCP Guide](mcps.md))
- **Artifact Collection**: Automatically saves logs and outputs

## Best Practices

### File Naming

- Use descriptive names: `issue-responder.md`, `pr-reviewer.md`
- Follow kebab-case convention: `weekly-summary.md`
- Avoid spaces and special characters

### Version Control

- **Commit source files**: Always commit `.md` files
- **Commit generated files**: Also commit `.lock.yml` files for transparency
- **Ignore patterns**: Consider `.gitignore` entries if needed:

```gitignore
# Temporary workflow files (if any)
.github/workflows/*.tmp
```

## Integration with GitHub Actions

Generated workflows integrate seamlessly with GitHub Actions:

- **Standard triggers**: All GitHub Actions `on:` events supported
- **Permissions model**: Full GitHub Actions permissions syntax
- **Secrets access**: Automatic handling of required secrets
- **Artifact storage**: Logs and outputs saved as artifacts
- **Concurrency control**: Built-in safeguards against parallel runs

## Related Documentation

- [Commands](commands.md) - CLI commands for workflow management
- [Frontmatter Options](frontmatter.md) - Configuration options for workflows
- [MCPs](mcps.md) - Model Context Protocol configuration
- [Tools Configuration](tools.md) - GitHub and other tools setup
- [Include Directives](include-directives.md) - Modularizing workflows with includes
- [Secrets Management](secrets.md) - Managing secrets and environment variables
