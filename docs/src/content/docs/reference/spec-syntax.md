---
title: Spec Syntax
description: Reference guide for repository and workflow specification syntax used in CLI commands and workflow source fields.
sidebar:
  order: 1600
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

Format: `owner/repo/workflow-name[@version]` or `owner/repo/path/to/workflow.md[@version]` or full GitHub URL

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

**GitHub URL form**: Full GitHub URL to workflow file
```bash
gh aw add https://github.com/owner/repo/blob/main/workflows/ci-doctor.md
gh aw add https://github.com/owner/repo/blob/v1.0.0/custom/path/workflow.md
gh aw add https://github.com/owner/repo/tree/develop/workflows/helper.md
```

**GitHub /files/ path form**: GitHub UI file path format
```bash
gh aw add owner/repo/files/main/.github/workflows/ci-doctor.md
gh aw add owner/repo/files/fc7992627494253a869e177e5d1985d25f3bb316/workflows/helper.md
```

**Raw GitHub URL form**: raw.githubusercontent.com URLs
```bash
gh aw add https://raw.githubusercontent.com/owner/repo/refs/heads/main/workflows/ci-doctor.md
gh aw add https://raw.githubusercontent.com/owner/repo/refs/tags/v1.0.0/workflows/helper.md
gh aw add https://raw.githubusercontent.com/owner/repo/COMMIT_SHA/workflows/helper.md
```

**Validation:**
- Minimum 3 parts (owner/repo/workflow-name) for spec format
- Explicit paths must end with `.md` extension
- Version optional (tag, branch, or commit SHA)
- GitHub URLs support github.com and raw.githubusercontent.com domains
- GitHub URLs must use /blob/, /tree/, or /raw/ format for github.com
- GitHub URLs automatically extract branch/tag/commit from the URL path
- /files/ format automatically extracts ref from path

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
gh aw add https://github.com/githubnext/agentics/blob/main/workflows/ci-doctor.md  # GitHub URL
gh aw add githubnext/agentics/files/main/workflows/ci-doctor.md  # /files/ path format
gh aw add https://raw.githubusercontent.com/githubnext/agentics/refs/heads/main/workflows/ci-doctor.md  # raw URL
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

- [Packaging and Updating](/gh-aw/guides/packaging-imports/) - Complete guide to adding, updating, and importing workflows
- [CLI Commands](/gh-aw/tools/cli/) - Full CLI reference
- [Workflow Structure](/gh-aw/reference/workflow-structure/) - Directory layout
- [Frontmatter](/gh-aw/reference/frontmatter/) - Configuration options
- [Imports](/gh-aw/reference/imports/) - Modularizing workflows
