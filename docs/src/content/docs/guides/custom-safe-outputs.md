---
title: Custom Safe Outputs
description: How to create custom safe outputs for third-party integrations using custom jobs and MCP servers.
sidebar:
  order: 5
---

Custom safe outputs extend GitHub Agentic Workflows beyond built-in GitHub operations. While built-in safe outputs handle GitHub issues, PRs, and discussions, custom safe outputs let you integrate with third-party services like Notion, Slack, databases, or any external API.

## When to Use Custom Safe Outputs

Use custom safe outputs when you need to:

- Send data to external services (Slack, Discord, Notion, Jira)
- Trigger deployments or CI/CD pipelines
- Update databases or external storage
- Call custom APIs that require authentication
- Perform any write operation that built-in safe outputs don't cover

## Quick Start

Here's a minimal custom safe output that sends a Slack message:

```yaml wrap title=".github/workflows/shared/slack-notify.md"
---
safe-outputs:
  jobs:
    slack-notify:
      description: "Send a message to Slack"
      runs-on: ubuntu-latest
      output: "Message sent to Slack!"
      inputs:
        message:
          description: "The message to send"
          required: true
          type: string
      steps:
        - name: Send Slack message
          env:
            SLACK_WEBHOOK: "${{ secrets.SLACK_WEBHOOK }}"
            MESSAGE: "${{ inputs.message }}"
          run: |
            # Use jq to safely escape JSON content
            PAYLOAD=$(jq -n --arg text "$MESSAGE" '{text: $text}')
            curl -X POST "$SLACK_WEBHOOK" \
              -H 'Content-Type: application/json' \
              -d "$PAYLOAD"
---
```

Use it in a workflow:

```aw wrap title=".github/workflows/issue-notifier.md"
---
on:
  issues:
    types: [opened]
permissions:
  contents: read
imports:
  - shared/slack-notify.md
---

# Issue Notifier

A new issue was opened: "${{ needs.activation.outputs.text }}"

Summarize the issue and use the slack-notify tool to send a notification.
```

The agent can now call `slack-notify` with a message, and the custom job executes with access to the `SLACK_WEBHOOK` secret.

## Architecture

Custom safe outputs separate read and write operations for security:

1. **Read operations**: Use MCP servers configured with `allowed:` lists limiting to read-only tools
2. **Write operations**: Use custom jobs that run after the agent completes, with appropriate permissions

This separation ensures the agent cannot directly access secrets or perform write operations—only the custom job can, and only after the agent explicitly calls the safe output tool.

```
┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│   Agent (AI)    │────▶│  MCP Server     │────▶│  External API   │
│                 │     │  (read-only)    │     │  (GET requests) │
└─────────────────┘     └─────────────────┘     └─────────────────┘
        │
        │ calls safe-job tool
        ▼
┌─────────────────┐     ┌─────────────────┐
│  Custom Job     │────▶│  External API   │
│  (with secrets) │     │  (POST/PUT)     │
└─────────────────┘     └─────────────────┘
```

## Creating a Custom Safe Output

### Step 1: Define the MCP Server (Read-Only)

Create a shared configuration file with the MCP server for read operations:

```yaml wrap
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

### Step 2: Define the Custom Job (Write Operations)

In the same shared configuration file, add a custom job under `safe-outputs.jobs` for write operations:

```yaml wrap
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

The `description` appears in MCP tool registration. All custom jobs require an `inputs` section defining parameters. Use `output` for custom success messages and `actions/github-script@v8` for API calls with `core.setFailed()` error handling. Store configurations in `.github/workflows/shared/` for reusability.

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

imports:
  - shared/mcp/notion.md
---

# Issue Summary to Notion

Analyze the issue: "${{ needs.activation.outputs.text }}"

Search for the GitHub Issues page in Notion using the read-only Notion tools, then add a summary comment using the notion-add-comment safe-job.
```

The `imports` directive loads both the MCP server and safe-job. The agent uses read-only tools to query, then calls the safe-job tool which executes with write permissions after completion.

## Safe Job Reference

Custom jobs are defined under `safe-outputs.jobs` in your workflow frontmatter. Each job becomes an MCP tool that the agent can call.

### Job Properties

| Property | Type | Required | Description |
|----------|------|----------|-------------|
| `description` | string | Yes | Tool description shown to the agent |
| `runs-on` | string | Yes | GitHub Actions runner (e.g., `ubuntu-latest`) |
| `inputs` | object | Yes | Tool parameters (see [Input Types](#input-types)) |
| `steps` | array | Yes | GitHub Actions steps to execute |
| `output` | string | No | Success message returned to the agent |
| `permissions` | object | No | GitHub token permissions for the job |
| `env` | object | No | Environment variables for all steps |
| `if` | string | No | Conditional execution expression |
| `timeout-minutes` | number | No | Maximum job duration (default: 360) |

### Input Types

Every custom job must define `inputs`. These become the tool's parameters that the agent provides when calling the tool.

| Type | Description | Example |
|------|-------------|---------|
| `string` | Text input | `"production"` |
| `boolean` | True/false toggle | `"true"` or `"false"` |
| `choice` | Selection from options | `["staging", "production"]` |

```yaml wrap
inputs:
  # Required string input
  message:
    description: "Message content"
    required: true
    type: string

  # Optional boolean with default
  notify:
    description: "Send notification"
    required: false
    type: boolean
    default: "true"

  # Choice from predefined options
  environment:
    description: "Target environment"
    required: true
    type: choice
    options: ["staging", "production"]
```

:::note
All input values are passed as strings. For booleans, use `"true"` or `"false"` string values and parse them in your steps.
:::

### Environment Variables

Custom jobs automatically receive these environment variables:

| Variable | Description |
|----------|-------------|
| `GH_AW_AGENT_OUTPUT` | Path to JSON file containing agent output |
| `GH_AW_SAFE_OUTPUTS_STAGED` | Set to `"true"` in staged mode |
| `${{ inputs.NAME }}` | Each input parameter as a variable |

### Accessing Inputs in Steps

Access inputs using GitHub Actions expressions:

```yaml wrap
steps:
  - name: Use inputs
    env:
      MESSAGE: "${{ inputs.message }}"
      ENV: "${{ inputs.environment }}"
    run: |
      echo "Message: $MESSAGE"
      echo "Environment: $ENV"
```

For JavaScript steps with `actions/github-script@v8`, access inputs via environment variables:

```yaml wrap
steps:
  - name: Process with JavaScript
    uses: actions/github-script@v8
    env:
      MESSAGE: "${{ inputs.message }}"
    with:
      script: |
        const message = process.env.MESSAGE;
        core.info(`Received: ${message}`);
```

## Complete Examples

### Simple Shell-Based Job

```yaml wrap title="Send a webhook notification"
safe-outputs:
  jobs:
    webhook-notify:
      description: "Send a notification to a webhook URL"
      runs-on: ubuntu-latest
      output: "Notification sent!"
      inputs:
        title:
          description: "Notification title"
          required: true
          type: string
        body:
          description: "Notification body"
          required: true
          type: string
      steps:
        - name: Send webhook
          env:
            WEBHOOK_URL: "${{ secrets.WEBHOOK_URL }}"
            TITLE: "${{ inputs.title }}"
            BODY: "${{ inputs.body }}"
          run: |
            # Use jq to safely escape JSON content
            PAYLOAD=$(jq -n --arg title "$TITLE" --arg body "$BODY" \
              '{title: $title, body: $body}')
            curl -X POST "$WEBHOOK_URL" \
              -H "Content-Type: application/json" \
              -d "$PAYLOAD"
```

### JavaScript-Based Job with Error Handling

```yaml wrap title="Create a Jira ticket"
safe-outputs:
  jobs:
    create-jira-ticket:
      description: "Create a ticket in Jira"
      runs-on: ubuntu-latest
      output: "Jira ticket created!"
      inputs:
        summary:
          description: "Ticket summary"
          required: true
          type: string
        description:
          description: "Ticket description"
          required: true
          type: string
        priority:
          description: "Ticket priority"
          required: false
          type: choice
          options: ["Low", "Medium", "High"]
        issue_type:
          description: "Jira issue type"
          required: false
          type: string
          default: "Task"
      steps:
        - name: Create Jira ticket
          uses: actions/github-script@v8
          env:
            JIRA_URL: "${{ secrets.JIRA_URL }}"
            JIRA_TOKEN: "${{ secrets.JIRA_TOKEN }}"
            JIRA_PROJECT: "${{ vars.JIRA_PROJECT }}"
            SUMMARY: "${{ inputs.summary }}"
            DESCRIPTION: "${{ inputs.description }}"
            PRIORITY: "${{ inputs.priority }}"
            ISSUE_TYPE: "${{ inputs.issue_type }}"
          with:
            script: |
              const jiraUrl = process.env.JIRA_URL;
              const jiraToken = process.env.JIRA_TOKEN;
              const project = process.env.JIRA_PROJECT;
              
              // Validate required secrets
              if (!jiraUrl || !jiraToken) {
                core.setFailed('JIRA_URL and JIRA_TOKEN secrets are required');
                return;
              }
              
              const payload = {
                fields: {
                  project: { key: project },
                  issuetype: { name: process.env.ISSUE_TYPE || 'Task' },
                  summary: process.env.SUMMARY,
                  description: process.env.DESCRIPTION,
                  priority: { name: process.env.PRIORITY || 'Medium' }
                }
              };
              
              try {
                // Uses Jira REST API v2 (check your instance version)
                const response = await fetch(`${jiraUrl}/rest/api/2/issue`, {
                  method: 'POST',
                  headers: {
                    'Authorization': `Basic ${jiraToken}`,
                    'Content-Type': 'application/json'
                  },
                  body: JSON.stringify(payload)
                });
                
                if (!response.ok) {
                  const error = await response.text();
                  core.setFailed(`Jira API error: ${error}`);
                  return;
                }
                
                const data = await response.json();
                core.info(`Created ticket: ${data.key}`);
                core.setOutput('ticket_key', data.key);
              } catch (error) {
                core.setFailed(`Failed to create ticket: ${error.message}`);
              }
```

### Job with Read-Only MCP Server

Combine a read-only MCP server with a write-capable custom job for complete third-party integrations:

```yaml wrap title=".github/workflows/shared/notion-integration.md"
---
# Read-only MCP server for querying Notion
mcp-servers:
  notion:
    container: "mcp/notion"
    env:
      NOTION_TOKEN: "${{ secrets.NOTION_TOKEN }}"
    allowed:
      - "search_pages"
      - "get_page"
      - "query_database"

# Write-capable custom job for appending to Notion pages
safe-outputs:
  jobs:
    notion-append-content:
      description: "Append content to a Notion page"
      runs-on: ubuntu-latest
      output: "Content appended to page!"
      inputs:
        page_id:
          description: "Notion page ID to append content to"
          required: true
          type: string
        content:
          description: "Content to append to the page"
          required: true
          type: string
      steps:
        - name: Append content to Notion page
          uses: actions/github-script@v8
          env:
            NOTION_TOKEN: "${{ secrets.NOTION_TOKEN }}"
            PAGE_ID: "${{ inputs.page_id }}"
            CONTENT: "${{ inputs.content }}"
          with:
            script: |
              // Note: This appends a new paragraph to the page
              // To update existing content, use the Update Block API
              const response = await fetch(
                `https://api.notion.com/v1/blocks/${process.env.PAGE_ID}/children`,
                {
                  method: 'PATCH',
                  headers: {
                    'Authorization': `Bearer ${process.env.NOTION_TOKEN}`,
                    'Notion-Version': '2022-06-28',
                    'Content-Type': 'application/json'
                  },
                  body: JSON.stringify({
                    children: [{
                      type: 'paragraph',
                      paragraph: {
                        rich_text: [{ type: 'text', text: { content: process.env.CONTENT } }]
                      }
                    }]
                  })
                }
              );
              
              if (!response.ok) {
                core.setFailed(`Notion error: ${await response.text()}`);
                return;
              }
              core.info('Content appended successfully');
---
```

## Importing Custom Jobs

Custom jobs can be defined in shared files and imported into workflows.

### Basic Import

```aw wrap
---
on: issues
permissions:
  contents: read
imports:
  - shared/slack-notify.md
  - shared/jira-integration.md
---

# Issue Handler

Handle the issue and notify via Slack and Jira.
```

### Conflict Detection

If two imported files define jobs with the same name, compilation fails:

```
failed to merge safe-jobs: safe-job name conflict: 'notify' is defined in both main workflow and included files
```

Rename one of the jobs to resolve the conflict.

## Best Practices

### Error Handling

Always handle errors gracefully using GitHub Actions' `core` utilities:

```javascript
// Check required inputs
if (!process.env.API_KEY) {
  core.setFailed('API_KEY secret is not configured');
  return;
}

// Handle API errors
try {
  const response = await fetch(url);
  if (!response.ok) {
    core.setFailed(`API error (${response.status}): ${await response.text()}`);
    return;
  }
  // Success
  core.info('Operation completed successfully');
} catch (error) {
  core.setFailed(`Request failed: ${error.message}`);
}
```

### Logging Levels

Use appropriate logging levels for visibility:

| Function | When to Use |
|----------|-------------|
| `core.debug()` | Detailed debugging info (hidden by default) |
| `core.info()` | Normal operation messages |
| `core.warning()` | Non-fatal issues |
| `core.error()` | Error messages (doesn't stop the job) |
| `core.setFailed()` | Fatal errors (stops the job) |

### Security

- Store all secrets in GitHub Secrets, never hardcode them
- Use environment variables to pass secrets to steps
- Limit permissions to only what the job needs
- Validate all inputs before using them

### Staged Mode Support

Custom jobs should respect staged mode by checking `GH_AW_SAFE_OUTPUTS_STAGED`:

```javascript
const isStaged = process.env.GH_AW_SAFE_OUTPUTS_STAGED === 'true';
const notificationText = process.env.MESSAGE;

if (isStaged) {
  core.info('🎭 Staged mode: would send notification');
  await core.summary.addRaw('## Preview\nWould send: ' + notificationText).write();
  return;
}

// Actually send the notification
```

## Troubleshooting

### Job Not Appearing as a Tool

**Problem**: The agent doesn't see your custom job as a callable tool.

**Solutions**:
1. Ensure `inputs` section is defined—jobs without inputs aren't registered as tools
2. Check that `description` is set—this is the tool description shown to the agent
3. Verify the import path is correct in your workflow
4. Run `gh aw compile` to check for configuration errors

### Secrets Not Available

**Problem**: Environment variables from secrets are empty.

**Solutions**:
1. Verify the secret exists in repository settings
2. Check the secret name matches exactly (case-sensitive)
3. Ensure the secret is referenced correctly: `"${{ secrets.SECRET_NAME }}"`

### Job Fails Silently

**Problem**: The job exits without error but nothing happens.

**Solutions**:
1. Add `core.info()` logging to trace execution
2. Check `if (!response.ok)` conditions are handled
3. Ensure `core.setFailed()` is called on errors
4. Review job logs in the GitHub Actions run

### Agent Calls Wrong Tool

**Problem**: The agent calls a different safe output instead of your custom job.

**Solutions**:
1. Make the `description` more specific and unique
2. In your workflow prompt, explicitly mention the custom job name
3. Use distinct naming that doesn't overlap with built-in safe outputs

## Related Documentation

- [Safe Outputs](/gh-aw/reference/safe-outputs/) - Built-in safe output types
- [MCPs](/gh-aw/guides/mcps/) - Model Context Protocol setup
- [Frontmatter](/gh-aw/reference/frontmatter/) - All configuration options
- [Imports](/gh-aw/reference/imports/) - Sharing workflow configurations
