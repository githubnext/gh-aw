---
title: Multi-Repository Operations
description: Learn how to manage workflows across multiple GitHub repositories with agentic workflows, including cross-repo safe outputs, authentication strategies, and remote repository access.
sidebar:
  order: 8
---

Multi-repository operations enable agentic workflows to coordinate work across multiple GitHub repositories. You can create issues, pull requests, and comments in external repositories, fetch data from remote repos, and synchronize features between related projects—all while maintaining secure authentication and proper access controls.

## Use Cases

Multi-repo operations support common organizational workflows:

- **Feature synchronization** - Propagate changes from a main repository to sub-repositories or forks
- **Cross-repo issue tracking** - Create tracking issues in a central repository when issues are opened in component repos
- **Organization-wide enforcement** - Apply security policies, documentation standards, or dependency updates across multiple projects
- **Monorepo coordination** - Manage related packages or services that live in separate repositories
- **Upstream/downstream workflows** - Sync features from upstream dependencies to downstream consumers

## Authentication Strategies

Cross-repository operations require authentication beyond the default `GITHUB_TOKEN`, which is scoped to the current repository only.

### Personal Access Token (PAT)

The most straightforward approach for multi-repo access uses a Personal Access Token with appropriate permissions:

```bash
# Create PAT with required scopes
gh auth token

# Store as repository secret
gh secret set CROSS_REPO_PAT --body "ghp_your_token_here"
```

**Required Permissions:**
- **Repository access**: Grant access to target repositories (public, or select private repos)
- **Permissions**: `contents: write`, `issues: write`, `pull-requests: write` (depending on operations)

Configure in your workflow:

```yaml wrap
safe-outputs:
  github-token: ${{ secrets.CROSS_REPO_PAT }}
  create-issue:
    target-repo: "org/tracking-repo"
```

### GitHub App Installation Token

For enhanced security, use GitHub App installation tokens that auto-revoke after job completion:

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

**Benefits:**
- Automatic token revocation (even on job failure)
- Fine-grained permissions per installation
- Clear audit trail with app attribution
- No personal token required
- Repository scoping for enhanced security

See [Safe Outputs - GitHub App Token](/gh-aw/reference/safe-outputs/#github-app-token-app) for complete configuration.

### Private Repository Access

When working with private repositories:

1. **PAT approach**: Ensure the PAT owner has access to target private repos
2. **GitHub App approach**: Install the app in organizations/repos with private repository access
3. **Repository visibility**: Configure `repositories:` list to include private repos

:::caution[Token Security]
Never commit tokens to version control. Always use GitHub secrets (`${{ secrets.TOKEN_NAME }}`). Limit token permissions to minimum required scope.
:::

## Cross-Repository Safe Outputs

Most safe output types support the `target-repo` parameter for cross-repository operations.

### Creating Issues in External Repos

Create tracking issues in a central repository when events occur in component repos:

```aw wrap
---
on:
  issues:
    types: [opened, labeled]
permissions:
  contents: read
  actions: read
safe-outputs:
  github-token: ${{ secrets.CROSS_REPO_PAT }}
  create-issue:
    target-repo: "org/issue-tracker"
    title-prefix: "[component-a] "
    labels: [from-component-a, auto-tracked]
---

# Cross-Repo Issue Tracker

When an issue is opened in this component repository, create a corresponding
tracking issue in the central issue tracking repository.

Include:
- Link to the original issue
- Component identifier
- Summary of the issue
- Suggested priority based on labels
```

### Cross-Repo Pull Requests

Create pull requests in target repositories:

```aw wrap
---
on:
  workflow_dispatch:
permissions:
  contents: read
  actions: read
safe-outputs:
  github-token: ${{ secrets.CROSS_REPO_PAT }}
  create-pull-request:
    target-repo: "org/downstream-service"
    title-prefix: "[sync] "
    labels: [automated-sync, upstream-update]
    draft: true
---

# Feature Sync to Downstream

Synchronize recent features from this repository to the downstream service.

1. Review recent changes in this repo
2. Identify features that should propagate downstream
3. Generate compatible changes for downstream service
4. Create a draft PR with the synchronized features
```

### Adding Comments Across Repos

Comment on issues or PRs in external repositories:

```aw wrap
---
on:
  issues:
    types: [closed]
permissions:
  contents: read
  actions: read
safe-outputs:
  github-token: ${{ secrets.CROSS_REPO_PAT }}
  add-comment:
    target-repo: "org/main-repo"
    target: 42  # Specific issue number in target repo
---

# Update Parent Issue

When this component issue is closed, add a status update comment
to the parent tracking issue in the main repository.
```

### Supported Cross-Repo Safe Outputs

| Safe Output | Cross-Repo Support | Target Parameter |
|-------------|-------------------|------------------|
| `create-issue` | ✅ | `target-repo` |
| `add-comment` | ✅ | `target-repo` + `target` |
| `update-issue` | ✅ | `target-repo` + `target` |
| `add-labels` | ✅ | `target-repo` + `target` |
| `assign-milestone` | ✅ | `target-repo` + `target` |
| `create-pull-request` | ✅ | `target-repo` |
| `create-pull-request-review-comment` | ✅ | `target-repo` + `target` |
| `create-discussion` | ✅ | `target-repo` |
| `close-discussion` | ✅ | `target-repo` + `target` |
| `create-agent-task` | ✅ | `target-repo` |
| `update-release` | ✅ | `target-repo` |

See [Safe Outputs Reference](/gh-aw/reference/safe-outputs/) for detailed configuration options.

## Teaching Agents Remote Repository Access

Agentic workflows can fetch information from remote repositories using GitHub API tools.

### Using GitHub Toolsets

Enable GitHub API toolsets to allow agents to query remote repositories:

```yaml wrap
tools:
  github:
    toolsets: [repos, issues, pull_requests, actions, discussions]
```

With these toolsets, agents can:
- **repos**: Read repository files, search code, list commits, get releases
- **issues**: List and search issues across repositories
- **pull_requests**: List and search PRs across repositories
- **actions**: Access workflow runs and artifacts
- **discussions**: Read discussions from any repository

### Agent Instructions for Remote Access

Provide clear instructions for agents to access remote repositories:

```aw wrap
---
on: workflow_dispatch
permissions:
  contents: read
tools:
  github:
    toolsets: [repos, issues, actions]
---

# Multi-Repo Status Report

Create a status report summarizing activity across multiple repositories.

**Repositories to check:**
- `org/frontend-app`
- `org/backend-api`
- `org/shared-library`

For each repository:
1. Get the latest release information
2. List open issues with `priority:high` label
3. Check recent workflow run status
4. Summarize any failures or concerns

Create a comprehensive status report as a markdown document.
```

### Code Search Across Repositories

Search code patterns across multiple repositories:

```aw wrap
---
on:
  schedule:
    - cron: "0 9 * * 1"  # Monday 9AM
permissions:
  contents: read
tools:
  github:
    toolsets: [repos]
safe-outputs:
  create-issue:
    title-prefix: "[audit] "
---

# Security Pattern Audit

Search for deprecated security patterns across organization repositories.

Search for:
- Deprecated crypto libraries
- Insecure authentication patterns
- Known vulnerable dependencies

Focus on repositories in the `security-critical` topic.
Create an issue summarizing findings with repository links.
```

## Deterministic Multi-Repo Workflows

For workflows requiring direct repository access, use custom steps with `actions/checkout`.

### Checking Out Multiple Repositories

Use the custom engine with explicit checkout steps:

```aw wrap
---
on: workflow_dispatch
permissions:
  contents: read
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

    - name: Compare versions
      run: |
        echo "Main version: $(cat main-repo/VERSION)"
        echo "Secondary version: $(cat secondary-repo/VERSION)"

    - name: Generate sync report
      run: |
        diff -u main-repo/package.json secondary-repo/package.json > /tmp/version-diff.txt || true
        cat /tmp/version-diff.txt
---

# Version Comparison Report

This workflow checks out both repositories and compares their dependency versions.
```

### Cloning Specific Branches or Tags

Target specific refs for synchronization workflows:

```yaml wrap
engine:
  id: custom
  steps:
    - name: Checkout upstream release
      uses: actions/checkout@v4
      with:
        repository: upstream-org/library
        ref: refs/tags/v2.5.0
        path: upstream

    - name: Checkout current repo
      uses: actions/checkout@v4
      with:
        path: current
```

### Fetching Remote Repository Data

Access releases, issues, or other GitHub data without full checkout:

```aw wrap
---
on: workflow_dispatch
permissions:
  contents: read
tools:
  github:
    toolsets: [repos]
  bash: true
---

# Fetch Upstream Releases

Check for new releases in upstream dependency repository `org/library`.

1. Get the latest release information from `org/library`
2. Compare with the version currently used in this repo (check package.json)
3. If newer version available, summarize changes from release notes
4. Recommend whether to upgrade based on breaking changes
```

## Feature Synchronization Pattern

A common multi-repo pattern synchronizes features from a main repository to sub-repositories.

### Example: Main to Sub-Repo Sync

```aw wrap
---
on:
  push:
    branches: [main]
    paths:
      - 'shared/**'
permissions:
  contents: read
  actions: read
tools:
  github:
    toolsets: [repos, pull_requests]
  edit:
  bash:
    - "git:*"
safe-outputs:
  github-token: ${{ secrets.CROSS_REPO_PAT }}
  create-pull-request:
    target-repo: "org/sub-repo-alpha"
    title-prefix: "[sync] "
    labels: [auto-sync, main-repo-update]
---

# Sync Shared Components

When shared components change in the main repository, synchronize
them to sub-repository `org/sub-repo-alpha`.

**Process:**
1. Identify changed files in `shared/**` from the recent push
2. Read the current versions of these files in `org/sub-repo-alpha`
3. Adapt changes for sub-repo structure (if needed)
4. Create PR in sub-repo with synchronized changes

**Important:**
- Include commit messages from main repo in PR description
- Link back to original commits
- Note any manual adaptation required
```

### Multi-Target Sync

Synchronize to multiple downstream repositories:

```aw wrap
---
on:
  release:
    types: [published]
permissions:
  contents: read
tools:
  github:
    toolsets: [repos]
safe-outputs:
  github-token: ${{ secrets.CROSS_REPO_PAT }}
  create-issue:
    max: 3
    title-prefix: "[upgrade] "
---

# Notify Downstream Projects

When a new release is published, create upgrade tracking issues
in downstream repositories.

**Target repositories:**
- `org/project-alpha`
- `org/project-beta`
- `org/project-gamma`

For each repository, create an issue with:
- Release version and notes
- Breaking changes (if any)
- Migration guide link
- Suggested upgrade timeline
```

## Organization-Wide Operations

Apply policies or updates across all repositories in an organization.

### Security Policy Enforcement

```aw wrap
---
on:
  schedule:
    - cron: "0 10 * * 1"  # Monday 10AM
permissions:
  contents: read
tools:
  github:
    toolsets: [repos, code_security]
safe-outputs:
  github-token: ${{ secrets.ORG_ADMIN_PAT }}
  create-issue:
    max: 10
    title-prefix: "[security] "
---

# Security Policy Audit

Audit all repositories in the organization for security policy compliance.

**Check each repository for:**
- Branch protection on default branch
- Required code review settings
- Secret scanning enabled
- Dependabot alerts enabled

For non-compliant repositories, create an issue with:
- List of missing security controls
- Links to security policy documentation
- Recommended remediation steps
```

### Dependency Update Coordination

```aw wrap
---
on: workflow_dispatch
permissions:
  contents: read
tools:
  github:
    toolsets: [repos, pull_requests]
safe-outputs:
  github-token: ${{ secrets.CROSS_REPO_PAT }}
  create-issue:
    max: 20
    title-prefix: "[deps] "
---

# Coordinate Dependency Updates

Find all repositories using a specific dependency and create
tracking issues for coordinated updates.

**Target dependency:** `lodash` (specify version to update from/to)

**Process:**
1. Search code across organization for package.json files
2. Identify repos using the target dependency
3. For each repo, create issue with:
   - Current version
   - Target version
   - Breaking changes summary
   - Estimated effort
```

## Best Practices

### Authentication

1. **Use GitHub Apps when possible** for automatic token revocation and better security
2. **Scope PATs minimally** - only grant access to required repositories
3. **Rotate tokens regularly** and monitor for unauthorized use
4. **Use organization secrets** for centralized token management

### Repository Access

1. **Validate repository existence** before creating issues/PRs in target repos
2. **Handle private repos carefully** - ensure tokens have proper access
3. **Document cross-repo dependencies** in workflow comments
4. **Test with public repos first** before expanding to private repositories

### Safe Output Configuration

1. **Use `max` limits** to prevent runaway workflows creating excessive issues/PRs
2. **Apply meaningful prefixes** (`title-prefix`) to identify automated content
3. **Add consistent labels** to track cross-repo automated activities
4. **Include source repository links** in created issues for traceability

### Error Handling

1. **Plan for permission failures** - target repos may restrict automation
2. **Handle rate limits** when operating on many repositories
3. **Provide fallback workflows** if cross-repo operations fail
4. **Monitor workflow costs** across repository operations

## Common Patterns

### Pattern: Hub-and-Spoke Issue Tracking

Central repository aggregates issues from multiple component repos:

```yaml wrap
# In each component repo
safe-outputs:
  github-token: ${{ secrets.CROSS_REPO_PAT }}
  create-issue:
    target-repo: "org/central-tracker"
    title-prefix: "[component-name] "
```

### Pattern: Upstream Change Notification

Notify downstream repos when upstream dependencies change:

```yaml wrap
# In upstream repo
on:
  release:
    types: [published]
safe-outputs:
  create-issue:
    max: 5
    # Creates issues in each downstream repo
```

### Pattern: Organization-Wide Standards

Apply consistent standards across all repositories:

```yaml wrap
# Single workflow, multiple target repos
tools:
  github:
    toolsets: [repos, issues]
# Agent searches and creates issues in non-compliant repos
```

## Examples

Explore complete multi-repo workflow examples:

- [Feature Synchronization](/gh-aw/examples/multi-repo/feature-sync/) - Sync features from main to sub-repos
- [Cross-Repo Tracking](/gh-aw/examples/multi-repo/issue-tracking/) - Central issue tracking across components
- [Org-Wide Updates](/gh-aw/examples/multi-repo/org-wide-updates/) - Coordinate dependency updates

## Related Documentation

- [Safe Outputs Reference](/gh-aw/reference/safe-outputs/) - Complete safe output configuration
- [GitHub Tools](/gh-aw/reference/tools/#github-tools-github) - GitHub API toolsets
- [Security Best Practices](/gh-aw/guides/security/) - Security and authentication strategies
- [Packaging & Distribution](/gh-aw/guides/packaging-imports/) - Sharing workflows across repos
