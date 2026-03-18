---
on:
  issues:
    types: [opened]
permissions:
  contents: read

github-app:
  app-id: ${{ vars.APP_ID }}
  private-key: ${{ secrets.APP_PRIVATE_KEY }}
dependencies:
  packages:
    - myorg/private-skill
safe-outputs:
  create-issue:
    title-prefix: "[automated] "
engine: copilot
---

# Top-Level GitHub App Fallback for APM Dependencies

This workflow demonstrates using a top-level github-app as a fallback for APM dependencies.

The top-level `github-app` is automatically applied to APM package installations when no
`dependencies.github-app` is configured. This allows installing APM packages from private
repositories across organizations using the GitHub App installation token.
