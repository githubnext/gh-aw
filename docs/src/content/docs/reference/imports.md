---
title: Imports
description: Learn how to modularize and reuse workflow components across multiple workflows using the imports field in frontmatter for better organization and maintainability.
sidebar:
  order: 325
---

The `imports` fields in frontmatter or markdown allow you to modularize and reuse workflow components across multiple workflows.

## Overview

Imports can be specified either in frontmatter or in markdown. In frontmatter the `imports:` field is used:

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

In markdown, use the special `{{#import ...}}` directive:

```aw wrap
---
...
---

# Your Workflow

Workflow instructions here...

{{#import shared/common-tools.md}}
```

### Import Path Resolution

- **Relative paths**: Resolved relative to the importing file
- **Nested imports**: Imported files can import other files
- **Circular protection**: System prevents infinite import loops

## Frontmatter Merging

When importing files, frontmatter fields are merged with the main workflow:
- **Only `tools:` and `mcp-servers:` frontmatter** is allowed in imported files, other entries give a warning.
- **Tool merging**: `allowed:` tools are merged across all imported files
- **MCP server merging**: MCP servers defined in imported files are merged with the main workflow

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

## Related Documentation

- [Packaging and Updating](/gh-aw/guides/packaging-imports/) - Complete guide to adding, updating, and importing workflows
- [CLI Commands](/gh-aw/tools/cli/) - CLI commands for workflow management
- [Workflow Structure](/gh-aw/reference/workflow-structure/) - Directory layout and organization
- [Frontmatter](/gh-aw/reference/frontmatter/) - All configuration options
- [Tools](/gh-aw/reference/tools/) - GitHub and other tools setup
- [MCPs](/gh-aw/guides/mcps/) - Model Context Protocol setup and configuration
