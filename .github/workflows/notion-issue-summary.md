---
on:
  workflow_dispatch:
    inputs:
      text:
        description: "Issue text to analyze"
        required: true
        type: string
tools:
  github:
imports:
  - shared/mcp/notion.md
strict: true
---

# Issue Summary to Notion

Analyze the issue and create a brief summary, then add it as a comment to the Notion page.

## Instructions

1. Read and analyze the issue content
2. Create a concise summary (2-3 sentences) of the issue
3. Use the `notion-add-comment` safe-job to add your summary as a comment to the Notion page
