---
title: Cross-Repository Operations
description: Configure workflows to access, modify, and operate across multiple GitHub repositories using checkout, target-repo, and allowed-repos settings
sidebar:
  order: 850
---

Cross-repository operations enable workflows to access code from multiple repositories and create resources (issues, PRs, comments) in external repositories. This page documents all declarative frontmatter features for cross-repository workflows.

## Overview

Cross-repository features fall into two categories:

1. **Code access** - Check out code from multiple repositories into the workflow workspace using the `checkout:` frontmatter field
2. **Resource creation** - Create issues, PRs, comments, and other resources in external repositories using `target-repo` and `allowed-repos` in safe outputs

Both require authentication beyond the default `GITHUB_TOKEN`, which is scoped to the current repository only.

## Repository Checkout (`checkout:`)

The `checkout:` frontmatter field controls how `actions/checkout` is invoked in the agent job. Configure custom checkout settings or check out multiple repositories.

### Single Repository Configuration

Override default checkout settings for the main repository:

```yaml wrap
checkout:
  fetch-depth: 0                           # Full git history
  github-token: ${{ secrets.MY_TOKEN }}    # Custom authentication
```

### Multiple Repository Checkout

Check out additional repositories alongside the main repository:

```yaml wrap
checkout:
  - path: .
    fetch-depth: 0
  - repository: owner/other-repo
    path: ./libs/other
    ref: main
    github-token: ${{ secrets.CROSS_REPO_PAT }}
```

### Checkout Configuration Options

| Field | Type | Description |
|-------|------|-------------|
| `repository` | string | Repository in `owner/repo` format. Defaults to the current repository. |
| `ref` | string | Branch, tag, or SHA to checkout. Defaults to the triggering ref. |
| `path` | string | Path within `GITHUB_WORKSPACE` to place the checkout. Defaults to workspace root. |
| `github-token` | string | Token for authentication. Use `${{ secrets.MY_TOKEN }}` syntax. |
| `fetch-depth` | integer | Commits to fetch. `0` = full history, `1` = shallow clone (default). |
| `sparse-checkout` | string | Newline-separated patterns for sparse checkout (e.g., `.github/\nsrc/`). |
| `submodules` | string/bool | Submodule handling: `"recursive"`, `"true"`, or `"false"`. |
| `lfs` | boolean | Download Git LFS objects. |

> [!TIP]
> Credentials are always removed after checkout (`persist-credentials: false` is enforced) to prevent credential exfiltration by agents.

### Multiple Checkout Merging

When multiple configurations target the same path and repository:

- **Fetch depth**: Deepest value wins (`0` = full history always takes precedence)
- **Sparse patterns**: Merged (union of all patterns)
- **LFS**: OR-ed (if any config enables `lfs`, the merged configuration enables it)
- **Submodules**: First non-empty value wins for each `(repository, path)`; once set, later values are ignored
- **Ref/Token**: First-seen wins

### Example: Monorepo Development

```aw wrap
---
on:
  pull_request:
    types: [opened, synchronize]

checkout:
  - path: .
    fetch-depth: 0
  - repository: org/shared-libs
    path: ./libs/shared
    ref: main
    github-token: ${{ secrets.LIBS_PAT }}
  - repository: org/config-repo
    path: ./config
    sparse-checkout: |
      defaults/
      overrides/

permissions:
  contents: read
  pull-requests: read
---

# Cross-Repo PR Analysis

Analyze this PR considering shared library compatibility and configuration standards.

Check compatibility with shared libraries in `./libs/shared` and verify configuration against standards in `./config`.
```

## Cross-Repository Safe Outputs

Most safe output types support creating resources in external repositories using `target-repo` and `allowed-repos` parameters.

### Target Repository (`target-repo`)

Specify a single target repository for resource creation:

```yaml wrap
safe-outputs:
  github-token: ${{ secrets.CROSS_REPO_PAT }}
  create-issue:
    target-repo: "org/tracking-repo"
    title-prefix: "[component] "
```

Without `target-repo`, safe outputs operate on the repository where the workflow is running.

### Allowed Repositories (`allowed-repos`)

Allow the agent to dynamically select from multiple repositories:

```yaml wrap
safe-outputs:
  github-token: ${{ secrets.CROSS_REPO_PAT }}
  create-issue:
    target-repo: "org/default-repo"
    allowed-repos: ["org/repo-a", "org/repo-b", "org/repo-c"]
    title-prefix: "[cross-repo] "
```

When `allowed-repos` is specified:

- Agent can include a `repo` field in output to select which repository
- Target repository (from `target-repo` or current repo) is always implicitly allowed
- Creates a union of allowed destinations

### Example: Hub-and-Spoke Tracking

```aw wrap
---
on:
  issues:
    types: [opened, labeled]

permissions:
  contents: read
  issues: read

safe-outputs:
  github-token: ${{ secrets.CROSS_REPO_PAT }}
  create-issue:
    target-repo: "org/central-tracker"
    title-prefix: "[component-a] "
    labels: [tracking, multi-repo]
    max: 1
---

# Cross-Repository Issue Tracker

When issues are created in this component repository, create tracking issues in the central coordination repo.

Analyze the issue and create a tracking issue that:
- Links back to the original component issue
- Summarizes the problem and impact
- Tags relevant teams for coordination
```

## Authentication

Cross-repository operations require authentication with access to target repositories.

### Personal Access Token (PAT)

Create a fine-grained PAT with access to target repositories:

```yaml wrap
safe-outputs:
  github-token: ${{ secrets.CROSS_REPO_PAT }}
  create-issue:
    target-repo: "org/target-repo"
```

**Required permissions** (on target repositories only):

| Operation | Permissions |
|-----------|-------------|
| Create/update issues | `issues: write` |
| Create PRs | `contents: write`, `pull-requests: write` |
| Add comments | `issues: write` or `pull-requests: write` |
| Checkout code | `contents: read` |

> [!TIP]
> **Security Best Practice**: Scope PATs to have read access on source repositories and write access only on target repositories. Use separate tokens for different operations when possible.

### GitHub App Installation Token

For enhanced security, use GitHub Apps. See [Authentication Reference](/gh-aw/reference/auth/#using-a-github-app-for-authentication) for complete configuration examples.

## Deterministic Multi-Repo Workflows

For direct repository access without agent involvement, use custom steps with `actions/checkout`:

```aw wrap
---
engine:
  id: claude

steps:
  - name: Checkout main repo
    uses: actions/checkout@v5
    with:
      path: main-repo

  - name: Checkout secondary repo
    uses: actions/checkout@v5
    with:
      repository: org/secondary-repo
      token: ${{ secrets.CROSS_REPO_PAT }}
      path: secondary-repo

permissions:
  contents: read
---

# Compare Repositories

Compare code structure between main-repo and secondary-repo.
```

This approach provides full control over checkout timing and configuration.

## Related Documentation

- [MultiRepoOps Pattern](/gh-aw/patterns/multi-repo-ops/) - Cross-repository workflow pattern
- [CentralRepoOps Pattern](/gh-aw/patterns/central-repo-ops/) - Central control plane pattern
- [Safe Outputs Reference](/gh-aw/reference/safe-outputs/) - Complete safe output configuration
- [Authentication Reference](/gh-aw/reference/auth/) - PAT and GitHub App setup
- [Multi-Repository Examples](/gh-aw/examples/multi-repo/) - Complete working examples
