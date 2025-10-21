---
# Test workflow for CredSweeper shared configuration
# This workflow validates that CredSweeper scans and masks credentials correctly

on:
  workflow_dispatch:

permissions:
  contents: read
  actions: read

engine: copilot

imports:
  - ../../../.github/workflows/shared/credsweeper.md

tools:
  bash:
    - "echo *"
    - "mkdir *"

timeout_minutes: 10
---

# Test CredSweeper Integration

This workflow tests the CredSweeper shared configuration.

## Your Task

Create some test files in `/tmp/gh-aw/` directory with fake credentials to test the CredSweeper scanner:

1. Create `/tmp/gh-aw/test-creds.txt` with a fake AWS access key pattern like "AKIAIOSFODNN7EXAMPLE"
2. Create `/tmp/gh-aw/test-password.json` with a JSON object containing a password field
3. Create `/tmp/gh-aw/test-token.log` with a fake API token pattern

After creating these files, respond with "Test files created successfully".

**Repository**: ${{ github.repository }}
