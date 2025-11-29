---
on: 
  workflow_dispatch:
name: Dev
description: Test workflow for development and experimentation purposes
timeout-minutes: 5
strict: false
# Using experimental Claude engine for testing
engine: claude
permissions:
  contents: read
  issues: read
  pull-requests: read
  discussions: read
imports:
  - shared/pr-data-safe-input.md
tools:
  bash: ["*"]
  edit:
  github:
    toolsets: [default, repos, issues, discussions]
safe-outputs:
  assign-to-agent:
---
Use the `fetch-pr-data` tool to fetch Copilot agent PRs from this repository using `search: "head:copilot/"`. Then compute basic PR statistics:
- Total number of Copilot PRs in the last 30 days
- Number of merged vs closed vs open PRs
- Average time from PR creation to merge (for merged PRs)
- Most active day of the week for PR creation

Present the statistics in a clear summary.