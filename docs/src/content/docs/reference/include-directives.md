---
title: Imports
description: Learn how to modularize and reuse workflow components across multiple workflows using the imports field in frontmatter for better organization and maintainability.
sidebar:
  order: 4
---

The `imports:` field in frontmatter allows you to modularize and reuse workflow components across multiple workflows.

## Frontmatter Imports

The recommended way to import shared components is using the `imports:` field in the frontmatter:

```aw wrap
---
on: issues
engine: copilot
imports:
  - shared/common-tools.md
  - shared/mcp/tavily.md
---

# Your Workflow

Workflow instructions here...
```

### Import Path Resolution

- **Relative paths**: Resolved relative to the importing file
- **Nested imports**: Imported files can import other files
- **Circular protection**: System prevents infinite import loops

## Frontmatter Merging

- **Only `tools:` and `mcp-servers:` frontmatter** is allowed in imported files, other entries give a warning.
- **Tool merging**: `allowed:` tools are merged across all imported files
- **MCP server merging**: MCP servers defined in imported files are merged with the main workflow

### Example Tool Merging
```aw wrap
# Base workflow
---
on: issues
tools:
  github:
    allowed: [get_issue]
imports:
  - shared/extra-tools.md
---
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

### Example MCP Server Merging

```aw wrap
# Base workflow
---
on: issues
engine: copilot
imports:
  - shared/mcp/tavily.md
---
```

```aw wrap
# shared/mcp/tavily.md
---
mcp-servers:
  tavily:
    url: "https://mcp.tavily.com/mcp/?tavilyApiKey=${{ secrets.TAVILY_API_KEY }}"
    allowed: ["*"]
---
```

**Result**: Final workflow has the Tavily MCP server configured and available to the AI engine.

## Legacy Directive Syntax (Deprecated)

:::caution[Deprecated]
The `{{#import}}`, `@import`, and `@include` directive syntax is deprecated. Use the `imports:` field in frontmatter instead.

**Migration example:**
```diff
# Old approach
---
on: issues
---
- @import shared/extra-tools.md

# New approach
+ ---
+ on: issues
+ imports:
+   - shared/extra-tools.md
+ ---
```
:::

## Related Documentation

- [Packaging and Imports](/gh-aw/guides/packaging-imports/) - Complete guide to adding, updating, and importing workflows
- [CLI Commands](/gh-aw/tools/cli/) - CLI commands for workflow management
- [Workflow Structure](/gh-aw/reference/workflow-structure/) - Directory layout and organization
- [Frontmatter Options](/gh-aw/reference/frontmatter/) - All configuration options
- [Tools Configuration](/gh-aw/reference/tools/) - GitHub and other tools setup
- [MCPs](/gh-aw/guides/mcps/) - Model Context Protocol setup and configuration
