---
title: Packaging and Updating
description: Complete guide to adding, updating, and importing workflows from external repositories using workflow specifications and import directives.
sidebar:
  order: 600
---

This guide covers adding, updating, and importing agentic workflows from external repositories using workflow specifications and import directives.

## Adding Workflows

Install workflows from external repositories:

```bash
gh aw add githubnext/agentics/ci-doctor              # short form
gh aw add githubnext/agentics/ci-doctor@v1.0.0       # with version
gh aw add githubnext/agentics/workflows/ci-doctor.md # explicit path
```

**Options:**

| Flag | Description |
|------|-------------|
| `--name NAME` | Custom workflow name |
| `--pr` | Create pull request for review |
| `--force` | Overwrite existing files |
| `--number N` | Create N numbered copies |
| `--engine ENGINE` | Override AI engine |
| `--verbose` | Show detailed output |

The `source` field is automatically added to workflow frontmatter for tracking origin and enabling updates:

```yaml
source: "githubnext/agentics/workflows/ci-doctor.md@v1.0.0"
```

## Updating Workflows

Keep workflows in sync with source repositories:

```bash
gh aw update                           # update all workflows
gh aw update ci-doctor                 # update specific workflow
gh aw update ci-doctor issue-triage    # update multiple
```

**Options:**

| Flag | Description |
|------|-------------|
| `--major` | Allow major version updates |
| `--force` | Update even if no changes detected |
| `--engine ENGINE` | Override AI engine for compilation |
| `--verbose` | Show detailed resolution steps |

**Update Behavior:**

- **Semantic versions** (e.g., `v1.2.3`): Updates to latest compatible release within same major version (use `--major` for major updates)
- **Branch references** (e.g., `main`): Updates to latest commit on that branch
- **No reference**: Updates to latest commit on default branch

**Conflict Resolution:**

Updates use 3-way merge via `git merge-file`. When conflicts occur:

1. Review conflict markers in the workflow file
2. Manually edit to keep desired changes
3. Remove conflict markers (`<<<<<<<`, `|||||||`, `=======`, `>>>>>>>`)
4. Run `gh aw compile` to recompile

## Imports Field

Import reusable components using the `imports:` field in frontmatter:

```yaml
---
on: issues
engine: copilot
imports:
  - shared/common-tools.md
  - shared/security-setup.md
  - shared/mcp/tavily.md
---
```

File paths are relative to the workflow location. During `gh aw add`, imports are expanded to track source repository:

```yaml
# Before (source repo)
imports: [shared/common-tools.md]

# After (your repo)
imports: [githubnext/agentics/shared/common-tools.md@abc123def]
```

## Examples

### Version Management

```bash
# Add versioned workflow
gh aw add githubnext/agentics/ci-doctor@v1.0.0

# Update to latest v1.x
gh aw update ci-doctor

# Allow major version update
gh aw update ci-doctor --major

# Track development branch
gh aw add githubnext/agentics/experimental@develop
gh aw update experimental
```

### Modular Workflows with Imports

Create shared MCP server configuration:

```aw wrap
# .github/workflows/shared/mcp/tavily.md
---
mcp-servers:
  tavily:
    url: "https://mcp.tavily.com/mcp/?tavilyApiKey=${{ secrets.TAVILY_API_KEY }}"
    allowed: ["*"]
---
```

Use in workflow:

```aw wrap
# .github/workflows/research-agent.md
---
on:
  issues:
    types: [opened]
imports:
  - shared/mcp/tavily.md
tools:
  github:
    allowed: [add_issue_comment]
---

# Research Agent
Perform web research using Tavily and respond to issues.
```

The compiled workflow includes both Tavily MCP server and GitHub tools.

## Validation

**Common Errors:**

| Error | Fix |
|-------|-----|
| `repository must be in format 'org/repo'` | Include owner and repo with slash |
| `workflow specification must be in format 'owner/repo/workflow-name[@version]'` | Include owner, repo, and workflow name |
| `workflow specification with path must end with '.md' extension` | Add `.md` extension for explicit paths |
| `'owner-/repo' does not look like a valid GitHub repository` | Don't start/end identifiers with hyphens |

**Specification Requirements:**

- Minimum 3 parts (owner/repo/workflow-name) for short form
- Explicit paths must end with `.md`
- Version optional (tag, branch, or commit SHA)
- Identifiers: alphanumeric with hyphens/underscores (cannot start/end with hyphen)

## Best Practices

**Version Management:**
- Use semantic versioning for stable workflows (`@v1.0.0`)
- Use branches for development (`@develop`)
- Pin to commit SHAs for immutability (`@abc123def...`)

**Import Organization:**
- Create `shared/` directory for reusable components
- Use descriptive names (`github-tools.md`, `security-notice.md`)
- Keep imports focused on specific functionality

**Workflow Updates:**
- Review changes with `gh aw update --verbose` before applying
- Test updates on a branch before merging
- Resolve conflicts promptly to avoid compilation failures
- Keep local modifications minimal to reduce merge conflicts
- Preserve the `source` field to enable updates

## Reference

**Specification Formats:**

| Format | Example | Notes |
|--------|---------|-------|
| Repository | `owner/repo[@version]` | Version optional |
| Short workflow | `owner/repo/workflow[@version]` | Adds `workflows/` prefix and `.md` |
| Explicit workflow | `owner/repo/path/to/workflow.md[@version]` | Full path required |
| GitHub URL | `https://github.com/owner/repo/blob/main/workflows/ci-doctor.md` | Extracts ref from URL |
| Raw URL | `https://raw.githubusercontent.com/owner/repo/refs/heads/main/workflows/ci-doctor.md` | Direct file access |

**Version Types:**

- **Semantic versions**: `v1.0.0`, `v1.2.3`, `1.0.0`, `v2.0.0-beta`
- **Branch names**: `main`, `develop`, `feature/new-feature`
- **Commit SHAs**: `abc123def456789012345678901234567890abcdef` (40 chars)
- **No version**: Uses repository default branch

**Source Field:**

Automatically added to workflow frontmatter to track origin:

```yaml
source: "githubnext/agentics/workflows/ci-doctor.md@v1.0.0"
```

## Related Documentation

- [CLI Commands](/gh-aw/tools/cli/) - Complete CLI reference
- [Workflow Structure](/gh-aw/reference/workflow-structure/) - Directory layout and organization
- [Frontmatter](/gh-aw/reference/frontmatter/) - All configuration options
- [Imports](/gh-aw/reference/imports/) - Include directive details
