---
title: MultiRepoOps
description: Coordinate agentic workflows across multiple GitHub repositories with automated issue tracking, feature synchronization, and organization-wide enforcement
sidebar:
  badge: { text: 'Advanced', variant: 'caution' }
---

MultiRepoOps extends operational automation patterns (IssueOps, ChatOps, etc.) across multiple GitHub repositories. Using cross-repository safe outputs and secure authentication, MultiRepoOps enables coordinating work between related projects-creating tracking issues in central repos, synchronizing features to sub-repositories, and enforcing organization-wide policies-all through AI-powered workflows.

## When to Use MultiRepoOps

Use MultiRepoOps for feature synchronization (main repo to sub-repos), hub-and-spoke issue tracking (components → central tracker), org-wide enforcement (security patches, policy rollouts), and upstream/downstream feature sync.

## How It Works

MultiRepoOps workflows use the `target-repo` parameter on safe outputs to create issues, pull requests, and comments in external repositories. Combined with GitHub API toolsets for querying remote repos and proper authentication (PAT or GitHub App tokens), workflows can coordinate complex multi-repository operations automatically.

```aw wrap
---
on:
  issues:
    types: [opened, labeled]
permissions:
  contents: read
  actions: read
safe-outputs:
  github-token: ${{ secrets.GH_AW_CROSS_REPO_PAT }}
  create-issue:
    target-repo: "org/tracking-repo"
    title-prefix: "[component-a] "
    labels: [tracking, multi-repo]
---

# Cross-Repo Issue Tracker

When issues are created in component repositories, automatically create tracking issues in the central coordination repo.

Analyze the issue and create a tracking issue that:
- Links back to the original component issue
- Summarizes the problem and impact
- Tags relevant teams across the organization
- Provides context for cross-component coordination
```

## Authentication for Cross-Repo Access

Cross-repository operations require authentication beyond the default `GITHUB_TOKEN`, which is scoped to the current repository only.

### Personal Access Token (PAT)

Configure a Personal Access Token with access to target repositories:

```yaml wrap
safe-outputs:
  github-token: ${{ secrets.GH_AW_CROSS_REPO_PAT }}
  create-issue:
    target-repo: "org/tracking-repo"
```

The PAT needs permissions only on target repositories — `contents: write`, `issues: write`, or `pull-requests: write` depending on operations (not on the source repo).

> [!TIP]
> Security Best Practice
> If you only need to read from one repo and write to another, scope your PAT to have read access on the source and write access only on target repositories.

### GitHub App Installation Token

For enhanced security, use GitHub Apps with automatic token revocation. GitHub App tokens provide per-job minting, automatic revocation after job completion, fine-grained permissions, and better attribution than long-lived PATs.

See [Using a GitHub App for Authentication](/gh-aw/reference/auth/#using-a-github-app-for-authentication) for complete configuration including specific repository scoping and org-wide access.

## Common MultiRepoOps Patterns

Three topologies cover most use cases:

| Pattern | Description |
|---------|-------------|
| **Hub-and-spoke** | Each component workflow creates tracking issues in a central repo via `target-repo` |
| **Upstream-to-downstream** | Main repo propagates changes using `create-pull-request` with `target-repo` per downstream |
| **Org-wide broadcast** | Single workflow creates issues in many repos up to the configured `max` limit |

## Cross-Repository Safe Outputs

Most safe output types support the `target-repo` parameter for cross-repository operations. **Without `target-repo`, these safe outputs operate on the repository where the workflow is running.**

| Safe Output | Cross-Repo Support | Example Use Case |
|-------------|-------------------|------------------|
| `create-issue` | ✅ | Create tracking issues in central repo |
| `add-comment` | ✅ | Comment on issues in other repos |
| `update-issue` | ✅ | Update issue status across repos |
| `add-labels` | ✅ | Label issues in target repos |
| `create-pull-request` | ✅ | Create PRs in downstream repos |
| `create-discussion` | ✅ | Create discussions in any repo |
| `create-agent-session` | ✅ | Create tasks in target repos |
| `update-release` | ✅ | Update release notes across repos |

## Teaching Agents Multi-Repo Access

Enable GitHub toolsets to allow agents to query multiple repositories:

```yaml wrap
tools:
  github:
    toolsets: [repos, issues, pull_requests, actions]
    github-token: ${{ secrets.CROSS_REPO_PAT }}  # Required for cross-repo reading
```

> [!IMPORTANT]
> When reading from repositories other than the workflow's repository, you must configure additional authentication. The default `GITHUB_TOKEN` only has access to the current repository. Use a PAT, GitHub App token, or the magic secret `GH_AW_GITHUB_MCP_SERVER_TOKEN`. See [GitHub Tools Reference](/gh-aw/reference/cross-repository/#cross-repository-reading) for details.

Agent instructions can reference remote repositories:

```markdown
Search for open issues in org/upstream-repo related to authentication.
Check the latest release notes from org/dependency-repo.
Compare code structure between this repo and org/reference-repo.
```

## Deterministic Multi-Repo Workflows

For direct repository access without agent involvement, use an AI engine with custom steps:

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
      token: ${{ secrets.GH_AW_CROSS_REPO_PAT }}
      path: secondary-repo

  - name: Compare and sync
    run: |
      # Deterministic sync logic
      rsync -av main-repo/shared/ secondary-repo/shared/
      cd secondary-repo
      git add .
      git commit -m "Sync from main repo"
      git push
---

# Deterministic Feature Sync

Workflow that directly checks out multiple repos and synchronizes files.
```

## Example Workflows

Explore detailed MultiRepoOps examples:

- **[Feature Synchronization](/gh-aw/examples/multi-repo/feature-sync/)** - Sync code changes from main repo to sub-repositories
- **[Cross-Repo Issue Tracking](/gh-aw/examples/multi-repo/issue-tracking/)** - Hub-and-spoke tracking architecture

## Best Practices

Use GitHub Apps over PATs for automatic token revocation; scope tokens minimally to target repositories. Set appropriate `max` limits and consistent label/prefix conventions. Test against public repositories first before rolling out to private or org-wide targets.

## Related

- [IssueOps](/gh-aw/patterns/issue-ops/) — Single-repo issue automation
- [ChatOps](/gh-aw/patterns/chat-ops/) — Command-driven workflows
- [Orchestration](/gh-aw/patterns/orchestration/) — Multi-issue initiative coordination
- [Cross-Repository Operations](/gh-aw/reference/cross-repository/) — Checkout and `target-repo` configuration
- [Safe Outputs Reference](/gh-aw/reference/safe-outputs/) — Complete safe output configuration
- [GitHub Tools](/gh-aw/reference/github-tools/) — GitHub API toolsets
- [Reusing Workflows](/gh-aw/guides/packaging-imports/) — Sharing workflows across repos
