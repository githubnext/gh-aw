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
          with:
            script: |
              const fs = require('fs');
              const notionToken = process.env.NOTION_API_TOKEN;
              const pageId = process.env.NOTION_PAGE_ID;
              const isStaged = process.env.GH_AW_SAFE_OUTPUTS_STAGED === 'true';
              const outputContent = process.env.GH_AW_AGENT_OUTPUT;
              
              if (!notionToken) {
                core.setFailed('NOTION_API_TOKEN secret is not configured');
                return;
              }
              if (!pageId) {
                core.setFailed('NOTION_PAGE_ID variable is not set');
                return;
              }
              
              // Read and parse agent output
              if (!outputContent) {
                core.info('No GH_AW_AGENT_OUTPUT environment variable found');
                return;
              }
              
              let agentOutputData;
              try {
                const fileContent = fs.readFileSync(outputContent, 'utf8');
                agentOutputData = JSON.parse(fileContent);
              } catch (error) {
                core.setFailed(`Error reading or parsing agent output: ${error instanceof Error ? error.message : String(error)}`);
                return;
              }
              
              if (!agentOutputData.items || !Array.isArray(agentOutputData.items)) {
                core.info('No valid items found in agent output');
                return;
              }
              
              // Filter for notion_add_comment items
              const notionCommentItems = agentOutputData.items.filter(item => item.type === 'notion_add_comment');
              
              if (notionCommentItems.length === 0) {
                core.info('No notion_add_comment items found in agent output');
                return;
              }
              
              core.info(`Found ${notionCommentItems.length} notion_add_comment item(s)`);
              
              // Process each comment item
              for (let i = 0; i < notionCommentItems.length; i++) {
                const item = notionCommentItems[i];
                const comment = item.comment;
                
                if (!comment) {
                  core.warning(`Item ${i + 1}: Missing comment field, skipping`);
                  continue;
                }
                
                if (isStaged) {
                  let summaryContent = "## 🎭 Staged Mode: Notion Comment Preview\n\n";
                  summaryContent += "The following comment would be added to Notion if staged mode was disabled:\n\n";
                  summaryContent += `**Page ID:** ${pageId}\n\n`;
                  summaryContent += `**Comment:**\n${comment}\n\n`;
                  await core.summary.addRaw(summaryContent).write();
                  core.info("📝 Notion comment preview written to step summary");
                  continue;
                }
                
                core.info(`Adding comment ${i + 1}/${notionCommentItems.length} to Notion page: ${pageId}`);
                
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
                  core.info(`✅ Comment ${i + 1} added successfully`);
                  core.info(`Comment ID: ${data.id}`);
                } catch (error) {
                  core.setFailed(`Failed to add comment ${i + 1}: ${error instanceof Error ? error.message : String(error)}`);
                  return;
                }
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

The `notion_add_comment` safe-job allows the agentic workflow to add comments to Notion pages through the Notion API.
Requires the **insert comment** access on the token.

**Required Inputs:**
- `comment`: The comment text to add
-->