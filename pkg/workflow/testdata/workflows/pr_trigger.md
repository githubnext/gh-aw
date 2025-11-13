---
on:
  pull_request:
    types: [opened, synchronize]
permissions:
  pull-requests: write
  contents: read
engine: claude
tools:
  github:
    toolsets: [default]
timeout-minutes: 15
---

# Pull Request Review Workflow

Review the pull request changes and provide feedback.
