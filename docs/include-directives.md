# 📝 Include Directives

Include directives allow you to modularize and reuse workflow components across multiple workflows.

## Basic Include Syntax

```markdown
@include relative/path/to/file.md
```

Includes files relative to the current markdown file's location.

## Optional Include Syntax

```markdown
@include? relative/path/to/file.md
```

Includes files optionally - if the file doesn't exist, no error occurs and a friendly informational comment is added to the workflow. The optional file will be watched for changes in `gh aw compile --watch` mode, so creating the file later will automatically include it.

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
---

# Issue Auto-Handler

@include shared/common-tools.md#Tool Usage Guidelines

When an issue is opened, analyze and respond appropriately.

@include shared/github-permissions.md

@include shared/common-tools.md

@include? perf-goals.md

```

The workflow above includes required shared files and optionally includes `perf-goals.md` if it exists. If `perf-goals.md` doesn't exist, the workflow will compile successfully with a friendly note that you can create the file to configure performance goals for the workflow.

## Frontmatter Merging

- **Only `tools:` frontmatter** is allowed in included files, other entries give a warning.
- **Tool merging**: `allowed:` tools are merged across all included files

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

## Include Path Resolution

- **Relative paths**: Resolved relative to the including file
- **Nested includes**: Included files can include other files
- **Circular protection**: System prevents infinite include loops

## Related Documentation

- [Commands](commands.md) - CLI commands for workflow management
- [Workflow Structure](workflow-structure.md) - Directory layout and organization
- [Frontmatter Options](frontmatter.md) - All configuration options
- [Tools Configuration](tools.md) - GitHub and other tools setup
- [MCPs](mcps.md) - Model Context Protocol setup and configuration
- [Secrets Management](secrets.md) - Managing secrets and environment variables
