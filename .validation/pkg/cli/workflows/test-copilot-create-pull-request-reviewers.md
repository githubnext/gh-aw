---
on:
  workflow_dispatch:
permissions:
  contents: read
  actions: read
engine: copilot
safe-outputs:
  create-pull-request:
    title-prefix: "[TEST-REVIEWERS] "
    labels: [test, automation]
    reviewers: copilot
    draft: true
---

# Test Copilot Create Pull Request with Reviewers

This is a test workflow to verify that Copilot can create pull requests with automatically assigned reviewers.

Please:
1. Create a new file called `test-reviewers-demo.txt` with a simple message
2. Create a pull request with your changes
3. The pull request should automatically have:
   - The Copilot bot assigned as a reviewer
   - The title prefix "[TEST-REVIEWERS]"
   - Labels: test, automation
   - Draft status: true

This workflow demonstrates the `reviewers` field in the `create-pull-request` safe output configuration with a single reviewer value (string format).
