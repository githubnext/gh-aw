---
on: 
  workflow_dispatch:
concurrency:
  group: dev-workflow-${{ github.ref }}
  cancel-in-progress: true
name: Dev
description: Test workflow for development and experimentation purposes
engine: copilot
permissions:
  contents: read
  issues: read
  actions: read
tools:
  github:
    allowed:
      - list_issues
safe-outputs:
  assign-milestone:
    allowed: ["ultimate"]
    target: "*"
    max: 1
timeout-minutes: 10
---

# Assign Random Issue to Ultimate Milestone

Find a random open issue in the repository and assign it to the "ultimate" milestone.

**Instructions**:

1. Use the GitHub tool to list open issues in the repository
2. Select a random issue from the list
3. Assign that issue to the "ultimate" milestone using the assign_milestone safe output

Output the assignment as JSONL format:
```jsonl
{"type": "assign_milestone", "milestone": "ultimate", "item_number": <issue_number>}
```

Replace `<issue_number>` with the actual issue number you selected.
