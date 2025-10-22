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

## Supported Path Formats

The imports system supports multiple path formats for maximum flexibility:

### 1. Local Relative Paths

Standard file paths relative to the importing workflow file:

```yaml
imports:
  - shared/common-tools.md
  - shared/mcp/tavily.md
  - ../templates/security-notice.md
```

**Notes:**
- Paths starting with `shared/` are treated as local files
- Paths starting with `.` or `..` are treated as relative local paths
- Paths starting with `/` are treated as absolute local paths

### 2. Workflowspec Format (Remote GitHub Files)

Import files directly from GitHub repositories using the workflowspec format:

**Format:** `owner/repo/path/to/file.md[@ref][#section]`

```yaml
imports:
  # Basic format (uses default branch)
  - githubnext/agentics/shared/security-tools.md
  
  # With version/branch reference
  - githubnext/agentics/shared/security-tools.md@v1.0.0
  - githubnext/agentics/shared/security-tools.md@main
  - githubnext/agentics/shared/security-tools.md@abc123def
  
  # With section reference
  - githubnext/agentics/docs/guidelines.md#Security
  
  # Combined: version and section
  - githubnext/agentics/docs/guidelines.md@v2.0.0#BestPractices
```

**Components:**
- **owner**: GitHub username or organization
- **repo**: Repository name
- **path**: Path to the file within the repository (must be at least one path segment)
- **@ref** (optional): Branch name, tag, or commit SHA (defaults to `main`)
- **#section** (optional): Markdown section name to extract (H1, H2, or H3 headers)

**Requirements:**
- Must have at least 3 parts: `owner/repo/path`
- Cannot start with `.`, `shared/`, or `/` (these indicate local paths)
- File is downloaded from GitHub at compile time

### 3. Section References

Import only a specific section from a markdown file:

```yaml
imports:
  - shared/guidelines.md#SecurityNotice
  - githubnext/agentics/docs/best-practices.md@v1.0.0#Testing
```

**Behavior:**
- Extracts content from the specified header to the next same-level or higher-level header
- Supports H1 (`#`), H2 (`##`), and H3 (`###`) headers
- Section names are case-sensitive
- Works with both local and remote (workflowspec) paths

### 4. Optional Imports

Mark imports as optional using the `?` modifier (supported in markdown directives only):

```markdown
{{#import? shared/optional-config.md}}
{{#import? githubnext/agentics/experimental/beta-feature.md@develop}}
```

**Behavior:**
- If the file doesn't exist, a friendly informational message is shown
- Compilation continues without error
- Useful for environment-specific or conditional configurations

**Note:** Optional imports are only supported in markdown `{{#import?}}` directives, not in frontmatter `imports:` field.

### Import Path Resolution

- **Relative paths**: Resolved relative to the importing file's directory
- **Workflowspec paths**: Downloaded from GitHub and cached temporarily
- **Nested imports**: Imported files can import other files
- **Circular protection**: System prevents infinite import loops
- **Section extraction**: Applied after downloading/reading the file

## Frontmatter Merging

When importing files, frontmatter fields are merged with the main workflow:
- **Only `tools:`, `mcp-servers:`, and `services:` frontmatter** is allowed in imported files, other entries give a warning.
- **Tool merging**: `allowed:` tools are merged across all imported files
- **MCP server merging**: MCP servers defined in imported files are merged with the main workflow
- **Services merging**: Docker services defined in imported files are merged with the main workflow

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

### Example Services Merging

```aw wrap
# Base workflow
---
on: issues
engine: copilot
imports:
  - shared/mcp/jupyter.md
---
```

```aw wrap
# shared/mcp/jupyter.md
---
services:
  jupyter:
    image: jupyter/base-notebook:latest
    ports:
      - 8888:8888
    env:
      JUPYTER_TOKEN: ${{ github.run_id }}

mcp-servers:
  jupyter:
    type: http
    url: "http://jupyter:3000"
    allowed: ["*"]
---
```

**Result**: Final workflow has the Jupyter service and MCP server configured with shared networking.

## Path Format Examples

### Local Repository Files

Import files from within the same repository:

```yaml
---
on: issues
imports:
  # From shared directory (same level as .github/workflows/)
  - shared/tools/github-tools.md
  
  # From parent directory
  - ../templates/common-setup.md
  
  # From nested shared directory
  - shared/mcp/tavily.md
  - shared/mcp/playwright.md
---
```

### Remote Repository Files

Import files from other GitHub repositories:

```yaml
---
on: issues
imports:
  # Basic remote import (uses default branch)
  - githubnext/agentics/shared/security-notice.md
  
  # With semantic version tag
  - githubnext/agentics/shared/tools/web-tools.md@v1.2.0
  
  # With branch reference
  - githubnext/agentics/shared/experimental/beta-feature.md@develop
  
  # With commit SHA (for immutability)
  - githubnext/agentics/shared/tools/mcp-servers.md@abc123def456
---
```

### Section Extraction

Import specific sections from larger documentation files:

```yaml
---
on: issues
imports:
  # Local file, specific section
  - shared/docs/guidelines.md#Security
  
  # Remote file, specific section with version
  - githubnext/agentics/docs/best-practices.md@v2.0.0#ErrorHandling
---
```

### Markdown Import Directives

Use `{{#import}}` directives in workflow markdown content:

```aw wrap
---
on: issues
engine: copilot
tools:
  github:
    allowed: [add_issue_comment]
---

# Issue Triage Workflow

{{#import shared/security-notice.md#Warning}}

## Instructions

Please analyze the issue and provide a helpful response.

{{#import? shared/optional-guidelines.md}}
```

### Combined Example

A complete workflow using multiple import formats:

```aw wrap
---
on:
  issues:
    types: [opened]
permissions:
  contents: read
  issues: write
engine: copilot
imports:
  # Local shared tools
  - shared/tools/github-tools.md
  
  # Remote MCP configuration with version
  - githubnext/agentics/shared/mcp/tavily.md@v1.0.0
  
  # Security guidelines from specific section
  - shared/docs/security.md#SafetyNotice
---

# Advanced Issue Analyzer

{{#import shared/templates/issue-analysis-prompt.md}}

Analyze the issue using web research capabilities and GitHub API.

{{#import? shared/experimental/advanced-features.md}}
```

## Related Documentation

- [Packaging and Updating](/gh-aw/guides/packaging-imports/) - Complete guide to adding, updating, and importing workflows
- [CLI Commands](/gh-aw/tools/cli/) - CLI commands for workflow management
- [Workflow Structure](/gh-aw/reference/workflow-structure/) - Directory layout and organization
- [Frontmatter](/gh-aw/reference/frontmatter/) - All configuration options
- [Tools](/gh-aw/reference/tools/) - GitHub and other tools setup
- [MCPs](/gh-aw/guides/mcps/) - Model Context Protocol setup and configuration
