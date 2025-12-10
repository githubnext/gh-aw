---
on:
  workflow_dispatch:
engine: claude
safe-outputs:
  hide-comment:
    max: 3
timeout-minutes: 5
---

# Test Claude Hide Comment

This is a test workflow to verify that Claude can hide comments on GitHub issues.

Test the hide_comment safe output by hiding a comment with the following node ID:

- comment_id: "IC_kwDOABCD123456"

Output the hide-comment action as JSONL format using the hide_comment tool.
