---
on: 
  workflow_dispatch:
name: Dev
description: Test workflow for development and experimentation purposes
timeout-minutes: 5
strict: false
engine: codex
permissions:
  contents: read
  issues: read
  pull-requests: read
  discussions: read
imports:
  - shared/gh.md
tools:
  github: false
  bash: ["*"]
  edit:
---
Use the `gh` safe-input tool to get information about the last PR in this repository.

1. First, use the `gh` tool with `args: "pr list --limit 1 --json number,title,body,author,state,createdAt,mergedAt,url"` to get the most recent PR
2. Summarize the PR including its title, author, state, and a brief description of what it does

Present the summary in a clear format.