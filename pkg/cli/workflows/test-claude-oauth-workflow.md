---
name: test-claude-oauth
description: Test workflow to validate CLAUDE_CODE_OAUTH_TOKEN support
on:
  issues:
    types: [opened]
permissions:
  contents: read
  issues: read
  pull-requests: read
engine: claude
---

Test the Claude OAuth token support by listing files.
