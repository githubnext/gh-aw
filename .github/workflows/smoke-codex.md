---
on: 
  schedule:
    - cron: "0 0,6,12,18 * * *"  # Every 6 hours
  workflow_dispatch:
  pull_request:
    types: [labeled]
    names: ["smoke"]
permissions:
  contents: read
  issues: read
  pull-requests: read
name: Smoke Codex
engine: codex
tools:
  github:
safe-outputs:
    staged: true
    create-issue:
    create-commit-status:
      context: "ci/smoke-codex"
      max: 1
timeout-minutes: 10
strict: true
---

Review the last 2 merged pull requests in this repository and post summary in an issue.

If triggered by a pull request (labeled with "smoke"), create a commit status indicating the smoke test result:
- Use `state: "success"` if the review completes successfully
- Use `state: "failure"` if there are any errors
- Include brief description of the test result