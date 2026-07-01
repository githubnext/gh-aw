---
on:
  workflow_dispatch:
    inputs:
      review-team:
        description: 'Team slug to request on the created pull request'
        required: true
        type: string
permissions:
  contents: read
  actions: read
engine: copilot
safe-outputs:
  create-pull-request:
    title-prefix: "[TEST-RUNTIME-ROUTING] "
    reviewers: ${{ github.actor }}
    team-reviewers: ${{ inputs.review-team }}
    assignees: ${{ github.actor }}
    draft: true
---

# Test Copilot Create Pull Request with Runtime Routing

This is a test workflow to verify that `create-pull-request` accepts runtime expressions for reviewers, team reviewers, and assignees.

Please:
1. Create a new file called `test-runtime-routing-demo.txt` with a simple message
2. Use the `create-pull-request` safe output to create a pull request with your changes
3. Confirm the created pull request automatically has:
   - The triggering actor assigned as a reviewer (check the Reviewers section in the GitHub PR sidebar)
   - The workflow input team assigned as a team reviewer (check the Reviewers section in the GitHub PR sidebar)
   - The triggering actor assigned as an assignee (check the Assignees section in the GitHub PR sidebar)
   - The title prefix "[TEST-RUNTIME-ROUTING]"
   - Draft status: true

This workflow demonstrates runtime expression support for `reviewers`, `team-reviewers`, and `assignees` on `create-pull-request`.
