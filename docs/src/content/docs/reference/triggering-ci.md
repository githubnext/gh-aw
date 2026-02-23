---
title: Triggering CI
description: How to trigger CI workflow runs on pull requests created by agentic workflows
sidebar:
  order: 805
---

By default, pull requests created using the default `GITHUB_TOKEN` in GitHub Actions **do not trigger CI workflow runs**. This is a [GitHub Actions security feature](https://docs.github.com/en/actions/security-for-github-actions/security-guides/automatic-token-authentication#using-the-github_token-in-a-workflow) to prevent recursive workflow triggers and event cascades.

This applies to both [`create-pull-request`](/gh-aw/reference/safe-outputs/#pull-request-creation-create-pull-request) and [`push-to-pull-request-branch`](/gh-aw/reference/safe-outputs/#push-to-pr-branch-push-to-pull-request-branch) safe outputs.

## Solution 1: Authorize a token for triggering CI on PRs created by workflows

To trigger CI checks on PRs created by agentic workflows, configure a CI trigger token:

```yaml wrap
safe-outputs:
  create-pull-request:
    github-token-for-extra-empty-commit: ${{ secrets.GH_AW_CI_TRIGGER_TOKEN }}  # PAT or token to trigger CI
```

When configured, the token will be used to push an extra empty commit to the PR branch using the specified token after PR creation. Since this push comes from a different authentication context, it triggers `push` and `pull_request` events normally.

This also works for `push-to-pull-request-branch`:

```yaml wrap
safe-outputs:
  push-to-pull-request-branch:
    github-token-for-extra-empty-commit: ${{ secrets.GH_AW_CI_TRIGGER_TOKEN }}
```

## Token Setup

Use a secret expression (e.g. `${{ secrets.GH_AW_CI_TRIGGER_TOKEN }}`) or `app` for GitHub App auth.

### Creating a Fine-Grained PAT

1. Create a [fine-grained PAT](https://github.com/settings/personal-access-tokens/new) with `Contents: Read & Write` scoped to the relevant repositories where pull requests will be created.

2. Add the PAT as a repository secret (e.g., `GH_AW_CI_TRIGGER_TOKEN`).

3. Reference it in your workflow:

```yaml wrap
safe-outputs:
  create-pull-request:
    github-token-for-extra-empty-commit: ${{ secrets.GH_AW_CI_TRIGGER_TOKEN }}
```

### Using GitHub App Authentication

You can also use `app` to authenticate via a GitHub App:

```yaml wrap
safe-outputs:
  create-pull-request:
    github-token-for-extra-empty-commit: app
```

This uses the GitHub App configured for the workflow. See the [Authentication reference](/gh-aw/reference/auth/) for GitHub App setup.

## Alternative: Full Token Override

If you want all PR operations to use a different token (not just the CI trigger), use the `github-token` field instead:

```yaml wrap
safe-outputs:
  create-pull-request:
    github-token: ${{ secrets.CI_USER_PAT }}
```

This changes the author of the PR to the user or app associated with the token, and triggers CI directly. However, it grants more permissions than the empty commit approach.

## See Also

- [Authentication Reference](/gh-aw/reference/auth/) — Token setup and permissions
- [Safe Outputs Reference](/gh-aw/reference/safe-outputs/) — Full safe outputs configuration
