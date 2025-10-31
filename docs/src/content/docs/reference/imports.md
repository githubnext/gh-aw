---
title: Imports
description: Learn how to modularize and reuse workflow components across multiple workflows using the imports field in frontmatter for better organization and maintainability.
sidebar:
  order: 325
---

Using imports in frontmatter or markdown allows you to modularize and reuse workflow components across multiple workflows.

## Syntax

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

## Path Formats

Import paths support local files (`shared/file.md`, `../file.md`), remote repositories (`owner/repo/file.md@v1.0.0`), and section references (`file.md#SectionName`). Optional imports use `{{#import? file.md}}` syntax in markdown.

Paths are resolved relative to the importing file, with support for nested imports and circular import protection.

## Agent Files

Import custom agent files from `.github/agents/` to customize AI engine behavior. Agent files are markdown documents with specialized instructions that modify how the AI interprets and executes workflows.

```yaml
---
on: pull_request
engine: copilot
imports:
  - .github/agents/code-reviewer.md
---
```

Only one agent file can be imported per workflow. See [Custom Agent Files](/gh-aw/reference/engines/#custom-agent-files) for details on creating and using agent files.

## Frontmatter Merging

Imported files can only define `tools:`, `mcp-servers:`, and `services:` frontmatter (other fields trigger warnings). Agent files can also define `name` and `description`. These fields are merged with the main workflow's configuration.

### Example

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

The imported MCP server configuration is merged into the final workflow, making it available to the AI engine.

## Related Documentation

- [Packaging and Updating](/gh-aw/guides/packaging-imports/) - Complete guide to managing workflow imports
- [Frontmatter](/gh-aw/reference/frontmatter/) - Configuration options reference
- [MCPs](/gh-aw/guides/mcps/) - Model Context Protocol setup
