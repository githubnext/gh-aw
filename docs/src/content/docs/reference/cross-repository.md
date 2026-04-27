---
title: Cross-Repository Operations
description: Configure workflows to access, modify, and operate across multiple GitHub repositories using checkout, target-repo, and allowed-repos settings
sidebar:
  order: 850
---

Cross-repository operations enable workflows to access code from multiple repositories and create resources (issues, PRs, comments) in external repositories. This page documents all declarative frontmatter features for cross-repository workflows.

Cross-repository features fall into three categories:

1. **Cross-Repository Checkout** - Check out code from other repositories
2. **Cross-Repository Reading** - Read issues, pull requests and other information from other repositories
3. **Cross-Repository Safe Outputs** - Create issues, PRs, comments, and other resources in external repositories using `target-repo` and `allowed-repos` in safe outputs

All require additional authentication.

## Cross-Repository Checkout (`checkout:`)

The `checkout:` frontmatter field controls how `actions/checkout` is invoked in the agent job. Use it to check out one or more repositories, override fetch depth or sparse-checkout settings, fetch additional refs (e.g., all open PR branches), or disable checkout entirely with `checkout: false`.

For multi-repository workflows, list multiple entries to clone several repos into the workspace. Mark the agent's primary target with `current: true` when working from a central repository that targets a different repo.

```yaml wrap
checkout:
  - fetch-depth: 0                 # checkout this repository with full history
    fetch: ["refs/pulls/open/*"]   # fetch all open PR branches after checkout
  - repository: owner/other-repo   # another repository to check out
    path: ./libs/other             # path within workspace to check out to
    github-token: ${{ secrets.CROSS_REPO_PAT }} # additional auth for cross-repo access
```

See [GitHub Repository Checkout](/gh-aw/reference/checkout/) for the full configuration reference, including fetch options, sparse checkout, merging rules, and examples.

## Cross-Repository Reading

The [GitHub Tools](/gh-aw/reference/github-tools/) are used to read information such as issues and pull requests from repositories. By default, these tools can access the current repository and all public repositories (if permitted by the network firewall). This set can be further restricted by using [GitHub Repository Access Restrictions](/gh-aw/reference/github-tools/#github-repository-access-restrictions-toolsgithuballowed-repos).

To read from other private repositories, you must configure additional authorization. Configure a PAT or GitHub App in your GitHub Tools configuration:

```yaml wrap
tools:
  github:
    toolsets: [repos, issues, pull_requests]
    github-token: ${{ secrets.CROSS_REPO_PAT }}
```

This enables operations like:

- Reading files and searching code in external repositories dynamically, even if the repository is not checked out
- Querying issues and pull requests from other repos
- Accessing commits, releases, and workflow runs across repositories
- Reading organization-level information

See [Additional Authentication for GitHub Tools](/gh-aw/reference/github-tools/#additional-authentication-for-github-tools) for full details on creating a PAT, using a GitHub App, or using the magic secret `GH_AW_GITHUB_MCP_SERVER_TOKEN`.

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

### Wildcard Target Repository (`target-repo: "*"`)

Set `target-repo: "*"` to allow the agent to dynamically target any repository at runtime. When configured, the agent receives a `repo` parameter in its tool call where it supplies the target repository in `owner/repo` format:

```yaml wrap
safe-outputs:
  github-token: ${{ secrets.CROSS_REPO_PAT }}
  create-issue:
    target-repo: "*"
    title-prefix: "[component] "
```

Use this when the target repository is not known at workflow authoring time — for example, when building a workflow that routes issues to different repositories based on labels or content.

:::caution
The following safe-output types do **not** support `target-repo: "*"`: `create-pull-request-review-comment`, `reply-to-pull-request-review-comment`, `submit-pull-request-review`, `create-agent-session`, and `manage-project-items`. Use an explicit `owner/repo` value or `allowed-repos` for these types.
:::

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

### Checkout Requirement for `push-to-pull-request-branch`

Unlike other safe output types, `push-to-pull-request-branch` with `target-repo` requires the target repository to be **checked out into the workflow workspace** using the `checkout:` frontmatter field with a `path:` specified. Without a checkout, the agent has no local git history to create and push a patch from.

See the [Scheduled Push to Pull-Request Branch](#example-scheduled-push-to-pull-request-branch) example and the [Push to PR Branch cross-repo usage](/gh-aw/reference/safe-outputs-pull-requests/#cross-repo-usage) documentation for a complete setup.

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

- [GitHub Repository Checkout](/gh-aw/reference/checkout/) - Full checkout configuration reference
- [MultiRepoOps Pattern](/gh-aw/patterns/multi-repo-ops/) - Cross-repository workflow pattern
- [CentralRepoOps Pattern](/gh-aw/patterns/central-repo-ops/) - Central control plane pattern
- [GitHub Tools Reference](/gh-aw/reference/github-tools/) - Complete GitHub Tools configuration
- [Safe Outputs Reference](/gh-aw/reference/safe-outputs/) - Complete safe output configuration
- [Authentication Reference](/gh-aw/reference/auth/) - PAT and GitHub App setup
- [Multi-Repository Examples](/gh-aw/examples/multi-repo/) - Complete working examples
