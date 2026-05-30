---
on:
  issues:
    types: [opened]
  reaction: eyes
permissions:
  contents: read

github-app:
  app-id: ${{ vars.APP_ID }}
  private-key: ${{ secrets.APP_PRIVATE_KEY }}
safe-outputs:
  create-issue:
    title-prefix: "[automated] "
engine: copilot
---

# Top-Level GitHub App Fallback for Activation

This workflow demonstrates using a top-level github-app as a fallback for activation operations.

The top-level `github-app` is automatically applied to the activation job (reactions, status
comments, skip-if checks) when no `on.github-app` is defined.

When an issue is opened, react with 👀 and create a follow-up issue using the GitHub App token.
