---
title: Spec Syntax
description: Reference guide for repository and workflow specification syntax used in CLI commands and workflow source fields.
sidebar:
  order: 12
---

This guide explains the specification syntax used to reference repositories and workflows in CLI commands and workflow frontmatter.

## Repository Specification (RepoSpec)

Repository specifications identify a GitHub repository with an optional version reference.

### Format

```
owner/repo[@version]
```

### Components

- **`owner`**: GitHub username or organization name
- **`repo`**: Repository name
- **`version`**: Optional reference (tag, branch, or commit SHA)

### Examples

**Repository with version tag:**
```bash
gh aw add githubnext/agentics@v1.0.0
```

**Repository with branch:**
```bash
gh aw add githubnext/agentics@main
```

**Repository without version (uses default branch):**
```bash
gh aw add githubnext/agentics
```

**Repository with commit SHA:**
```bash
gh aw add githubnext/agentics@abc123def456789012345678901234567890abcdef
```

### Validation Rules

- **Owner and repo required**: Must be in format `owner/repo`
- **Valid GitHub identifiers**: 
  - Alphanumeric characters, hyphens (`-`), and underscores (`_`)
  - Cannot start or end with hyphen
- **Version optional**: Can be tag, branch, or 40-character commit SHA

## Workflow Specification (WorkflowSpec)

Workflow specifications identify a specific workflow file in a repository with an optional version reference.

### Format

```
owner/repo/workflow-name[@version]
owner/repo/path/to/workflow.md[@version]
```

### Components

- **`owner`**: GitHub username or organization name
- **`repo`**: Repository name
- **`workflow-name`**: Workflow identifier (short form)
- **`path/to/workflow.md`**: Full path to workflow file (explicit form)
- **`version`**: Optional reference (tag, branch, or commit SHA)

### Short Form

The short form automatically adds the `workflows/` prefix and `.md` extension:

```bash
# Short form
gh aw add githubnext/agentics/ci-doctor@v1.0.0

# Resolves to
# githubnext/agentics/workflows/ci-doctor.md@v1.0.0
```

### Explicit Form

The explicit form requires the full path with `.md` extension:

```bash
# Explicit form with workflows prefix
gh aw add githubnext/agentics/workflows/ci-doctor.md@v1.0.0

# Explicit form with custom path
gh aw add githubnext/agentics/custom/path/workflow.md@main
```

### Examples

**Short form with version:**
```bash
gh aw add owner/repo/workflow@v1.0.0
# → workflows/workflow.md@v1.0.0
```

**Short form without version:**
```bash
gh aw add owner/repo/workflow
# → workflows/workflow.md (default branch)
```

**Explicit path with version:**
```bash
gh aw add owner/repo/workflows/ci-doctor.md@v1.0.0
# → workflows/ci-doctor.md@v1.0.0
```

**Nested path with branch:**
```bash
gh aw add owner/repo/path/to/workflow.md@main
# → path/to/workflow.md@main
```

**Multiple workflow parts:**
```bash
gh aw add githubnext/agentics/workflows/weekly-research.md@main
# → workflows/weekly-research.md@main
```

### Validation Rules

- **Minimum parts**: Must have at least 3 parts (owner/repo/workflow-name)
- **Owner and repo required**: Cannot be empty
- **Valid GitHub identifiers**: Same rules as RepoSpec
- **Short form**: 3 parts without `.md` extension automatically adds `workflows/` prefix and `.md` extension
- **Explicit form**: 4+ parts or any path with `.md` extension must include the `.md` extension
- **Version optional**: Can be tag, branch, or commit SHA

## Source Specification (SourceSpec)

Source specifications are used in workflow frontmatter to track the origin of imported workflows.

### Format

```yaml
source: "owner/repo/path/to/workflow.md[@ref]"
```

### Components

- **`owner`**: GitHub username or organization
- **`repo`**: Repository name
- **`path/to/workflow.md`**: Full path to workflow file
- **`ref`**: Optional reference (tag, branch, or commit SHA)

### Examples

**Full source with tag:**
```yaml
source: "githubnext/agentics/workflows/ci-doctor.md@v1.0.0"
```

**Source with branch:**
```yaml
source: "githubnext/agentics/workflows/ci-doctor.md@main"
```

**Source without ref (default branch):**
```yaml
source: "githubnext/agentics/workflows/ci-doctor.md"
```

**Nested path with version:**
```yaml
source: "owner/repo/path/to/workflow.md@v2.0.0"
```

### Validation Rules

- **Minimum parts**: Must have at least 3 parts (owner/repo/path)
- **Path required**: Cannot be empty
- **Ref optional**: Can be tag, branch, or commit SHA

## Version References

All specification types support these version reference formats:

### Semantic Version Tags

Version tags following semantic versioning (with or without `v` prefix):

```bash
v1.0.0
v1.2.3
1.0.0
v2.0.0-beta
```

Used for:
- Stable releases
- Version-based updates (respects semantic versioning)

### Branch Names

Branch names for tracking specific development lines:

```bash
main
develop
feature/new-feature
```

Used for:
- Latest changes on a specific branch
- Continuous updates from a branch

### Commit SHAs

Full 40-character commit SHA for precise version pinning:

```bash
abc123def456789012345678901234567890abcdef
1234567890abcdef1234567890abcdef12345678
```

Used for:
- Exact version pinning
- Reproducible workflows
- Audit trail

### No Version

Omitting the version reference uses the repository's default branch:

```bash
owner/repo/workflow
```

Used for:
- Always getting latest from default branch
- Quick testing and development

## CLI Command Usage

### Add Command

```bash
# Add with short form
gh aw add githubnext/agentics/ci-doctor

# Add with version
gh aw add githubnext/agentics/ci-doctor@v1.0.0

# Add with explicit path
gh aw add githubnext/agentics/workflows/ci-doctor.md@main
```

### Update Command

The update command uses the `source` field from workflow frontmatter:

```bash
# Update all workflows with source field
gh aw update

# Update specific workflow
gh aw update ci-doctor

# Update with major version bumps allowed
gh aw update ci-doctor --major
```

### Remove Command

```bash
# Remove by workflow name
gh aw remove ci-doctor
```

## Common Patterns

### Pinning to Specific Version

```bash
# Pin to exact semantic version
gh aw add githubnext/agentics/ci-doctor@v1.0.0
```

Result in frontmatter:
```yaml
source: "githubnext/agentics/workflows/ci-doctor.md@abc123def456789012345678901234567890abcdef"
```

### Tracking a Branch

```bash
# Track main branch
gh aw add githubnext/agentics/ci-doctor@main
```

Result in frontmatter:
```yaml
source: "githubnext/agentics/workflows/ci-doctor.md@main"
```

### Latest Default Branch

```bash
# Use default branch (no version specified)
gh aw add githubnext/agentics/ci-doctor
```

Result in frontmatter:
```yaml
source: "githubnext/agentics/workflows/ci-doctor.md@abc123def456789012345678901234567890abcdef"
```

## Error Messages

### Invalid Repository Format

```
Error: repository must be in format 'org/repo'
```

Fix: Ensure repository includes both owner and repo with a slash separator.

### Invalid Workflow Format

```
Error: workflow specification must be in format 'owner/repo/workflow-name[@version]'
```

Fix: Include owner, repo, and workflow name/path.

### Missing Extension

```
Error: workflow specification with path must end with '.md' extension: workflows/ci-doctor
```

Fix: Add `.md` extension when using explicit paths.

### Invalid GitHub Identifier

```
Error: invalid workflow specification: 'owner-/repo' does not look like a valid GitHub repository
```

Fix: Ensure owner and repo names don't start or end with hyphens.

## Related Documentation

- [CLI Commands](/gh-aw/tools/cli/) - Full CLI command reference
- [Workflow Structure](/gh-aw/reference/workflow-structure/) - Directory layout and organization
- [Frontmatter Options](/gh-aw/reference/frontmatter/) - Configuration options including source field
- [Include Directives](/gh-aw/reference/include-directives/) - Modularizing workflows
