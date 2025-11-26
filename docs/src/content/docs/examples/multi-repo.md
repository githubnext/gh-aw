---
title: Multi-Repository Examples
description: Complete examples for managing workflows across multiple GitHub repositories, including feature synchronization, cross-repo tracking, and organization-wide updates.
---

Multi-repository operations enable coordinating work across multiple GitHub repositories while maintaining security and proper access controls. These examples demonstrate common patterns for cross-repo workflows.

## Featured Examples

### [Feature Synchronization](/gh-aw/examples/multi-repo/feature-sync/)

Synchronize code changes from a main repository to sub-repositories or downstream services. Ideal for maintaining shared components, library updates, or multi-platform deployments.

**Key Features:**
- Automated pull request creation in target repositories
- Change detection with path filters
- Release-based and scheduled synchronization
- Bidirectional sync with conflict detection

**Use Cases:**
- Monorepo alternatives
- Shared component libraries
- Multi-platform deployment
- Fork maintenance

### [Cross-Repository Issue Tracking](/gh-aw/examples/multi-repo/issue-tracking/)

Centralize issue tracking across multiple component repositories by creating tracking issues in a central repository when issues are opened in component repos.

**Key Features:**
- Automatic tracking issue creation
- Status synchronization
- Multi-component coordination
- External dependency tracking

**Use Cases:**
- Component-based architecture visibility
- Multi-team coordination
- Cross-project initiatives
- Upstream dependency tracking

## Getting Started

All multi-repo workflows require proper authentication:

### Personal Access Token Setup

```bash
# Create PAT with required permissions
gh auth token

# Store as repository or organization secret
gh secret set CROSS_REPO_PAT --body "ghp_your_token_here"
```

**Required Permissions:**

The PAT needs permissions **only on target repositories** where you want to create resources, not on the source repository where the workflow runs.

- `repo` (for private target repositories)
- `contents: write` (for creating commits in target repos)
- `issues: write` (for creating issues in target repos)
- `pull-requests: write` (for creating PRs in target repos)

:::tip
**Security Best Practice**: If you only need to read from one repo and write to another, scope your PAT to have read access on the source and write access only on target repositories. Use separate tokens for different operations when possible.
:::

### GitHub App Configuration

For enhanced security, use GitHub Apps for automatic token minting and revocation:

```yaml wrap
safe-outputs:
  app:
    app-id: ${{ vars.APP_ID }}
    private-key: ${{ secrets.APP_PRIVATE_KEY }}
    owner: "my-org"
    repositories: ["repo1", "repo2", "repo3"]
  create-issue:
    target-repo: "my-org/repo1"
```

**Benefits**: GitHub App tokens are minted on-demand, automatically revoked after job completion, and provide better security than long-lived PATs.

See [Safe Outputs Reference](/gh-aw/reference/safe-outputs/#github-app-token-app) for complete GitHub App configuration.

## Common Patterns

### Hub-and-Spoke Architecture

Central repository aggregates information from multiple component repositories:

```
Component Repo A ──┐
Component Repo B ──┼──> Central Tracker
Component Repo C ──┘
```

### Upstream-to-Downstream Sync

Main repository propagates changes to downstream repositories:

```
Main Repo ──> Sub-Repo Alpha
          ──> Sub-Repo Beta
          ──> Sub-Repo Gamma
```

### Organization-Wide Coordination

Single workflow creates issues across multiple repositories:

```
Control Workflow ──> Repo 1 (tracking issue)
                 ──> Repo 2 (tracking issue)
                 ──> Repo 3 (tracking issue)
                 ──> ... (up to max limit)
```

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
| `create-agent-task` | ✅ | Create tasks in target repos |
| `update-release` | ✅ | Update release notes across repos |

**Configuration Example:**

```yaml wrap
safe-outputs:
  github-token: ${{ secrets.CROSS_REPO_PAT }}
  create-issue:
    target-repo: "org/tracking-repo"  # Cross-repo: creates in tracking-repo
    title-prefix: "[component] "
  add-comment:
    # No target-repo: operates on current repository
```

See [Safe Outputs Reference](/gh-aw/reference/safe-outputs/) for complete configuration options.

## GitHub API Tools for Multi-Repo Access

Enable GitHub toolsets to allow agents to query multiple repositories:

```yaml wrap
tools:
  github:
    toolsets: [repos, issues, pull_requests, actions]
```

**Available Operations:**
- **repos**: Read files, search code, list commits, get releases
- **issues**: List and search issues across repositories
- **pull_requests**: List and search PRs across repositories
- **actions**: Access workflow runs and artifacts

## Best Practices

### Authentication and Security

1. Use GitHub Apps for automatic token revocation
2. Scope PATs minimally to required repositories
3. Rotate tokens regularly
4. Store tokens as GitHub secrets (never in code)

### Workflow Design

1. Set appropriate `max` limits on safe outputs
2. Use meaningful title prefixes for tracking
3. Apply consistent labels across repositories
4. Include clear documentation in created items

### Error Handling

1. Validate repository access before operations
2. Handle rate limits appropriately
3. Provide fallback for permission failures
4. Monitor workflow execution across repositories

### Testing

1. Test with public repositories first
2. Pilot with small repository subset
3. Verify path mappings and configurations
4. Monitor costs and rate limits

## Advanced Topics

### Private Repository Access

When working with private repositories:
- Ensure PAT owner has repository access
- Install GitHub Apps in target organizations
- Configure repository lists explicitly
- Test permissions before full rollout

### Deterministic Workflows

For direct repository access, use custom engine with `actions/checkout`:

```yaml wrap
engine:
  id: custom
  steps:
    - name: Checkout main repo
      uses: actions/checkout@v4
      with:
        path: main-repo
    
    - name: Checkout secondary repo
      uses: actions/checkout@v4
      with:
        repository: org/secondary-repo
        token: ${{ secrets.CROSS_REPO_PAT }}
        path: secondary-repo
```

### Organization-Level Operations

For organization-wide workflows:
- Use organization-level secrets
- Configure GitHub Apps at organization level
- Plan phased rollouts
- Provide clear communication

## Complete Guide

For comprehensive documentation on the MultiRepoOps design pattern, see:

[MultiRepoOps Design Pattern](/gh-aw/guides/multirepoops/)

## Related Documentation

- [Safe Outputs Reference](/gh-aw/reference/safe-outputs/) - Configuration options
- [GitHub Tools](/gh-aw/reference/tools/#github-tools-github) - API access configuration
- [Security Best Practices](/gh-aw/guides/security/) - Authentication and security
- [Packaging & Distribution](/gh-aw/guides/packaging-imports/) - Sharing workflows
