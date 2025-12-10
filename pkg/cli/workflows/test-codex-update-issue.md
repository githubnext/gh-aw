---
on:
  workflow_dispatch:
permissions:
  contents: read
engine: codex
safe-outputs:
  update-issue:
    max: 1
    target: "1"
    status:
    title:
    body:
timeout-minutes: 5
---

# Test Codex Update Issue

This is a test workflow to verify that Codex can update existing GitHub issues using safe-outputs.

Your task: Update issue #1 using the update_issue tool.

Please:
1. Change the title to "Updated Test Issue"
2. Update the body to add a new section: "## Test Update\n\nThis issue was updated by an automated test workflow."
3. Set the status to "open" (or "closed" if it's already open)

Use the update_issue tool with the appropriate parameters. Output as JSONL format.
