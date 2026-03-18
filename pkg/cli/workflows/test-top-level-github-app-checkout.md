---
on:
  issues:
    types: [opened]
permissions:
  contents: read

github-app:
  app-id: ${{ vars.APP_ID }}
  private-key: ${{ secrets.APP_PRIVATE_KEY }}
checkout:
  repository: myorg/private-repo
  path: private
safe-outputs:
  create-issue:
    title-prefix: "[automated] "
engine: copilot
---

# Top-Level GitHub App Fallback for Checkout

This workflow demonstrates using a top-level github-app as a fallback for checkout operations.

The top-level `github-app` is automatically applied to checkout operations that do not have
their own `github-app` or `github-token` configured.

This is useful for checking out private repositories using the GitHub App installation token.
