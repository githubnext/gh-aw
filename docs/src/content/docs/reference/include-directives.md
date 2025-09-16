---
title: Include Directives
description: Learn how to modularize and reuse workflow components across multiple workflows using include directives for better organization and maintainability.
sidebar:
  order: 4
---

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
  edit:
---
```

**Result**: Final workflow has `github.allowed: [get_issue, add_issue_comment, update_issue]` and Claude Edit tool.

## Include Path Resolution

- **Relative paths**: Resolved relative to the including file
- **Nested includes**: Included files can include other files
- **Circular protection**: System prevents infinite include loops

## Related Documentation

- [CLI Commands](/gh-aw/tools/cli/) - CLI commands for workflow management
- [Workflow Structure](/gh-aw/reference/workflow-structure/) - Directory layout and organization
- [Frontmatter Options](/gh-aw/reference/frontmatter/) - All configuration options
- [Tools Configuration](/gh-aw/reference/tools/) - GitHub and other tools setup
- [MCPs](/gh-aw/guides/mcps/) - Model Context Protocol setup and configuration
