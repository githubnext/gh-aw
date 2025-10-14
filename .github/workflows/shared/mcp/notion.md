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
            NOTION_API_TOKEN: ${{ secrets.NOTION_API_TOKEN }}
            NOTION_PAGE_ID: ${{ vars.NOTION_PAGE_ID }}
            COMMENT: "${{ inputs.comment }}"
          with:
            script: |
              const notionToken = process.env.NOTION_API_TOKEN;
              const pageId = process.env.NOTION_PAGE_ID;
              const comment = process.env.COMMENT;
              
              if (!notionToken) {
                core.setFailed('NOTION_API_TOKEN secret is not configured');
                return;
              }
              if (!pageId) {
                core.setFailed('NOTION_PAGE_ID variable is not set');
                return;
              }
              if (!comment) {
                core.setFailed('comment is missing');
                return;
              }
              core.info(`Adding comment to Notion page: ${pageId}`);
              core.info(`Comment: ${comment}`);
              
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
<!--
## Notion Integration

This shared configuration provides Notion MCP server integration with read-only tools and a custom safe-job for adding comments to Notion pages.

### Configuration

- `NOTION_API_TOKEN` secret must be set in the repository settings with a Notion integration token that has access to the relevant pages/databases.
- `NOTION_PAGE_ID` environment variable must be set in the workflow or repository settings to specify the target Notion page for adding comments.

### Available Notion MCP Tools (Read-Only)

- `search_pages`: Search for Notion pages
- `get_page`: Get details of a specific page
- `get_database`: Get database schema
- `query_database`: Query database content

### Safe Job: notion-add-comment

The `notion-add-comment` safe-job allows the agentic workflow to add comments to Notion pages through the Notion API.
Requires the **insert comment** access on the token.

**Required Inputs:**
- `comment`: The comment text to add
-->