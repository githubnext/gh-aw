---
engine:
  id: copilot
  app:
    app-id: ${{ vars.APP_ID }}
    private-key: ${{ secrets.APP_PRIVATE_KEY }}
---

<!--

# GitHub Copilot with App Authentication

This shared workflow configures the GitHub Copilot engine to use GitHub App authentication instead of personal access tokens.

## Configuration

When imported, this provides:
- **GitHub App authentication** for Copilot CLI
- **Short-lived tokens** with `copilot-requests: read` permission
- **Automatic token invalidation** after workflow completion

## Usage

Import this workflow to enable GitHub App authentication for Copilot:

```yaml
---
engine: copilot
imports:
  - shared/copilot-app.md
---
```

## Requirements

Configure the following in your repository:
- **vars.APP_ID** - GitHub App ID
- **secrets.APP_PRIVATE_KEY** - GitHub App private key (PEM format)

## Permissions

The generated token will have:
- `copilot-requests: read` - Required for GitHub Copilot CLI access

Additional permissions can be inherited from workflow-level `permissions:` configuration.

## Benefits

- **Security**: Short-lived tokens (max 1 hour) instead of long-lived PATs
- **Audit**: App activity tracked separately in GitHub audit logs
- **Rotation**: No need to rotate tokens manually
- **Least privilege**: Minimal permissions for Copilot access

-->
