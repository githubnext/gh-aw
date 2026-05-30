---
on:
  workflow_dispatch:
permissions:
  contents: read
  actions: read
engine: claude
safe-outputs:
  update-release:
    max: 1
timeout-minutes: 5
---

# Test Claude Update Release

Test the update-release safe output with the Claude engine.

Find the latest release in this repository and update its description using the **append** operation.

Add this content to the release notes:

## Test Update from Claude

This section was added by an automated test workflow to verify the update-release functionality.

**Test Details:**
- Engine: Claude
- Operation: append
- Timestamp: Current date and time

Output as JSONL format:
```
{"type": "update_release", "tag": "<tag-name>", "operation": "append", "body": "<content>"}
```
