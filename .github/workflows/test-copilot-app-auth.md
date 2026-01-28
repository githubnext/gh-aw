---
description: Test Copilot with GitHub App Authentication
on: 
  workflow_dispatch:
permissions:
  contents: read
  pull-requests: read
  issues: read
imports:
  - shared/copilot-app.md
  - shared/reporting.md
network:
  allowed:
    - defaults
    - github
tools:
  github:
safe-outputs:
  messages:
    footer: "> 🤖 *Report by [{workflow_name}]({run_url}) with GitHub App auth*"
---

# Test: Copilot Engine with GitHub App Authentication

This workflow demonstrates using GitHub App authentication with the Copilot engine instead of personal access tokens.

## Test Steps

1. Verify that the GitHub App token is properly minted
2. Use the GitHub MCP server tool to list recent pull requests
3. Report success via a comment

## Test Instructions

Use the GitHub MCP server to:
- List the 3 most recent pull requests in ${{ github.repository }}
- Include title, author, and status for each
- Format the results nicely

Then create an issue with:
- Title: "GitHub App Auth Test - ${{ github.run_id }}"
- Body: Results from the PR query above
- Label the issue appropriately if the test passes
