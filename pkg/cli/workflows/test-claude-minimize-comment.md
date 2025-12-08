---
on:
  workflow_dispatch:
engine: claude
safe-outputs:
  minimize-comment:
    max: 3
timeout-minutes: 5
---

# Test Claude Minimize Comment

This is a test workflow to verify that Claude can minimize (hide) comments on GitHub issues.

Test the minimize_comment safe output by minimizing a comment with the following node ID:

- comment_id: "IC_kwDOABCD123456"

Output the minimize-comment action as JSONL format using the minimize_comment tool.
