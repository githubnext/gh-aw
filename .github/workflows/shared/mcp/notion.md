---
mcp-servers:
  notion:
    container: "mcp/notion"
    env:
      NOTION_API_TOKEN: "${{ secrets.NOTION_API_TOKEN }}"
    allowed:
      - "search_pages"
      - "get_page"
      - "get_database"
      - "query_database"
safe-outputs:
  jobs:
    notion-add-comment:
      description: "Add a comment to a Notion page"
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
          uses: actions/github-script@v8
          env:
            NOTION_API_TOKEN: "${{ secrets.NOTION_API_TOKEN }}"
            PAGE_ID: "${{ inputs.page_id }}"
            COMMENT: "${{ inputs.comment }}"
          with:
            script: |
              const notionToken = process.env.NOTION_API_TOKEN;
              const pageId = process.env.PAGE_ID;
              const comment = process.env.COMMENT;
              
              if (!notionToken) {
                core.setFailed('NOTION_API_TOKEN secret is not configured');
                return;
              }
              
              core.info(`Adding comment to Notion page: ${pageId}`);
              
              try {
                const response = await fetch('https://api.notion.com/v1/comments', {
                  method: 'POST',
                  headers: {
                    'Authorization': `Bearer ${notionToken}`,
                    'Notion-Version': '2022-06-28',
                    'Content-Type': 'application/json'
                  },
                  body: JSON.stringify({
                    parent: {
                      page_id: pageId
                    },
                    rich_text: [{
                      type: 'text',
                      text: {
                        content: comment
                      }
                    }]
                  })
                });
                
                if (!response.ok) {
                  const errorData = await response.text();
                  core.setFailed(`Notion API error (${response.status}): ${errorData}`);
                  return;
                }
                
                const data = await response.json();
                core.info('Comment added successfully');
                core.info(`Comment ID: ${data.id}`);
              } catch (error) {
                core.setFailed(`Failed to add comment: ${error.message}`);
              }
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
