---
title: Cross-Repository Operations
description: Configure workflows to access, modify, and operate across multiple GitHub repositories using checkout, target-repo, and allowed-repos settings
sidebar:
  order: 850
---

Cross-repository operations enable workflows to access code from multiple repositories and create resources (issues, PRs, comments) in external repositories. This page documents all declarative frontmatter features for cross-repository workflows.

## Overview

Cross-repository features fall into three categories:

1. **Code access** - Check out code from multiple repositories into the workflow workspace using the `checkout:` frontmatter field
2. **GitHub tools** - Read information from other repositories using GitHub Tools with additional authentication
3. **Safe outputs** - Create issues, PRs, comments, and other resources in external repositories using `target-repo` and `allowed-repos` in safe outputs

All require authentication beyond the default `GITHUB_TOKEN`, which is scoped to the current repository only.

## Cross-repository Checkout (`checkout:`)

The `checkout:` frontmatter field controls how `actions/checkout` is invoked in the agent job. Configure custom checkout settings or check out multiple repositories.

If only a the current repository, you can use `checkout:` to override default checkout settings (e.g., fetch depth, sparse checkout) without needing to define a custom job:

```yaml wrap
checkout:
  fetch-depth: 0                              # Full git history
  github-token: ${{ secrets.MY_TOKEN }}        # Custom authentication
```

Or use GitHub App authentication:

```yaml wrap
checkout:
  fetch-depth: 0
  github-app:
    app-id: ${{ vars.APP_ID }}
    private-key: ${{ secrets.APP_PRIVATE_KEY }}
```

You can also use `checkout:` to check out additional repositories alongside the main repository:

```yaml wrap
checkout:
  - fetch-depth: 0
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
| `app` | object | GitHub App credentials (`app-id`, `private-key`, optional `owner`, `repositories`). Mutually exclusive with `github-token`. |
| `fetch-depth` | integer | Commits to fetch. `0` = full history, `1` = shallow clone (default). |
| `fetch` | string \| string[] | Additional Git refs to fetch after checkout. See [Fetching Additional Refs](#fetching-additional-refs). |
| `sparse-checkout` | string | Newline-separated patterns for sparse checkout (e.g., `.github/\nsrc/`). |
| `submodules` | string/bool | Submodule handling: `"recursive"`, `"true"`, or `"false"`. |
| `lfs` | boolean | Download Git LFS objects. |
| `current` | boolean | Marks this checkout as the primary working repository. The agent uses this as the default target for all GitHub operations. Only one checkout may set `current: true`; the compiler rejects workflows where multiple checkouts enable it. |

### Fetching Additional Refs

By default, `actions/checkout` performs a shallow clone (`fetch-depth: 1`) of a single ref. For workflows that need to work with other branches — for example, a scheduled workflow that must push changes to open pull-request branches — use the `fetch:` option to retrieve additional refs after the checkout step.

A dedicated git fetch step is emitted after the `actions/checkout` step. Authentication re-uses the checkout token (or falls back to `github.token`) via a transient `http.extraheader` credential — no credentials are persisted to disk, consistent with the enforced `persist-credentials: false` policy.

| Value | Description |
|-------|-------------|
| `"*"` | All remote branches. |
| `"refs/pulls/open/*"` | All open pull-request head refs (GH-AW shorthand). |
| `"main"` | A specific branch name. |
| `"feature/*"` | A glob pattern matching branch names. |

```yaml wrap
checkout:
  - fetch: ["*"]                 # fetch all branches (default checkout)
    fetch-depth: 0               # fetch full history to ensure we can see all commits and PR details
```

```yaml wrap
checkout:
  - repository: githubnext/gh-aw-side-repo
    github-token: ${{ secrets.GH_AW_SIDE_REPO_PAT }}
    fetch: ["refs/pulls/open/*"]      # fetch all open PR refs after checkout
    fetch-depth: 0               # fetch full history to ensure we can see all commits and PR details
```

```yaml wrap
checkout:
  - repository: org/target-repo
    github-token: ${{ secrets.CROSS_REPO_PAT }}
    fetch: ["main", "feature/*"] # fetch specific branches
    fetch-depth: 0               # fetch full history to ensure we can see all commits and PR details
```

:::note
If a branch you need is not available after checkout and is not covered by a `fetch:` pattern, and you're in a private or internal repo, then the agent cannot access its Git history except inefficiently, file by file, via the GitHub MCP. For private repositories, it will be unable to fetch or explore additional branches. If the branch is required and unavailable, configure the appropriate pattern in `fetch:` (e.g., `fetch: ["*"]` for all branches, or `fetch: ["refs/pulls/open/*"]` for PR branches) and recompile the workflow.
:::

### Checkout Merging

Multiple `checkout:` configurations can target the same path and repository. This is useful for monorepos where different parts of the repository must be merged into the same workspace directory with different settings (e.g., sparse checkout for some paths, full checkout for others).

When multiple `checkout:` entries target the same repository and path, their configurations are merged with the following rules:

- **Fetch depth**: Deepest value wins (`0` = full history always takes precedence)
- **Fetch refs**: Merged (union of all patterns; duplicates are removed)
- **Sparse patterns**: Merged (union of all patterns)
- **LFS**: OR-ed (if any config enables `lfs`, the merged configuration enables it)
- **Submodules**: First non-empty value wins for each `(repository, path)`; once set, later values are ignored
- **Ref/Token/App**: First-seen wins

### Marking a Primary Repository (`current: true`)

When a workflow running from a central repository targets a different repository, use `current: true` to tell the agent which repository to treat as its primary working target. The agent uses this as the default for all GitHub operations (creating issues, opening PRs, reading content) unless the prompt instructs otherwise. When omitted, the agent defaults to the repository where the workflow is running.

```yaml wrap
checkout:
  - repository: org/target-repo
    path: ./target
    github-token: ${{ secrets.CROSS_REPO_PAT }}
    current: true                                    # agent's primary target
```

## GitHub Tools - Reading Other Repositories

When using [GitHub Tools](/gh-aw/reference/github-tools/) to read information from repositories other than the one where the workflow is running, you must configure additional authorization. The default `GITHUB_TOKEN` is scoped to the current repository only and cannot access other repositories.

Configure the additional authentication in your GitHub Tools configuration. For example, using a PAT:

```yaml wrap
tools:
  github:
    toolsets: [repos, issues, pull_requests]
    github-token: ${{ secrets.CROSS_REPO_PAT }}
```


See [GitHub Tools Reference](/gh-aw/reference/github-tools/#cross-repository-reading) for complete details on configuring cross-repository read access for GitHub Tools.

This authentication is for **reading** information from GitHub. Authorization for **writing** to other repositories (creating issues, PRs, comments) is configured separately, see below.

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

## Examples

### Example: Monorepo Development

This uses multiple `checkout:` entries to check out different parts of the same repository with different settings:

```aw wrap
---
on:
  pull_request:
    types: [opened, synchronize]

checkout:
  - fetch-depth: 0
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

### Example: Hub-and-Spoke Tracking

This creates issues in a central tracking repository when issues are opened in component repositories:

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

### Example: Cross-Repository Analysis

This checks out multiple repositories and compares code patterns across them:

```aw wrap
---
on:
  issue_comment:
    types: [created]

tools:
  github:
    toolsets: [repos, issues, pull_requests]
    github-token: ${{ secrets.CROSS_REPO_PAT }}

permissions:
  contents: read
  issues: read

safe-outputs:
  github-token: ${{ secrets.CROSS_REPO_WRITE_PAT }}
  add-comment:
    max: 1
---

# Multi-Repository Code Search

Search for similar patterns across org/repo-a, org/repo-b, and org/repo-c.

Analyze how each repository implements authentication and provide a comparison.
```

### Example: Deterministic Multi-Repo Workflows

For direct repository access without agent involvement, use custom steps with `actions/checkout`:

```aw wrap
---
engine:
  id: claude

steps:
  - name: Checkout main repo
    uses: actions/checkout@v6
    with:
      path: main-repo

  - name: Checkout secondary repo
    uses: actions/checkout@v6
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

### Example: Scheduled Push to Pull-Request Branch

A scheduled workflow that automatically pushes changes to open pull-request branches in another repository needs to fetch those branches after checkout. Without `fetch:`, only the default branch (usually `main`) is available.

```aw wrap
---
on:
  schedule:
    - cron: "0 * * * *"

checkout:
  - repository: org/target-repo
    github-token: ${{ secrets.GH_AW_SIDE_REPO_PAT }}
    fetch: ["refs/pulls/open/*"]   # fetch all open PR branches after checkout
    current: true

permissions:
  contents: read

safe-outputs:
  github-token: ${{ secrets.GH_AW_SIDE_REPO_PAT }}
  push-to-pull-request-branch:
    target-repo: "org/target-repo"
---

# Auto-Update PR Branches

Check open pull requests in org/target-repo and apply any pending automated
updates to each PR branch.
```

`fetch: ["refs/pulls/open/*"]` causes a `git fetch` step to run after `actions/checkout`, downloading all open PR head refs into the workspace. The agent can then inspect and modify those branches directly.

## Related Documentation

- [MultiRepoOps Pattern](/gh-aw/patterns/multi-repo-ops/) - Cross-repository workflow pattern
- [CentralRepoOps Pattern](/gh-aw/patterns/central-repo-ops/) - Central control plane pattern
- [GitHub Tools Reference](/gh-aw/reference/github-tools/) - Complete GitHub Tools configuration
- [Safe Outputs Reference](/gh-aw/reference/safe-outputs/) - Complete safe output configuration
- [Authentication Reference](/gh-aw/reference/auth/) - PAT and GitHub App setup
- [Multi-Repository Examples](/gh-aw/examples/multi-repo/) - Complete working examples
