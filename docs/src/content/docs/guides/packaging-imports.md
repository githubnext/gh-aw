---
title: Packaging & Distribution
description: How to add, share, update, and import workflows from external repositories using workflow specifications and import directives.
sidebar:
  order: 2
---

## Adding Workflows

Install workflows from external repositories with optional versioning:

```bash wrap
gh aw add githubnext/agentics/ci-doctor              # short form
gh aw add githubnext/agentics/ci-doctor@v1.0.0       # with version
gh aw add githubnext/agentics/workflows/ci-doctor.md # explicit path
```

Use `--name`, `--pr`, `--force`, `--number`, `--engine`, or `--verbose` flags to customize installation. The `source` field is automatically added to workflow frontmatter for tracking origin and enabling updates.

## Updating Workflows

Keep workflows synchronized with their source repositories:

```bash wrap
gh aw update                           # update all workflows
gh aw update ci-doctor                 # update specific workflow
gh aw update ci-doctor issue-triage    # update multiple
```

Use `--major`, `--force`, `--engine`, or `--verbose` flags to control update behavior. Semantic versions (e.g., `v1.2.3`) update to latest compatible release within same major version. Branch references update to latest commit. Updates use 3-way merge; when conflicts occur, manually resolve conflict markers and run `gh aw compile`.

## Imports

Import reusable components using the `imports:` field in frontmatter. File paths are relative to the workflow location:

```yaml wrap
---
on: issues
engine: copilot
imports:
  - shared/common-tools.md
  - shared/security-setup.md
  - shared/mcp/tavily.md
---
```

During `gh aw add`, imports are expanded to track source repository (e.g., `shared/common-tools.md` becomes `githubnext/agentics/shared/common-tools.md@abc123def`).

Remote imports are automatically cached in `.github/aw/imports/` by commit SHA. This enables offline workflow compilation once imports have been downloaded. The cache is shared across different refs pointing to the same commit, reducing redundant downloads.

## Example: Modular Workflow with Imports

Create a shared Model Context Protocol (MCP) server configuration in `.github/workflows/shared/mcp/tavily.md`:

```yaml wrap
---
mcp-servers:
  tavily:
    url: "https://mcp.tavily.com/mcp/?tavilyApiKey=${{ secrets.TAVILY_API_KEY }}"
    allowed: ["*"]
---
```

Reference it in your workflow to include the Tavily MCP server alongside other tools:

```yaml wrap
---
on:
  issues:
    types: [opened]
imports:
  - shared/mcp/tavily.md
tools:
  github:
    toolsets: [issues]
---

# Research Agent
Perform web research using Tavily and respond to issues.
```

## Specification Formats and Validation

Workflow specifications require minimum 3 parts (owner/repo/workflow-name) for short form. Explicit paths must end with `.md`. Versions can be semantic tags (`@v1.0.0`), branches (`@develop`), or commit SHAs. Identifiers use alphanumeric characters with hyphens/underscores (cannot start/end with hyphen).

**Examples:**
- Repository: `owner/repo[@version]`
- Short workflow: `owner/repo/workflow[@version]` (adds `workflows/` prefix and `.md`)
- Explicit workflow: `owner/repo/path/to/workflow.md[@version]`
- GitHub URL: `https://github.com/owner/repo/blob/main/workflows/ci-doctor.md`
- Raw URL: `https://raw.githubusercontent.com/owner/repo/refs/heads/main/workflows/ci-doctor.md`

## Best Practices

Use semantic versioning for stable workflows, branches for development, and commit SHAs for immutability. Organize reusable components in a `shared/` directory with descriptive names. Review updates with `--verbose` before applying, test on branches, and keep local modifications minimal to reduce merge conflicts.

**Related:** [CLI Commands](/gh-aw/setup/cli/) | [Workflow Structure](/gh-aw/reference/workflow-structure/) | [Frontmatter](/gh-aw/reference/frontmatter/) | [Imports](/gh-aw/reference/imports/)
