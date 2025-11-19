---
on:
  workflow_dispatch:
permissions:
  contents: read
  actions: read
engine: codex
safe-outputs:
  update-release:
    max: 1
timeout-minutes: 5
---

# Test Codex Update Release

Test the update-release safe output with the Codex engine.

Find the latest release in this repository and update its description using the **replace** operation.

Replace the release notes with this content:

## Updated Release Notes (Codex Test)

This release description was updated by an automated test workflow to verify the update-release functionality.

**Test Configuration:**
- Engine: Codex
- Operation: replace
- Timestamp: Current date and time

Output as JSONL format:
```
{"type": "update_release", "tag": "<tag-name>", "operation": "replace", "body": "<content>"}
```
