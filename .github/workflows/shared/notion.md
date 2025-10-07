---
mcp-servers:
  notion:
    container: "mcp/notion"
    env:
      NOTION_TOKEN: "${{ secrets.NOTION_TOKEN }}"
    allowed:
      - "search_pages"
      - "get_page"
      - "get_database"
      - "query_database"
safe-outputs:
  jobs:
    notion-add-comment:
      runs-on: ubuntu-latest
      output: "Comment added to Notion successfully!"
      inputs:
        page_id:
          description: "The Notion page ID to add a comment to"
          required: true
          type: string
        comment:
          description: "The comment text to add"
          required: true
          type: string
      permissions:
        contents: read
      steps:
        - name: Add comment to Notion page
          env:
            NOTION_TOKEN: "${{ secrets.NOTION_TOKEN }}"
            PAGE_ID: "${{ inputs.page_id }}"
            COMMENT: "${{ inputs.comment }}"
          run: |
            echo "Adding comment to Notion page: $PAGE_ID"
            
            # Make API request to add comment to Notion page
            curl -X POST "https://api.notion.com/v1/comments" \
              -H "Authorization: Bearer $NOTION_TOKEN" \
              -H "Notion-Version: 2022-06-28" \
              -H "Content-Type: application/json" \
              -d "{
                \"parent\": {
                  \"page_id\": \"$PAGE_ID\"
                },
                \"rich_text\": [{
                  \"type\": \"text\",
                  \"text\": {
                    \"content\": \"$COMMENT\"
                  }
                }]
              }"
            
            echo "Comment added successfully"
---

## Notion Integration

This shared configuration provides Notion MCP server integration with read-only tools and a custom safe-job for adding comments to Notion pages.

### Available Notion MCP Tools (Read-Only)

- `search_pages`: Search for Notion pages
- `get_page`: Get details of a specific page
- `get_database`: Get database schema
- `query_database`: Query database content

### Safe Job: notion-add-comment

The `notion-add-comment` safe-job allows the agentic workflow to add comments to Notion pages through the Notion API.

**Required Inputs:**
- `page_id`: The Notion page ID to add a comment to
- `comment`: The comment text to add

**Example Usage in Workflow:**

```
Please add a summary comment to the Notion page with ID "abc123" using the notion-add-comment safe-job.
```

### Setup

1. Add `NOTION_TOKEN` secret to your repository with a Notion integration token
2. Grant the integration access to the pages/databases you want to interact with
3. Include this configuration in your workflow: `@include shared/notion.md`
