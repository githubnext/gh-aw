---
title: Include Directives
description: Learn how to modularize and reuse workflow components across multiple workflows using import directives for better organization and maintainability.
sidebar:
  order: 4
---

Import directives allow you to modularize and reuse workflow components across multiple workflows.

## Basic Import Syntax

```aw wrap
@import relative/path/to/file.md
```

Imports files relative to the current markdown file's location.

:::note
`@import` and `@include` are aliases - you can use either keyword interchangeably.
:::

## Optional Import Syntax

```aw wrap
@import? relative/path/to/file.md
```

Imports files optionally - if the file doesn't exist, no error occurs and a friendly informational comment is added to the workflow. The optional file will be watched for changes in `gh aw compile --watch` mode, so creating the file later will automatically import it.

## Section-Specific Imports

```aw wrap
@import filename.md#Section
```

Imports only a specific section from a markdown file using the section header.

## Frontmatter Merging

- **Only `tools:` frontmatter** is allowed in imported files, other entries give a warning.
- **Tool merging**: `allowed:` tools are merged across all imported files

### Example Tool Merging
```aw wrap
# Base workflow
---
tools:
  github:
    allowed: [get_issue]
---

@import shared/extra-tools.md  # Adds more GitHub tools
```

```aw wrap
# shared/extra-tools.md
---
tools:
  github:
    allowed: [add_issue_comment, update_issue]
  edit:
---
```

**Result**: Final workflow has `github.allowed: [get_issue, add_issue_comment, update_issue]` and Claude Edit tool.

## Import Path Resolution

- **Relative paths**: Resolved relative to the importing file
- **Nested imports**: Imported files can import other files
- **Circular protection**: System prevents infinite import loops

## Related Documentation

- [Packaging and Imports](/gh-aw/guides/packaging-imports/) - Complete guide to adding, updating, and importing workflows
- [CLI Commands](/gh-aw/tools/cli/) - CLI commands for workflow management
- [Workflow Structure](/gh-aw/reference/workflow-structure/) - Directory layout and organization
- [Frontmatter Options](/gh-aw/reference/frontmatter/) - All configuration options
- [Tools Configuration](/gh-aw/reference/tools/) - GitHub and other tools setup
- [MCPs](/gh-aw/guides/mcps/) - Model Context Protocol setup and configuration
