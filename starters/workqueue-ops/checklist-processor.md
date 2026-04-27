---
name: Checklist Processor
description: Process a queue of work items stored as checkboxes in a GitHub issue, checking each off as it completes.
on:
  workflow_dispatch:
    inputs:
      queue_issue:
        description: "Issue number containing the checklist queue"
        required: true
        type: string
permissions:
  contents: read
  issues: read
tools:
  github:
    toolsets: [issues]
safe-outputs:
  update-issue:
    body: true
  add-comment:
    max: 1
concurrency:
  group: checklist-${{ inputs.queue_issue }}
  cancel-in-progress: false
---

Read issue #${{ inputs.queue_issue }} and find all unchecked checkboxes in its body.
Process up to 5 unchecked items — for each one, perform the described action, then check it off by updating the issue body.
When done, add a comment summarizing how many items were completed and how many remain.
