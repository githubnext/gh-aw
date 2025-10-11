---
title: Custom Safe Outputs
description: Learn how to create custom safe outputs for third-party integrations using safe-jobs and MCP servers.
sidebar:
  order: 700
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
  - shared/mcp/notion.md
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

## Safe Jobs Technical Specification

The `safe-outputs.jobs:` element of your workflow's frontmatter enables you to define custom post-processing jobs that execute after the main agentic workflow completes. Safe-jobs provide a powerful way to create sophisticated automation workflows while maintaining security through controlled job execution.

**How It Works:**
1. Safe-jobs are defined under the `safe-outputs.jobs` section of the frontmatter
2. Each safe-job **must have an "inputs" section with at least one input** - these inputs become the arguments of the MCP tool
3. Each safe-job has access to all standard GitHub Actions job properties (runs-on, if, needs, env, permissions, steps)
4. Safe-jobs automatically download the agent output artifact and can process it using jq
5. Safe-jobs become available as callable tools in the safe-outputs MCP server
6. Safe-jobs can be imported from included workflows with automatic conflict detection

**Important Requirement**: Every safe-job definition must include an `inputs` section with at least one input parameter. These inputs define the MCP tool's arguments and enable the agentic workflow to call the safe-job with appropriate parameters.

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

**Every safe-job must define inputs** using workflow_dispatch syntax for parameterization. The inputs become the MCP tool arguments:

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

Safe-jobs can return custom response messages via the MCP server:

```yaml
safe-outputs:
  jobs:
    notify:
      runs-on: ubuntu-latest
      output: "Notification sent successfully!"
      inputs:
        message:
          description: "Notification message"
          required: true
          type: string
      steps:
        - name: Send notification
          run: echo "Sending notification..."
```

### Agent Output Processing

Safe-jobs automatically receive access to the agent output artifact:

```yaml
safe-outputs:
  jobs:
    analyze:
      runs-on: ubuntu-latest
      inputs:
        data_type:
          description: "Type of data to analyze"
          required: true
          type: string
      steps:
        - name: Process agent output
          run: |
            if [ -f "$GITHUB_AW_AGENT_OUTPUT" ]; then
              # Extract specific data from agent output
              RESULT=$(cat "$GITHUB_AW_AGENT_OUTPUT" | jq -r 'select(.tool == "analyze") | .result')
              echo "Agent analysis result: $RESULT"
            else
              echo "No agent output available"
            fi
```

### Include Support

Safe-jobs can be imported from included workflows with automatic conflict detection:

**Main workflow:**
```aw wrap
---
safe-outputs:
  jobs:
    deploy:
      runs-on: ubuntu-latest
      inputs:
        target:
          description: "Deployment target"
          required: true
          type: string
      steps:
        - name: Deploy
          run: echo "Deploying..."
---

@import shared/common-jobs.md
```

**Imported file (shared/common-jobs.md):**
```aw wrap
---
safe-outputs:
  jobs:
    test:
      runs-on: ubuntu-latest
      inputs:
        suite:
          description: "Test suite to run"
          required: true
          type: string
      steps:
        - name: Test
          run: echo "Testing..."
---
```

**Result:** Both `deploy` and `test` safe-jobs are available.

**Conflict Detection:** If both files define a safe-job with the same name, compilation fails with:
```
failed to merge safe-jobs: safe-job name conflict: 'deploy' is defined in both main workflow and included files
```

### MCP Server Integration

Safe-jobs are automatically registered as tools in the safe-outputs MCP server, allowing the agentic workflow to call them:

```yaml
safe-outputs:
  jobs:
    database-backup:
      runs-on: ubuntu-latest
      inputs:
        database:
          description: "Database to backup"
          required: true
          type: string
      steps:
        - name: Backup database
          run: echo "Backing up database..."
```

The agent can then call this safe-job:
```
Please backup the user database using the database-backup safe-job.
```

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
