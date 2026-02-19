---
name: with-expressions-test
description: Workflow with GitHub expressions in the prompt
on:
  issues:
    types: [opened]
permissions:
  contents: read
  issues: read
engine: copilot
timeout-minutes: 10
---

# Mission

Handle issue ${{ github.event.issue.number }} titled "${{ github.event.issue.title }}" in repository ${{ github.repository }}.

Analyze the issue content and provide a helpful response.
