---
on:
  workflow_dispatch:
permissions:
  contents: read
engine:
  id: codex
  env:
    OPENAI_API_KEY: ${{ secrets.OPENAI_API_KEY_CI }}
---

# Test Codex Custom Environment Variable

This is a test workflow to demonstrate how to configure custom environment variables for the Codex engine, specifically overriding the default OPENAI_API_KEY secret.

Please analyze the current repository structure and list the main directories and their purposes.