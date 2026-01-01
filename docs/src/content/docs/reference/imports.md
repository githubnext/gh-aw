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

## Shared Workflow Components

Workflows without an `on` field are automatically detected as shared workflow components. These files are validated but not compiled into GitHub Actions, as they are meant to be imported by other workflows. When you attempt to compile a shared workflow directly, the compiler displays an informative message and skips compilation:

```bash wrap
$ gh aw compile shared/mcp/deepwiki.md
ℹ️  Shared agentic workflow detected: deepwiki.md

This workflow is missing the 'on' field and will be treated as a shared workflow component.
Shared workflows are reusable components meant to be imported by other workflows.

To use this shared workflow:
  1. Import it in another workflow's frontmatter:
     ---
     on: issues
     imports:
       - shared/mcp/deepwiki.md
     ---

  2. Compile the workflow that imports it

Skipping compilation.
✓ Compiled 1 workflow(s): 0 error(s), 0 warning(s)
```

This allows you to organize reusable components in your repository without generating unnecessary lock files.

## Path Formats

Import paths support local files (`shared/file.md`, `../file.md`), remote repositories (`owner/repo/file.md@v1.0.0`), and section references (`file.md#SectionName`). Optional imports use `{{#import? file.md}}` syntax in markdown.

Paths are resolved relative to the importing file, with support for nested imports and circular import protection.

## Remote Repository Imports

Import shared components from external repositories using the `owner/repo/path@ref` format:

```aw wrap
---
on: issues
engine: copilot
imports:
  - acme-org/shared-workflows/mcp/tavily.md@v1.0.0
  - acme-org/shared-workflows/tools/github-setup.md@main
---

# Issue Triage Workflow

Analyze incoming issues using imported tools and configurations.
```

Version references support semantic tags (`@v1.0.0`), branch names (`@main`, `@develop`), or commit SHAs for immutable references. See [Packaging & Distribution](/gh-aw/guides/packaging-imports/) for installation and update workflows.

## Import Cache

Remote imports are automatically cached in `.github/aw/imports/` to enable offline compilation. The cache stores imports by commit SHA, allowing different refs (branches, tags) pointing to the same commit to share cached files.

When compiling workflows with remote imports:
- First compilation downloads the import and stores it in the cache
- Subsequent compilations use the cached file, eliminating network calls
- Cache is organized by owner/repo/sha/path for efficient lookups
- Local imports are never cached and are always read from the filesystem

The cache directory is git-tracked and automatically configured with `.gitattributes` to mark cached files as generated content with conflict-free merge behavior.

## Agent Files

Import custom agent files from `.github/agents/` to customize AI engine behavior. Agent files are markdown documents with specialized instructions that modify how the AI interprets and executes workflows.

```yaml wrap
---
on: pull_request
engine: copilot
imports:
  - .github/agents/code-reviewer.md
---
```

Only one agent file can be imported per workflow. See [Custom Agent Files](/gh-aw/reference/engines/#custom-agent-files) for details on creating and using agent files.

## Frontmatter Merging

Imported files can define `tools:`, `mcp-servers:`, `services:`, and `safe-outputs:` frontmatter (other fields trigger warnings). Agent files can also define `name` and `description`. These fields are merged with the main workflow's configuration.

### Tools and Model Context Protocol (MCP) Servers

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

### Safe Outputs

Safe output configurations (like `create-issue`, `add-comment`, etc.) and safe output jobs can be imported from shared workflows. If the same safe output type is defined in both the main workflow and an imported workflow, compilation fails with a conflict error.

```aw wrap
# shared-config.md
---
safe-outputs:
  create-issue:
    title-prefix: "[bot] "
    labels: [automated]
  allowed-domains:
    - "api.example.com"
  staged: true
  jobs:
    notify:
      runs-on: ubuntu-latest
      steps:
        - run: echo "Notification sent"
---
```

```aw wrap
# main-workflow.md
---
on: issues
imports:
  - shared-config.md
---
```

The main workflow inherits the `create-issue` configuration, allowed domains, staged setting, and the `notify` job from the imported file. Safe output meta fields (like `allowed-domains`, `staged`, `env`, `github-token`, `max-patch-size`, `runs-on`) from the main workflow take precedence over imported values.

## Related Documentation

- [Packaging and Updating](/gh-aw/guides/packaging-imports/) - Complete guide to managing workflow imports
- [Frontmatter](/gh-aw/reference/frontmatter/) - Configuration options reference
- [MCPs](/gh-aw/guides/mcps/) - Model Context Protocol setup
