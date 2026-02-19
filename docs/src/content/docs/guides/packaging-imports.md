---
title: Reusing Workflows
description: How to reuse, add, share, update, and distribute workflows.
sidebar:
  order: 2
---

## Adding Workflows

You can add any existing workflow you have access to from external repositories.

Use the `gh aw add-wizard` command to add a workflow with interactive guidance:

```bash wrap
gh aw add-wizard <workflow-url>
```

For example, to add the `daily-repo-status` workflow from the `githubnext/agentics` repository:

```bash wrap
# Full GitHub URL
gh aw add-wizard https://github.com/githubnext/agentics/blob/main/workflows/daily-repo-status.md

# Short form (for workflows in top-level workflows/ directory)
gh aw add-wizard githubnext/agentics/daily-repo-status
```

This checks requirements, adds the workflow markdown file to your repository, and generates the corresponding YAML workflow. After adding, commit and push the changes to your repository.

For non-interactive installation, use `gh aw add` with optional versioning. By default this looks in the `workflows/` directory, but you can specify an explicit path if needed:

```bash wrap
gh aw add githubnext/agentics/ci-doctor              # short form
gh aw add githubnext/agentics/ci-doctor@v1.0.0       # with version
gh aw add githubnext/agentics/workflows/ci-doctor.md # explicit path
```

Use `--name`, `--pr`, `--force`, `--engine`, or `--verbose` flags to customize installation. The `source` field is automatically added to workflow frontmatter for tracking origin and enabling updates.

> [!NOTE]
> Check carefully that the workflow comes from a trusted source and is appropriate for your use in your repository. Review the workflow's content and understand what it does before adding it to your repository.

## Updating Workflows

When you add a workflow, a tracking `source:` entry remembers where it came from. You can keep workflows synchronized with their source repositories:

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

### Import Merge Behavior

The compiler uses a **breadth-first search (BFS)** algorithm to process imports:

1. **Queueing**: Main workflow's direct imports are added to queue
2. **Processing**: Each import is loaded, parsed, and its configuration extracted
3. **Recursion**: Nested imports from imported files are added to queue
4. **Cycle Detection**: Already-processed files are skipped to prevent infinite loops
5. **Merging**: Configurations are merged according to field-specific rules
6. **Validation**: Final merged configuration is validated for conflicts

**Processing order** follows BFS traversal:

```text
Main Workflow
├── shared/tools.md        (1st - direct import)
│   └── shared/base.md     (3rd - nested from tools.md)
└── shared/mcp.md          (2nd - direct import)
    └── shared/network.md  (4th - nested from mcp.md)
```

**Merge semantics by field**:

| Field | Merge Strategy | Main Workflow Precedence |
|-------|---------------|-------------------------|
| `tools:` | Deep merge, arrays concatenate | Keys merged, arrays deduplicated |
| `mcp-servers:` | Override by name | ❌ Imported servers override |
| `network:` | Union of allowed domains | ✅ Mode and firewall settings |
| `permissions:` | Validation only | ✅ Main must explicitly declare |
| `safe-outputs:` | Type-level override | ✅ Main overrides imported types |
| `safe-outputs.jobs:` | Conflict detection | Neither (names must be unique) |
| `runtimes:` | Version override | ✅ Main versions override |
| `services:` | Conflict detection | Neither (must be unique) |
| `steps:` | Array prepend | Imported first, then main |
| `jobs:` | Not importable | ✅ Main only (ignored in imports) |

**Example: Tool merging**

```yaml wrap
# shared/tools.md
tools:
  bash:
    allowed: [read, list]
  github:
    toolsets: [issues]

# main.md imports shared/tools.md
tools:
  bash:
    allowed: [write]  # Result: [read, list, write]
  web-fetch: {}       # Added to merged tools
```

**Example: Safe output overriding**

```yaml wrap
# shared/outputs.md
safe-outputs:
  create-issue:
    title-prefix: "[shared] "

# main.md imports shared/outputs.md
safe-outputs:
  create-issue:
    title-prefix: "[main] "  # Overrides imported config
```

See [Imports Reference](/gh-aw/reference/imports/) for complete merge semantics.

## Importing Agent Files

Agent files provide specialized AI instructions and behavior. See [Importing Copilot Copilot Agent Files](/gh-aw/reference/copilot-custom-agents/) for details on creating and importing agent files from external repositories.

## Example: Modular Workflow with Imports

Create a shared Model Context Protocol (MCP) server configuration in `.github/workflows/shared/mcp/tavily.md`:

```yaml wrap
---
mcp-servers:
  tavily:
    url: "https://mcp.tavily.com/mcp/?tavilyApiKey=${{ secrets.TAVILY_API_KEY }}"
    allowed: ["*"]
network:
  allowed:
    - mcp.tavily.com
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
permissions:
  contents: read
  issues: write
---

# Research Agent
Perform web research using Tavily and respond to issues.
```

**Result**: The compiled workflow includes both the Tavily MCP server from the import and the GitHub tools from the main workflow, with network permissions automatically merged to allow access to both `mcp.tavily.com` and GitHub API endpoints.

## Specification Formats and Validation

Workflow and import specifications require minimum 3 parts (owner/repo/path) for remote imports. Explicit paths must end with `.md`. Versions can be semantic tags (`@v1.0.0`), branches (`@develop`), or commit SHAs. Identifiers use alphanumeric characters with hyphens/underscores (cannot start/end with hyphen).

**Examples:**

- Repository: `owner/repo[@version]`
- Short workflow: `owner/repo/workflow[@version]` (adds `workflows/` prefix and `.md`)
- Explicit workflow: `owner/repo/path/to/workflow.md[@version]`
- Agent file import: `owner/repo/.github/agents/agent-name.md[@version]`
- Shared import: `owner/repo/shared/tools/config.md[@version]`
- GitHub URL: `https://github.com/owner/repo/blob/main/workflows/ci-doctor.md`
- Raw URL: `https://raw.githubusercontent.com/owner/repo/refs/heads/main/workflows/ci-doctor.md`

## Best Practices

Use semantic versioning for stable workflows and agent files, branches for development, and commit SHAs for immutability. Organize reusable components in `shared/` directories and agent files in `.github/agents/` with descriptive names. Review updates with `--verbose` before applying, test on branches, and keep local modifications minimal to reduce merge conflicts.

**Related:** [CLI Commands](/gh-aw/setup/cli/) | [Workflow Structure](/gh-aw/reference/workflow-structure/) | [Frontmatter](/gh-aw/reference/frontmatter/) | [Imports](/gh-aw/reference/imports/) | [Copilot Agent Files](/gh-aw/reference/copilot-custom-agents/)
