---
title: Imports
description: Learn how to modularize and reuse workflow components across multiple workflows using the imports field in frontmatter for better organization and maintainability.
sidebar:
  order: 325
---

Imports allow you to modularize and reuse workflow components across multiple workflows. By extracting shared configurations into separate files, you can maintain consistency, reduce duplication, and simplify updates across your agentic workflows.

## Prerequisites

- Basic understanding of [workflow structure](/gh-aw/reference/workflow-structure/)
- Familiarity with [frontmatter configuration](/gh-aw/reference/frontmatter/)

## Quick Example

Create a reusable MCP server configuration:

**File: `.github/workflows/shared/tavily.md`**
```yaml
---
mcp-servers:
  tavily:
    url: "https://mcp.tavily.com/mcp/?tavilyApiKey=${{ secrets.TAVILY_API_KEY }}"
    allowed: ["*"]
---
```

**Import it in your workflow:**
```yaml
---
on: issues
engine: copilot
imports:
  - shared/tavily.md
---

# Research Assistant

You are a research assistant with web search capabilities.
```

The Tavily MCP server is now available to your workflow without duplicating configuration.

## Why Use Imports

Imports help you:

- **Reduce duplication** - Define common tools and services once
- **Maintain consistency** - Update shared configurations in one place
- **Organize workflows** - Separate concerns into logical modules
- **Share configurations** - Reuse components across teams and repositories

## Import Syntax

You can specify imports in two ways: frontmatter or markdown.

### Frontmatter Imports

Use the `imports:` field in your workflow frontmatter:

```yaml
---
on: issues
engine: copilot
imports:
  - shared/common-tools.md
  - shared/mcp/tavily.md
  - shared/security-setup.md
---

# Your Workflow

Workflow instructions here...
```

:::tip
Frontmatter imports are processed before the workflow runs, making them ideal for tool and service configurations.
:::

### Markdown Imports

Use the `{{#import ...}}` directive anywhere in your markdown:

```yaml
---
on: issues
engine: copilot
---

# Your Workflow

Workflow instructions here...

{{#import shared/common-tools.md}}
```

:::note
Markdown imports allow you to include shared instruction text and configuration at specific points in your workflow.
:::

## Path Formats

Import paths can reference local files, remote repositories, or specific sections within files.

### Local File Paths

Paths are resolved relative to the importing file:

```yaml
imports:
  - shared/tools.md           # relative to workflow directory
  - ../common/services.md     # parent directory
  - ./local-config.md         # current directory
```

### Remote Repository Imports

Import from external GitHub repositories with optional version pinning:

```yaml
imports:
  - owner/repo/shared/tools.md@v1.0.0      # specific version tag
  - owner/repo/shared/tools.md@main        # specific branch
  - owner/repo/shared/tools.md@abc123      # specific commit
  - owner/repo/shared/tools.md             # latest from default branch
```

:::caution
Remote imports without version pins may change unexpectedly. Use semantic versioning for production workflows.
:::

### Section References

Import specific sections from a file using hash syntax:

```yaml
imports:
  - shared/tools.md#GitHubTools     # only the GitHubTools section
  - shared/docs.md#Prerequisites    # only the Prerequisites section
```

### Optional Imports

Make imports optional using `{{#import? ...}}` syntax in markdown:

```markdown
{{#import? optional-config.md}}
```

If `optional-config.md` doesn't exist, the workflow continues without error.

:::tip
Use optional imports for environment-specific configurations that may not exist in all deployments.
:::

## Frontmatter Merging

Imported files can only define these frontmatter fields:

- `tools:` - Tool configurations
- `mcp-servers:` - MCP server configurations
- `services:` - Service configurations

All other frontmatter fields trigger warnings and are ignored.

### How Merging Works

Configurations from imported files are merged with your main workflow. Arrays are concatenated, and objects are deep-merged.

**Shared tools file:**
```yaml
---
# shared/github-tools.md
tools:
  github:
    allowed: ["create_issue", "add_comment"]
---
```

**Main workflow:**
```yaml
---
on: issues
engine: copilot
imports:
  - shared/github-tools.md
tools:
  github:
    allowed: ["search_issues"]
---
```

**Result after merging:**
```yaml
tools:
  github:
    allowed: ["create_issue", "add_comment", "search_issues"]
```

### Complete Example

**File: `.github/workflows/shared/research-tools.md`**
```yaml
---
mcp-servers:
  tavily:
    url: "https://mcp.tavily.com/mcp/?tavilyApiKey=${{ secrets.TAVILY_API_KEY }}"
    allowed: ["*"]

tools:
  github:
    allowed: ["search_code", "search_issues"]
---
```

**File: `.github/workflows/issue-researcher.md`**
```yaml
---
on:
  issues:
    types: [opened, labeled]

engine: copilot

imports:
  - shared/research-tools.md

tools:
  github:
    allowed: ["add_comment"]
---

# Issue Research Assistant

When an issue is opened or labeled, research relevant code and
similar issues, then post your findings as a comment.
```

The final workflow has access to:
- Tavily MCP server for web search
- GitHub tools: `search_code`, `search_issues`, and `add_comment`

## Advanced Features

### Nested Imports

Imported files can themselves import other files:

```yaml
# main-workflow.md
imports:
  - shared/base-config.md

# shared/base-config.md
imports:
  - common/tools.md
  - common/services.md
```

:::note
Circular imports are automatically detected and prevented.
:::

### Import Resolution Order

Imports are processed in the order listed. Later imports override earlier ones if they define the same keys.

```yaml
imports:
  - base-config.md        # loaded first
  - override-config.md    # overrides base-config.md
```

## Best Practices

1. **Organize shared files** - Use a `shared/` directory structure
2. **Use descriptive names** - Name files by purpose (e.g., `github-tools.md`, `mcp-servers.md`)
3. **Version remote imports** - Pin to specific versions in production
4. **Document shared files** - Add markdown documentation to imported files
5. **Keep imports focused** - Each shared file should serve one clear purpose

## Related Documentation

- [Packaging and Updating](/gh-aw/guides/packaging-imports/) - Complete guide to managing workflow imports
- [Frontmatter](/gh-aw/reference/frontmatter/) - Configuration options reference
- [MCPs](/gh-aw/guides/mcps/) - Model Context Protocol setup
- [Workflow Structure](/gh-aw/reference/workflow-structure/) - Overall workflow organization
