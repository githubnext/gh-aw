---
on: workflow_dispatch
permissions:
  contents: read
  actions: read
  discussions: read
engine: copilot
tools:
  github:
    toolsets: [default, discussions]
safe-outputs:
  close-discussion:
    required-category: "Ideas"
    max: 1
timeout-minutes: 5
strict: false
---

# Test Close Discussion

Test the close-discussion safe output functionality.

## Task

Create a close_discussion output to close the current discussion.

1. Add a comment summarizing: "This discussion has been resolved and converted into actionable tasks."
2. Set the resolution reason to "RESOLVED"
3. Output as JSONL format with type "close_discussion"

The close-discussion safe output should:
- Only close discussions in the "Ideas" category (configured via required-category filter)
- Add the comment before closing
- Apply the RESOLVED reason

Example JSONL output:
```jsonl
{"type":"close_discussion","body":"This discussion has been resolved and converted into actionable tasks.","reason":"RESOLVED"}
```
