---
on:
  workflow_dispatch:
permissions:
  contents: read
  issues: write
engine: copilot
safe-outputs:
  create-issue:
    max: 1
  failure-issue-repo: github/gh-aw
timeout-minutes: 5
---

# Test Copilot Failure Issue Repo

Test the `failure-issue-repo` safe-outputs configuration which redirects
agent failure issues to a specific repository (format: "owner/repo").

Create an issue summarising the failure-issue-repo configuration:
- title: "Failure Issue Repo Test"
- body: "This workflow validates that failure-issue-repo is set to 'github/gh-aw'. Any agent failures will create tracking issues in that repository."

Output as JSONL using the `create_issue` tool.
