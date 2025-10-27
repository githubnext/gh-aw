---
title: Custom Safe Outputs
description: Learn how to create custom safe outputs for third-party integrations using safe-jobs and MCP servers.
sidebar:
  order: 700
---

Custom safe outputs extend GitHub Agentic Workflows with your own output processing logic for third-party services. Create reusable, secure integrations using safe-jobs combined with MCP servers.

## Architecture

The pattern separates read and write operations for security: a read-only MCP server queries external services, while a custom safe-job handles write operations with appropriate permissions. Store configuration in shared files for reusability across workflows.

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

Use `container:` for Docker-based servers or `command:`/`args:` for npx commands. List only read-only tools in `allowed` and store tokens in GitHub Secrets.

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

The `description` appears in MCP tool registration. All safe-jobs require an `inputs` section defining parameters. Use `output` for custom success messages and `actions/github-script@v8` for API calls with `core.setFailed()` error handling. Store configurations in `.github/workflows/shared/` for reusability.

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

The `imports` directive loads both the MCP server and safe-job. The agent uses read-only tools to query, then calls the safe-job tool which executes with write permissions after completion.

## Safe Jobs Technical Specification

Define custom post-processing jobs under `safe-outputs.jobs` in your workflow frontmatter. Each safe-job requires an `inputs` section (these become MCP tool arguments), supports all GitHub Actions job properties, automatically receives agent output artifacts, and registers as a callable tool in the safe-outputs MCP server. Safe-jobs can be imported with automatic conflict detection.

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

Every safe-job must define inputs using workflow_dispatch syntax. The inputs become the MCP tool arguments:

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
          if [ -f "$GH_AW_AGENT_OUTPUT" ]; then
            ENV=$(cat "$GH_AW_AGENT_OUTPUT" | jq -r 'select(.tool == "deploy") | .environment // "staging"')
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
            if [ -f "$GH_AW_AGENT_OUTPUT" ]; then
              # Extract specific data from agent output
              RESULT=$(cat "$GH_AW_AGENT_OUTPUT" | jq -r 'select(.tool == "analyze") | .result')
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

Always include error handling with `core.setFailed()` for API failures. Use appropriate logging levels: `core.info()`, `core.warning()`, `core.error()`, and `core.setFailed()` to stop jobs on failure.

## Example: MCP Diagnostic Reporting

The `mcp-debug` shared workflow provides a `report_diagnostics_to_pull_request` safe-job that accepts diagnostic messages, finds the associated pull request, and posts comments. Import with `shared/mcp-debug.md` to enable workflows to report diagnostic information without requiring write permissions in the main job.

## Related Documentation

- [Safe Outputs](/gh-aw/reference/safe-outputs/) - Built-in safe output types
- [MCPs](/gh-aw/guides/mcps/) - Model Context Protocol setup
- [Frontmatter](/gh-aw/reference/frontmatter/) - All configuration options
