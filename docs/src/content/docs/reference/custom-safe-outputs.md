---
title: Custom Safe Outputs
description: Learn how to create custom safe outputs for third-party integrations using safe-jobs and MCP servers.
sidebar:
  order: 7
---

Custom safe outputs enable you to extend GitHub Agentic Workflows with your own output processing logic for third-party services like Notion, Slack, Jira, or any custom API. This guide demonstrates how to create reusable, secure integrations using safe-jobs combined with MCP servers.

## Overview

Custom safe outputs provide a pattern for integrating external services while maintaining security:

1. **Read-only MCP server** provides tools for querying data from the external service
2. **Custom safe-job** handles write operations through a separate job with appropriate permissions
3. **Shared configuration files** make integrations reusable across multiple workflows

This pattern ensures the main agentic job runs with minimal permissions while still enabling powerful integrations.

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│ Main Agentic Job (Read-Only Permissions)                   │
├─────────────────────────────────────────────────────────────┤
│ • Uses read-only MCP tools to query external service       │
│ • Analyzes data and makes decisions                        │
│ • Calls custom safe-job tool to perform write operations   │
└─────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────┐
│ Custom Safe-Job (Write Permissions)                        │
├─────────────────────────────────────────────────────────────┤
│ • Executes after main job completes                        │
│ • Has appropriate permissions for write operations         │
│ • Uses secure API calls to perform actions                 │
└─────────────────────────────────────────────────────────────┘
```

## Creating a Custom Safe Output

### Step 1: Define the MCP Server (Read-Only)

Create a shared configuration file with the MCP server for read operations:

```yaml
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
---
```

**Key Points:**
- Use `container:` for Docker-based MCP servers
- Use `command:` and `args:` for npx or local commands
- List only **read-only tools** in the `allowed` section
- Store sensitive tokens in GitHub Secrets

### Step 2: Define the Custom Safe-Job (Write Operations)

In the same shared configuration file, add a safe-job for write operations:

```yaml
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
      description: "Add a comment to a Notion page"
      runs-on: ubuntu-latest
      output: "Comment added to Notion successfully!"
      permissions:
        contents: read
      inputs:
        page_id:
          description: "The Notion page ID to add a comment to"
          required: true
          type: string
        comment:
          description: "The comment text to add"
          required: true
          type: string
      steps:
        - name: Add comment to Notion page
          uses: actions/github-script@v8
          env:
            NOTION_TOKEN: "${{ secrets.NOTION_TOKEN }}"
            PAGE_ID: "${{ inputs.page_id }}"
            COMMENT: "${{ inputs.comment }}"
          with:
            script: |
              const notionToken = process.env.NOTION_TOKEN;
              const pageId = process.env.PAGE_ID;
              const comment = process.env.COMMENT;
              
              if (!notionToken) {
                core.setFailed('NOTION_TOKEN secret is not configured');
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
                    parent: { page_id: pageId },
                    rich_text: [{
                      type: 'text',
                      text: { content: comment }
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
```

**Key Points:**
- `description:` field appears in the MCP tool registration
- `inputs:` section defines the tool's parameters (required for all safe-jobs)
- `output:` field provides custom success message
- Use `actions/github-script@v8` for JavaScript-based API calls
- Include error handling with `core.setFailed()`
- Store the configuration in `.github/workflows/shared/` for reusability

### Step 3: Use the Custom Safe Output in a Workflow

Import the shared configuration in your workflow:

```aw wrap
---
on:
  issues:
    types: [opened]
permissions:
  contents: read
  actions: read
engine: claude
imports:
  - shared/notion.md
---

# Issue Summary to Notion

Analyze the issue: "${{ needs.activation.outputs.text }}"

Search for the GitHub Issues page in Notion using the read-only Notion tools, then add a summary comment using the notion-add-comment safe-job.
```

**How It Works:**
1. The `imports:` directive loads the Notion MCP server and safe-job
2. The agent can use read-only Notion tools to search and query
3. The agent calls the `notion-add-comment` tool (registered from the safe-job)
4. The safe-job executes with appropriate permissions after the main job completes

## Best Practices

### Security

**✓ DO:**
- Use read-only tools in the MCP server
- Implement write operations in safe-jobs
- Validate all inputs in safe-job steps
- Use GitHub Secrets for API tokens
- Set minimal required permissions on safe-jobs

**✗ DON'T:**
- Give write permissions to the main agentic job
- Include write operations in MCP server tools
- Hardcode API tokens or credentials
- Skip input validation

### Error Handling

Always include comprehensive error handling in safe-jobs:

```javascript
try {
  const response = await fetch(apiUrl, options);
  
  if (!response.ok) {
    const errorData = await response.text();
    core.setFailed(`API error (${response.status}): ${errorData}`);
    return;
  }
  
  const data = await response.json();
  core.info('Operation successful');
} catch (error) {
  core.setFailed(`Failed to complete operation: ${error.message}`);
}
```

### Logging

Use appropriate logging levels:

```javascript
core.info('Informational message');
core.warning('Warning message');
core.error('Error message');
core.setFailed('Failure message that stops the job');
```

### Input Validation

Define clear input schemas:

```yaml
inputs:
  page_id:
    description: "The Notion page ID (UUID format)"
    required: true
    type: string
  comment:
    description: "Comment text (max 2000 characters)"
    required: true
    type: string
  priority:
    description: "Comment priority level"
    required: false
    type: choice
    options: ["low", "medium", "high"]
    default: "medium"
```

## Complete Examples

### Notion Integration

**File:** `.github/workflows/shared/notion.md`

```aw wrap
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
      description: "Add a comment to a Notion page"
      runs-on: ubuntu-latest
      output: "Comment added to Notion successfully!"
      permissions:
        contents: read
      inputs:
        page_id:
          description: "The Notion page ID to add a comment to"
          required: true
          type: string
        comment:
          description: "The comment text to add"
          required: true
          type: string
      steps:
        - name: Add comment to Notion page
          uses: actions/github-script@v8
          env:
            NOTION_TOKEN: "${{ secrets.NOTION_TOKEN }}"
            PAGE_ID: "${{ inputs.page_id }}"
            COMMENT: "${{ inputs.comment }}"
          with:
            script: |
              const notionToken = process.env.NOTION_TOKEN;
              const pageId = process.env.PAGE_ID;
              const comment = process.env.COMMENT;
              
              if (!notionToken) {
                core.setFailed('NOTION_TOKEN secret is not configured');
                return;
              }
              
              try {
                const response = await fetch('https://api.notion.com/v1/comments', {
                  method: 'POST',
                  headers: {
                    'Authorization': `Bearer ${notionToken}`,
                    'Notion-Version': '2022-06-28',
                    'Content-Type': 'application/json'
                  },
                  body: JSON.stringify({
                    parent: { page_id: pageId },
                    rich_text: [{ type: 'text', text: { content: comment } }]
                  })
                });
                
                if (!response.ok) {
                  throw new Error(`API error: ${response.status}`);
                }
                
                core.info('Comment added successfully');
              } catch (error) {
                core.setFailed(`Failed: ${error.message}`);
              }
---
