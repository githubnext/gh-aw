---
on: 
  workflow_dispatch:
name: Dev
description: List the last 3 issues using gh CLI
timeout-minutes: 5
strict: false
engine: claude
permissions:
  contents: read
  issues: read
features:
  safe-inputs: true
tools:
  github: false
imports:
  - shared/gh.md
---
# List Last 3 Issues

List the last 3 issues in this repository using the gh CLI tool.

## Task

1. **Use gh CLI**: Use the `gh` tool to list the last 3 issues in this repository.
   
   Example invocation:
   ```
   gh with args: "issue list --limit 3 --repo ${{ github.repository }}"
   ```

2. **Display results**: Show the output from the gh CLI command.