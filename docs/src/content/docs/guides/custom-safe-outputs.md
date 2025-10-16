---
title: Custom Safe Outputs
description: Learn how to create custom safe outputs for third-party integrations using safe-jobs and MCP servers.
sidebar:
  order: 700
---

Custom safe outputs enable you to extend GitHub Agentic Workflows with your own output processing logic for third-party services like Notion, Slack, Jira, or any custom API. This guide demonstrates how to create reusable, secure integrations using safe-jobs combined with MCP servers.

## Overview

Custom safe outputs integrate external services securely by separating read and write operations:

- **Read-only MCP server** queries external service data
- **Custom safe-job** handles write operations with appropriate permissions in a separate job
- **Shared configuration files** enable reusable integrations

This pattern maintains minimal permissions in the main agentic job while enabling powerful integrations.

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

Use `container:` for Docker-based MCP servers or `command:`/`args:` for npx/local commands. List only read-only tools in `allowed` and store sensitive tokens in GitHub Secrets.

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

The `description:` appears in MCP tool registration. The required `inputs:` section defines tool parameters. Use `output:` for custom success messages. Include error handling with `core.setFailed()` and store configurations in `.github/workflows/shared/` for reusability.

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
  - shared/mcp/notion.md
---

# Issue Summary to Notion

Analyze the issue: "${{ needs.activation.outputs.text }}"

Search for the GitHub Issues page in Notion using the read-only Notion tools, then add a summary comment using the notion-add-comment safe-job.
```

The `imports:` directive loads the MCP server and safe-job. The agent uses read-only tools to query data, then calls the safe-job tool which executes with appropriate permissions after the main job completes.

## Safe Jobs Technical Specification

Define custom post-processing jobs under `safe-outputs.jobs:` in your workflow frontmatter. Safe-jobs execute after the main agentic workflow completes and provide secure, controlled automation.

**Requirements**: Each safe-job must have an `inputs` section with at least one input parameter. These inputs become the MCP tool's arguments. Safe-jobs support all standard GitHub Actions job properties (runs-on, if, needs, env, permissions, steps), automatically download agent output artifacts, register as callable tools in the safe-outputs MCP server, and can be imported from included workflows with automatic conflict detection.

### GitHub Actions Job Properties

Safe-jobs support all standard GitHub Actions job properties:

```yaml
safe-outputs:
  jobs:
    deploy:
      runs-on: ubuntu-latest
      if: github.event.issue.number
      timeout-minutes: 30
      permissions:
        contents: write
        deployments: write
      env:
        DEPLOY_ENV: production
      inputs:
        confirm:
          description: "Confirm deployment"
          required: true
          type: boolean
          default: "false"
    steps:
      - name: Deploy
        run: echo "Deploying..."
```

### Safe Job Inputs

Safe-jobs require inputs defined using workflow_dispatch syntax. The inputs become MCP tool arguments:

```yaml
safe-outputs:
  jobs:
    deploy:
      runs-on: ubuntu-latest
      inputs:
        environment:
          description: "Target deployment environment"
          required: true
          type: choice
          options: ["staging", "production"]
        force:
          description: "Force deployment even if tests fail"
          required: false
          type: boolean
          default: "false"
    steps:
      - name: Deploy application
        run: |
          if [ -f "$GITHUB_AW_AGENT_OUTPUT" ]; then
            ENV=$(cat "$GITHUB_AW_AGENT_OUTPUT" | jq -r 'select(.tool == "deploy") | .environment // "staging"')
            echo "Deploying to $ENV"
          fi
```

### Custom Output Messages

Use the `output:` field to return custom response messages:

```yaml
output: "Notification sent successfully!"
```

### Agent Output Processing

Safe-jobs automatically receive the agent output artifact via `$GITHUB_AW_AGENT_OUTPUT`. Extract data using jq:

```bash
RESULT=$(cat "$GITHUB_AW_AGENT_OUTPUT" | jq -r 'select(.tool == "analyze") | .result')
```

### Include Support

Import safe-jobs from shared files using `@import shared/common-jobs.md`. Both main and imported safe-jobs become available. Name conflicts fail compilation with a clear error message.

### MCP Server Integration

Safe-jobs automatically register as callable tools in the safe-outputs MCP server. For example, defining a `database-backup` safe-job with a `database` input allows the agent to call it naturally: "Please backup the user database using the database-backup safe-job."

## Best Practices

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
