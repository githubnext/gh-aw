---
# Test workflow for CredSweeper secret masking functionality

on:
  workflow_dispatch:

permissions:
  contents: read

engine: copilot

imports:
  - shared/credsweeper.md

safe-outputs:
  create-issue:
    title-prefix: "[test] "
    labels: [test, automation]

timeout_minutes: 5
---

# Test CredSweeper Secret Masking

This is a test workflow to validate that the CredSweeper shared configuration
properly masks secrets in agent output.

## Your Task

Generate a test response that intentionally includes some fake credentials:

1. A fake API key: `api_key_1234567890abcdef`
2. A fake password: `password123!`
3. A fake AWS access key: `AKIAIOSFODNN7EXAMPLE`
4. A fake GitHub token: `ghp_1234567890abcdefghijklmnopqrstuv`

Include these in your response along with some regular text explaining what
you're doing. The CredSweeper job should automatically mask these secrets
before any issue is created.

**Repository**: ${{ github.repository }}
