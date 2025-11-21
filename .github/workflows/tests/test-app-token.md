---
on:
  issues:
    types: [opened, labeled]
permissions:
  contents: read
safe-outputs:
  create-issue:
    title-prefix: "[automated] "
    labels: [automation]
  app:
    id: ${{ vars.APP_ID }}
    secret: ${{ secrets.APP_PRIVATE_KEY }}
    repository-ids:
      - "12345678"  # Scope token to specific repository
---

# Issue Triage with GitHub App Token

This workflow demonstrates using a GitHub App token for safe outputs.

When an issue is opened or labeled, analyze it and create a triage issue using the GitHub App token.

## Benefits of Using GitHub App Tokens

- **Enhanced Security**: Tokens are minted on-demand and automatically revoked
- **Fine-grained Permissions**: Scope tokens to specific repositories
- **Better Attribution**: Actions appear as the GitHub App, not a user
- **Audit Trail**: Clear tracking of automated actions

## How it Works

1. The workflow mints a GitHub App installation access token
2. All safe output operations use this token
3. The token is automatically invalidated when the job completes
