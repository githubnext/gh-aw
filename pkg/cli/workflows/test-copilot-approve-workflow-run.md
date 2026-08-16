---
on: workflow_dispatch
permissions:
  contents: read
  actions: write
engine: copilot
safe-outputs:
  approve-workflow-run:
    max: 1
timeout-minutes: 5
strict: false
---

# Test Approve Workflow Run

You are testing the approve_workflow_run safe output type.

## Instructions

Your task is to approve a pending workflow run that is awaiting required approval. Follow these steps:

1. Call the `approve_workflow_run` tool with:
   - `run_id`: The ID of the workflow run to approve
   - `comment` (optional): A short note explaining why the run is approved

## Example

```
approve_workflow_run({
  run_id: 123456789,
  comment: "Approved after reviewing the workflow changes; no risky permissions requested."
})
```

## Expected Behavior

- In **staged mode**: Shows a preview of what would be done (with 🎭 emoji)
- In **live mode**: Actually approves the pending workflow run
- Only runs matching the configured `allowed-workflows`/`allowed-pull-requests` policy may be approved
- Comment is sanitized to prevent XSS attacks

Please test this functionality by calling the tool with an appropriate run ID and comment.
