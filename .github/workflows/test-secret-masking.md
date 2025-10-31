---
on: workflow_dispatch
permissions:
  contents: read
engine: copilot
imports:
  - shared/secret-redaction-test.md
---

# Test Secret Masking Workflow

This workflow tests the secret-masking feature by importing custom secret redaction steps.

The imported steps will search for and replace the pattern "password123" with "REDACTED" in all files under /tmp/gh-aw/.
