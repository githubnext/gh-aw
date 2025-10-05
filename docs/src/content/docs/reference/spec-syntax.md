---
title: Spec Syntax
description: Reference guide for repository and workflow specification syntax used in CLI commands and workflow source fields.
sidebar:
  order: 12
---

Specification syntax for referencing repositories and workflows in CLI commands and workflow frontmatter.

## Repository Specification (RepoSpec)

Format: `owner/repo[@version]`

**Components:**
- `owner` - GitHub username or organization
- `repo` - Repository name  
- `version` - Optional tag, branch, or commit SHA

**Examples:**
```bash
gh aw add githubnext/agentics              # default branch
gh aw add githubnext/agentics@v1.0.0       # version tag
gh aw add githubnext/agentics@main         # branch
```

**Validation:**
- Owner and repo required in format `owner/repo`
- Alphanumeric, hyphens, underscores (cannot start/end with hyphen)
- Version can be tag, branch, or 40-character commit SHA

## Workflow Specification (WorkflowSpec)

Format: `owner/repo/workflow-name[@version]` or `owner/repo/path/to/workflow.md[@version]`

**Short form** (3 parts): Automatically adds `workflows/` prefix and `.md` extension
```bash
gh aw add owner/repo/workflow@v1.0.0
# → workflows/workflow.md@v1.0.0
```

**Explicit form** (4+ parts): Requires full path with `.md` extension
```bash
gh aw add owner/repo/workflows/ci-doctor.md@v1.0.0
gh aw add owner/repo/custom/path/workflow.md@main
```

**Validation:**
- Minimum 3 parts (owner/repo/workflow-name)
- Explicit paths must end with `.md` extension
- Version optional (tag, branch, or commit SHA)

## Source Specification (SourceSpec)

Used in workflow frontmatter to track workflow origin.

Format: `source: "owner/repo/path/to/workflow.md[@ref]"`

**Examples:**
```yaml
source: "githubnext/agentics/workflows/ci-doctor.md@v1.0.0"  # tag
source: "githubnext/agentics/workflows/ci-doctor.md@main"    # branch
source: "githubnext/agentics/workflows/ci-doctor.md"          # default branch
```

## Version References

**Semantic version tags** (with or without `v` prefix):
```
v1.0.0, v1.2.3, 1.0.0, v2.0.0-beta
```

**Branch names:**
```
main, develop, feature/new-feature
```

**Commit SHAs** (40-character hexadecimal):
```
abc123def456789012345678901234567890abcdef
```

**No version** (uses repository default branch):
```
owner/repo/workflow
```

## CLI Commands

**Add workflow:**
```bash
gh aw add githubnext/agentics/ci-doctor              # short form
gh aw add githubnext/agentics/ci-doctor@v1.0.0       # with version
gh aw add githubnext/agentics/workflows/ci-doctor.md@main  # explicit path
```

**Update workflow:**
```bash
gh aw update                    # all workflows with source field
gh aw update ci-doctor          # specific workflow
gh aw update ci-doctor --major  # allow major version bumps
```

**Remove workflow:**
```bash
gh aw remove ci-doctor
```

## Common Errors

**Invalid repository format:**
```
Error: repository must be in format 'org/repo'
```
Fix: Include both owner and repo with slash separator.

**Invalid workflow format:**
```
Error: workflow specification must be in format 'owner/repo/workflow-name[@version]'
```
Fix: Include owner, repo, and workflow name/path.

**Missing extension:**
```
Error: workflow specification with path must end with '.md' extension
```
Fix: Add `.md` extension for explicit paths.

**Invalid identifier:**
```
Error: 'owner-/repo' does not look like a valid GitHub repository
```
Fix: Don't start or end identifiers with hyphens.

## Related Documentation

- [CLI Commands](/gh-aw/tools/cli/) - Full CLI reference
- [Workflow Structure](/gh-aw/reference/workflow-structure/) - Directory layout
- [Frontmatter Options](/gh-aw/reference/frontmatter/) - Configuration options
- [Include Directives](/gh-aw/reference/include-directives/) - Modularizing workflows
