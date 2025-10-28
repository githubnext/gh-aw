---
on: 
  workflow_dispatch:
  pull_request:
    branches:
      - copilot/*
concurrency:
  group: dev-workflow-${{ github.ref }}
  cancel-in-progress: true
name: Dev
engine: copilot
permissions:
  contents: read
  actions: read
tools:
  edit:
safe-outputs:
  threat-detection: false
  push-to-pull-request-branch:
---

# Add Emoji to File

Create or update an `emoji.md` file with an emoji and push the changes to the pull request branch.

**Instructions**: 

Use the `edit` tool to either create a new `emoji.md` file or update the existing one if it already exists. Choose a fun, creative emoji that represents GitHub Agentic Workflows.

Commit your changes.

Call the `push-to-pull-request-branch` tool after making your changes.

**Example emoji file structure:**
```markdown
# Emoji for Agentic Workflows

🤖✨
```
