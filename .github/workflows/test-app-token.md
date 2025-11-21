---
on:
  issues:
    types: [opened]
permissions:
  contents: read
safe-outputs:
  create-issue:
    title-prefix: "[app-test] "
    labels: [automation, test]
  app:
    id: ${{ vars.APP_ID }}
    secret: ${{ secrets.APP_PRIVATE_KEY }}
    repository-ids:
      - "12345678"
---

# Test GitHub App Token

This workflow tests the GitHub App token minting functionality.

When an issue is created, analyze it and create a test issue using the GitHub App token.
