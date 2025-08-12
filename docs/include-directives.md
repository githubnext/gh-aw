# 📝 Include Directives

Include directives allow you to modularize and reuse workflow components across multiple workflows.

## Basic Include Syntax

```markdown
@include relative/path/to/file.md
```

Includes files relative to the current markdown file's location.

## Section-Specific Includes

```markdown
@include filename.md#Section
```

Includes only a specific section from a markdown file using the section header.

## Include Examples

### Directory Structure
```
.github/workflows/
├── shared/
│   ├── common-tools.md
│   └── github-permissions.md
├── issue-handler.md
└── pr-reviewer.md
```

### Shared Tools (`shared/common-tools.md`)
```markdown
---
tools:
  github:
    allowed: [get_issue, add_issue_comment, get_pull_request]
  claude:
    allowed:
      Edit:
      Read:
      Bash: ["git", "grep"]
---

# Common Tools Configuration

This file contains shared tool configurations used across multiple workflows.
```

### Shared Permissions (`shared/github-permissions.md`)
```markdown
---
permissions:
  issues: write
  contents: read
  pull-requests: write
---

# Standard GitHub Permissions

Common permission set for repository automation workflows.
```

### Main Workflow Using Includes
```markdown
---
on:
  issues:
    types: [opened]

@include shared/github-permissions.md
@include shared/common-tools.md

max-runs: 50
ai-reaction: "eyes"
---

# Issue Auto-Handler

@include shared/common-tools.md#Tool Usage Guidelines

When an issue is opened, analyze and respond appropriately.

Issue content: "${{ needs.task.outputs.text }}"
```

## Include Rules and Limitations

### Frontmatter Merging
- **Only `tools:` frontmatter** is allowed in included files
- **Tool merging**: `allowed:` tools are merged across all included files
- **Conflict resolution**: Later includes override earlier ones for the same tool

### Example Tool Merging
```markdown
# Base workflow
---
tools:
  github:
    allowed: [get_issue]
---

@include shared/extra-tools.md  # Adds more GitHub tools
```

```markdown
# shared/extra-tools.md
---
tools:
  github:
    allowed: [add_issue_comment, update_issue]
  claude:
    allowed:
      Edit:
---
```

**Result**: Final workflow has `github.allowed: [get_issue, add_issue_comment, update_issue]` and Claude Edit tool.

### Include Path Resolution
- **Relative paths**: Resolved relative to the including file
- **Nested includes**: Included files can include other files
- **Circular protection**: System prevents infinite include loops

## Advanced Include Patterns

### Conditional Includes by Environment
```markdown
---
on:
  issues:
    types: [opened]
---

@include shared/base-config.md

# Development environment
@include environments/development.md#Development Tools

# Production environment  
@include environments/production.md#Production Tools

# Conditional based on repository
```

### Include with Section Targeting
```markdown
# Include only the permissions section
@include shared/security-config.md#Permissions

# Include only tool configuration
@include shared/tool-config.md#GitHub Tools
@include shared/tool-config.md#MCP Servers

> **📘 Complete MCP Guide**: For comprehensive MCP server configuration, see the [MCPs](mcps.md).
```

## Related Documentation

- [Commands](commands.md) - CLI commands for workflow management
- [Workflow Structure](workflow-structure.md) - Directory layout and organization
- [Frontmatter Options](frontmatter.md) - All configuration options
- [Tools Configuration](tools.md) - GitHub and other tools setup
- [MCPs](mcps.md) - Model Context Protocol setup and configuration
- [Secrets Management](secrets.md) - Managing secrets and environment variables
