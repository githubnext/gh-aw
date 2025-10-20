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

1. Generate a test response that intentionally includes some fake credentials:
   - A fake API key: `api_key_1234567890abcdef`
   - A fake password: `password123!`
   - A fake AWS access key: `AKIAIOSFODNN7EXAMPLE`
   - A fake GitHub token: `ghp_1234567890abcdefghijklmnopqrstuv`

2. **IMPORTANT**: Before creating the issue, use the `mask_secrets` tool to scan
   your response text and mask any detected secrets.

3. After masking, create an issue with the masked text using the `create_issue` tool.

Include these fake credentials in your initial response along with some regular 
text explaining what you're doing. The CredSweeper job should mask these secrets 
when you call the mask_secrets tool.

**Repository**: ${{ github.repository }}
