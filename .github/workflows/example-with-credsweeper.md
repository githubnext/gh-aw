---
# Example workflow demonstrating CredSweeper integration
# This workflow shows how easy it is to add credential scanning to any agentic workflow

on:
  workflow_dispatch:

permissions:
  contents: read
  actions: read

engine: copilot

# Simply import the shared CredSweeper configuration to enable automatic credential scanning
imports:
  - shared/credsweeper.md

tools:
  bash:
    - "echo *"
    - "cat *"
    - "ls *"

timeout_minutes: 10
---

# Example Workflow with CredSweeper

This workflow demonstrates how to add automatic credential scanning to any agentic workflow.

## Your Task

1. List all files in the current directory
2. Create a test file at `/tmp/gh-aw/example.txt` with some sample content
3. Echo "Workflow completed successfully"

After you complete your task, CredSweeper will automatically:
- Scan all files in `/tmp/gh-aw/` for potential credentials
- Mask any detected credentials to prevent exposure in logs
- Generate a security summary in the job summary

**Repository**: ${{ github.repository }}
