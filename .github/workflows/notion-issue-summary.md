---
on:
  issues:
    types: [opened]
  workflow_dispatch:
    inputs:
      text:
        description: "Issue text to analyze"
        required: true
        type: string
permissions:
  contents: read
  actions: read
engine: claude
imports:
  - shared/mcp/notion.md
timeout_minutes: 10
strict: true
---

# Issue Summary to Notion

Analyze the issue and create a brief summary, then add it as a comment to the Notion page.

<issue_content>
${{ needs.activation.outputs.text }}
</issue_content>

## Instructions

1. Read and analyze the issue content
2. Create a concise summary (2-3 sentences) of the issue
3. Use the `notion-add-comment` safe-job to add your summary as a comment to the Notion page

**Important Notes:**
- Only use read-only Notion MCP tools (search_pages, get_page, get_database, query_database)
- Use the notion-add-comment safe-job to add comments (not the MCP tools)
- The Notion page ID should be obtained by searching for a page related to "GitHub Issues" or similar
