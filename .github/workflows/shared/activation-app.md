---
#on:
#  github-app:
#    app-id: ${{ vars.APP_ID }}
#    private-key: ${{ secrets.APP_PRIVATE_KEY }}
---

<!--
# Shared Activation GitHub App Configuration

This shared workflow provides repository-level GitHub App configuration for the activation job,
including pre-activation skip-if search checks, reactions, and status comments.

## Configuration Variables

This shared workflow expects:
- **Repository Variable**: `APP_ID` - The GitHub App ID
- **Repository Secret**: `APP_PRIVATE_KEY` - The GitHub App private key

## Usage

Import this configuration in your workflows to enable GitHub App authentication for
skip-if search queries and other activation-job operations:

```yaml
imports:
  - shared/activation-app.md
on:
  schedule: daily
  skip-if-match:
    query: "org:myorg label:in-progress is:issue is:open"
    scope: none
```

The configuration will be automatically inherited by importing workflows (first-wins strategy).

## Benefits

- **Cross-Org Search**: Combine with `scope: none` in skip-if-match / skip-if-no-match to search
  across an organization instead of only the current repository
- **Centralized Configuration**: Single source of truth for app credentials — update once,
  all importing workflows benefit automatically
- **Unified Token**: A single short-lived installation token is minted and shared across all
  skip-if search steps, reactions, and status comments in the activation job
- **Repository-Scoped**: Uses repository-specific variables and secrets

## How It Works

When this shared workflow is imported:
1. The `on.github-app` configuration is extracted and merged into the importing workflow
2. A single `pre-activation-app-token` step is emitted in the pre-activation job
3. All skip-if search steps (skip-if-match and skip-if-no-match) receive this token
4. The token is also used for reactions and status comments when configured

## Token Precedence

1. App token from `on.github-app` (this configuration) — **Highest priority for activation job**
2. Custom token from `on.github-token`
3. Default `GITHUB_TOKEN`

`github-app` and `github-token` are mutually exclusive at the top-level `on:` section.
-->
