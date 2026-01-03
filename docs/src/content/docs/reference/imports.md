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

Workflows without an `on` field are shared workflow components. These files are validated but not compiled into GitHub Actions - they're meant to be imported by other workflows. The compiler skips them with an informative message, allowing you to organize reusable components without generating unnecessary lock files.

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

Remote imports are cached in `.github/aw/imports/` to enable offline compilation. First compilation downloads and caches the import by commit SHA; subsequent compilations use the cached file. The cache is git-tracked with `.gitattributes` configured for conflict-free merges. Local imports are never cached.

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

Imported files can define `tools:`, `mcp-servers:`, `services:`, and `safe-outputs:` frontmatter (other fields trigger warnings). Agent files can also define `name` and `description`. These fields merge with the main workflow's configuration.

**MCP Servers:** Import MCP server configurations that merge into the final workflow:

```aw wrap
# Base workflow imports shared/mcp/tavily.md
---
on: issues
engine: copilot
imports:
  - shared/mcp/tavily.md
---
```

**Safe Outputs:** Import safe output configurations and jobs. Conflicts between main and imported safe outputs fail compilation. Meta fields from the main workflow take precedence:

```aw wrap
# Main workflow inherits create-issue config and notify job
---
on: issues
imports:
  - shared-config.md  # defines create-issue, allowed-domains, jobs.notify
---
```

## Related Documentation

- [Packaging and Updating](/gh-aw/guides/packaging-imports/) - Complete guide to managing workflow imports
- [Frontmatter](/gh-aw/reference/frontmatter/) - Configuration options reference
- [MCPs](/gh-aw/guides/mcps/) - Model Context Protocol setup
