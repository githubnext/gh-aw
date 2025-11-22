---
on:
  pull_request:
    types: [opened]
permissions:
  contents: read
  actions: read
engine: copilot
safe-outputs:
  add-reviewer:
    max: 3
timeout-minutes: 5
---

# Test Add Reviewer Safe Output

Test the add-reviewer safe output functionality.

Add reviewers to the pull request using the add_reviewer tool:
- Add "octocat" as a reviewer
- Add "github" as a reviewer

Output as JSONL format.
