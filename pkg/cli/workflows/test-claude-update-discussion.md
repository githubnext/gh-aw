---
on:
  workflow_dispatch:
permissions:
  contents: read
  actions: read
engine: claude
safe-outputs:
  update-discussion:
    max: 1
timeout-minutes: 5
---

# Test Claude Update Discussion

Test the update-discussion safe output with the Claude engine.

Find discussion #1 in this repository and update it.

Change the title to "Updated Discussion Title (Claude Test)" and update the body to include this content:

## Test Update from Claude

This discussion was updated by an automated test workflow to verify the update-discussion functionality.

**Test Information:**
- Engine: Claude
- Test type: Safe output validation
- Timestamp: Test execution time

Output as JSONL format:
```
{"type": "update_discussion", "discussion_number": 1, "title": "Updated Discussion Title (Claude Test)", "body": "<your-content>"}
```
