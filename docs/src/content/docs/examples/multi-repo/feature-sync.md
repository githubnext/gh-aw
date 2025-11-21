---
title: Feature Synchronization
description: Synchronize features from a main repository to sub-repositories or downstream services with automated pull requests.
sidebar:
  badge: { text: 'Multi-Repo', variant: 'note' }
---

Feature synchronization workflows propagate changes from a main repository to related sub-repositories, ensuring downstream projects stay current with upstream improvements while maintaining proper change tracking through pull requests.

## When to Use

- **Monorepo alternatives** - Maintain related projects in separate repos while sharing common code
- **Library updates** - Sync shared utilities or components to dependent projects
- **Multi-platform deployment** - Update platform-specific repos when core logic changes
- **Fork maintenance** - Keep downstream forks synchronized with upstream changes

## How It Works

The workflow monitors specific paths in the main repository and creates pull requests in target repositories when changes occur, adapting the changes for each target's structure while maintaining full audit trails.

## Basic Feature Sync

Synchronize changes from shared directory to downstream repository:

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
    toolsets: [repos]
  edit:
  bash:
    - "git:*"
safe-outputs:
  github-token: ${{ secrets.CROSS_REPO_PAT }}
  create-pull-request:
    target-repo: "myorg/downstream-service"
    title-prefix: "[sync] "
    labels: [auto-sync, upstream-update]
    reviewers: [team-lead]
    draft: true
---

# Sync Shared Components to Downstream Service

When shared components change in this repository, synchronize them to
`myorg/downstream-service`.

**Changed files:** Review the git diff to identify modified files in `shared/**`

**Synchronization steps:**
1. Read current versions of these files from `myorg/downstream-service`
2. Adapt changes if needed (check for path differences)
3. Create descriptive commit messages referencing original commits
4. Include migration notes if breaking changes detected

**PR Description should include:**
- List of synchronized files
- Links to original commits in main repo
- Any structural adaptations made
- Required follow-up actions (if any)
```

## Multi-Target Sync

Synchronize to multiple repositories simultaneously:

```aw wrap
---
on:
  push:
    branches: [main]
    paths:
      - 'core/**'
permissions:
  contents: read
  actions: read
tools:
  github:
    toolsets: [repos]
  edit:
  bash:
    - "git:*"
safe-outputs:
  github-token: ${{ secrets.CROSS_REPO_PAT }}
  create-pull-request:
    max: 3
    title-prefix: "[core-sync] "
    labels: [automated-sync]
    draft: true
---

# Sync Core Library to All Services

When core library files change, create PRs in all dependent services.

**Target repositories:**
- `myorg/api-service`
- `myorg/web-frontend`
- `myorg/mobile-backend`

For each target repository:
1. Check if they use the changed core modules
2. Adapt imports/paths for target's structure
3. Create PR with synchronized changes
4. Include compatibility notes

**Requirements:**
- Maintain backward compatibility when possible
- Document any breaking changes clearly
- Link to main repo commits
```

## Release-Based Sync

Synchronize when new releases are published:

```aw wrap
---
on:
  release:
    types: [published]
permissions:
  contents: read
  actions: read
tools:
  github:
    toolsets: [repos]
  edit:
  bash:
    - "git:*"
safe-outputs:
  github-token: ${{ secrets.CROSS_REPO_PAT }}
  create-pull-request:
    target-repo: "myorg/production-service"
    title-prefix: "[upgrade] "
    labels: [version-upgrade, auto-generated]
    reviewers: [release-manager]
    draft: false
---

# Upgrade Production Service to New Release

When a new release is published, create an upgrade PR in the production service.

**Release information:**
- Version: ${{ github.event.release.tag_name }}
- Release notes: ${{ github.event.release.body }}

**Upgrade steps:**
1. Update version references in production service
2. Apply any necessary API changes from release notes
3. Update configuration if breaking changes exist
4. Include migration guide in PR description

**PR should contain:**
- Updated dependency version
- Code adaptations for API changes
- Configuration updates
- Link to release notes
- Testing recommendations
```

## Selective File Sync

Synchronize only specific file types or patterns:

```aw wrap
---
on:
  push:
    branches: [main]
    paths:
      - 'types/**/*.ts'
      - 'interfaces/**/*.ts'
permissions:
  contents: read
  actions: read
tools:
  github:
    toolsets: [repos]
  edit:
  bash:
    - "git:*"
safe-outputs:
  github-token: ${{ secrets.CROSS_REPO_PAT }}
  create-pull-request:
    target-repo: "myorg/client-sdk"
    title-prefix: "[types] "
    labels: [type-definitions]
    draft: true
---

# Sync TypeScript Type Definitions

Synchronize TypeScript type definitions and interfaces to client SDK.

**Process:**
1. Identify changed `.ts` files in `types/` and `interfaces/` directories
2. Read corresponding files in `myorg/client-sdk`
3. Update type definitions maintaining existing structure
4. Preserve client-specific type extensions
5. Validate no breaking changes to public interfaces

**Include in PR:**
- List of updated type files
- Breaking changes (if any)
- Compatibility notes
```

## Bidirectional Sync with Conflict Detection

Handle bidirectional synchronization with conflict awareness:

```aw wrap
---
on:
  push:
    branches: [main]
    paths:
      - 'shared-config/**'
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
    target-repo: "myorg/sister-project"
    title-prefix: "[config-sync] "
    labels: [config-update, needs-review]
    draft: true
---

# Bidirectional Config Sync

Synchronize shared configuration files while detecting conflicts.

**Important:** This project and `myorg/sister-project` share configuration
that may be modified independently.

**Sync process:**
1. Get current state of shared-config in both repos
2. Compare timestamps and change history
3. Identify if sister-project has newer changes
4. If conflict detected:
   - Create PR with this repo's changes
   - Add comment noting the conflict
   - Mark for manual review
5. If no conflict:
   - Apply changes automatically
   - Note last sync timestamp

**Conflict resolution notes in PR if needed**
```

## Feature Branch Sync

Synchronize feature branches between repositories:

```aw wrap
---
on:
  pull_request:
    types: [opened, synchronize]
    branches:
      - 'feature/**'
permissions:
  contents: read
  pull-requests: read
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
    target-repo: "myorg/integration-tests"
    title-prefix: "[feature-test] "
    labels: [feature-branch, auto-sync]
    draft: true
---

# Sync Feature Branch for Integration Testing

When a feature branch is updated, synchronize to integration test repository.

**Source PR:** #${{ github.event.pull_request.number }}
**Branch:** ${{ github.event.pull_request.head.ref }}

**Process:**
1. Create matching feature branch in integration test repo
2. Sync relevant changes from this PR
3. Update test configurations for new feature
4. Create PR for integration test updates

**PR description should include:**
- Link to source PR
- Feature description
- Test scenarios to cover
- Expected integration points
```

## Scheduled Sync Check

Regularly check for sync drift and create catch-up PRs:

```aw wrap
---
on:
  schedule:
    - cron: "0 9 * * 1"  # Monday 9AM
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
    target-repo: "myorg/downstream-fork"
    title-prefix: "[weekly-sync] "
    labels: [scheduled-sync]
    draft: true
---

# Weekly Sync Check

Check for accumulated changes that need synchronization to downstream fork.

**Comparison:**
- Last sync commit in downstream: Check PR history for last `[sync]` PR
- Current HEAD in main repo
- Identify all commits since last sync

**Sync process:**
1. List all commits since last sync
2. Categorize changes (features, fixes, docs)
3. Identify changes relevant to downstream
4. Create comprehensive PR with all updates
5. Group commits by category in description

**PR should include:**
- Summary of all synced commits
- Breaking changes highlighted
- Migration guide if needed
- Testing recommendations
```

## Authentication Setup

All cross-repo sync workflows require proper authentication:

### PAT Configuration

```bash
# Create PAT with required permissions
gh auth token

# Store as repository secret
gh secret set CROSS_REPO_PAT --body "ghp_your_token_here"
```

**Required PAT Permissions:**
- `repo` (full control for private repos)
- `contents: write` (for creating commits)
- `pull-requests: write` (for creating PRs)

### GitHub App Configuration

For enhanced security, use GitHub App installation tokens:

```yaml wrap
safe-outputs:
  app:
    app-id: ${{ vars.APP_ID }}
    private-key: ${{ secrets.APP_PRIVATE_KEY }}
    repositories: ["downstream-service", "integration-tests"]
  create-pull-request:
    target-repo: "myorg/downstream-service"
```

## Best Practices

### Change Detection

1. **Use path filters** in trigger configuration to avoid unnecessary runs
2. **Check for meaningful changes** before creating PRs (not just whitespace)
3. **Group related changes** into single PRs when appropriate
4. **Track sync history** to avoid duplicate PRs

### PR Management

1. **Create draft PRs** for automatic syncs requiring review
2. **Use consistent labels** for tracking automated syncs
3. **Add reviewers** appropriate to the changes
4. **Include comprehensive descriptions** linking back to source commits

### Error Handling

1. **Handle merge conflicts** gracefully with clear documentation
2. **Validate target repo structure** before creating PRs
3. **Provide rollback instructions** in PR descriptions
4. **Monitor for failed syncs** and alert maintainers

### Testing

1. **Test sync workflows** on public repos first
2. **Verify path mappings** between source and target
3. **Check for breaking changes** before applying
4. **Validate in staging** before production deployment

## Related Documentation

- [Multi-Repo Operations Guide](/gh-aw/guides/multi-repo-ops/) - Complete multi-repo overview
- [Cross-Repo Issue Tracking](/gh-aw/examples/multi-repo/issue-tracking/) - Issue management patterns
- [Safe Outputs Reference](/gh-aw/reference/safe-outputs/) - Pull request configuration
- [GitHub Tools](/gh-aw/reference/tools/#github-tools-github) - Repository access tools
