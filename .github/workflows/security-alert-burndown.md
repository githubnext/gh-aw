---
name: Security Alert Burndown
description: Discovers security work items (Dependabot PRs, code scanning alerts, secret scanning alerts)
on:
  #schedule:
  #  - cron: "0 * * * *"
  workflow_dispatch:
permissions:
  issues: read
  pull-requests: read
  contents: read
  security-events: read
tools:
  github:
    read-only: true
    toolsets: [repos, issues, pull_requests, code_security, secret_protection]
project: https://github.com/orgs/githubnext/projects/144
---

# Security Alert Burndown

This workflow discovers security alert work items in the githubnext/gh-aw repository and updates the project board with their status:

- Dependabot-created PRs for JavaScript dependency updates
- Open code scanning alerts
- Open secret scanning alerts

## Task

You need to discover and update security work items on the project board. Follow these steps:

### Step 1: Discover Dependabot PRs

Use the GitHub MCP server to search for pull requests in the `githubnext/gh-aw` repository with:
- Author: `app/dependabot`
- Labels: `dependencies`, `javascript`
- State: open

Example search query:
```
repo:githubnext/gh-aw is:pr author:app/dependabot label:dependencies label:javascript is:open
```

### Step 2: Discover Code Scanning Alerts

Use the GitHub MCP server to list **open** code scanning alerts in `githubnext/gh-aw`.

- Tool: `list_code_scanning_alerts` (GitHub MCP `code_security` toolset)
- Parameters:
  - `owner`: `githubnext`
  - `repo`: `gh-aw`
  - `state`: `open`

From results, collect each alert’s:
- Alert number/id
- Severity (if available)
- Created date
- URL

### Step 3: Discover Secret Scanning Alerts

Use the GitHub MCP server to list **open** secret scanning alerts in `githubnext/gh-aw`.

- Tool: `list_secret_scanning_alerts` (GitHub MCP `secret_protection` toolset)
- Parameters:
  - `owner`: `githubnext`
  - `repo`: `gh-aw`
  - `state`: `open`

From results, collect each alert’s:
- Alert number/id
- Secret type (if available)
- Created date
- URL

### Step 4: Check for Work

If *no* items were found across all categories (Dependabot PRs, code scanning alerts, secret scanning alerts):
- Call the `noop` tool with message: "No security alerts found to process"
- Exit successfully

### Step 5: Update Project Board

For each discovered item (up to 100 total per run):
- Add or update the corresponding work item on the project board: <https://github.com/orgs/githubnext/projects/144>
- Use the `update-project` safe output tool
- Always include the campaign project URL (this is what makes it a campaign):
  - `project`: "<https://github.com/orgs/githubnext/projects/144>"
- Always include the content identity:
  - `content_type`: `pull_request` (Dependabot PRs) or `issue` (tracking issues)
  - `content_number`: PR/issue number
- Set fields:
  - `campaign_id`: "security-alert-burndown"
  - `status`: "Todo" (for open items)
  - `target_repo`: "githubnext/gh-aw"
  - `worker_workflow`: who discovered it, using one of:
    - "dependabot"
    - "code-scanning"
    - "secret-scanning"
  - `priority`: "Medium"
  - `size`: "Small"
  - `start_date`: Item created date (YYYY-MM-DD format)
  - `end_date`: Today's date (YYYY-MM-DD format)

Notes:
- `update-project` requires an existing GitHub **issue** or **pull request** reference (`content_type` + `content_number`).
- For alerts, prefer updating an existing tracking issue/PR if one exists; otherwise, report that the alert has no tracking item yet.

### Step 6: Report

Summarize how many items were discovered and added/updated on the project board, broken down by category.

## Important

- Always use the `update-project` tool for project board updates
- If no work is found, call `noop` to indicate successful completion with no actions
- Focus only on open items:
  - PRs: open only
  - Alerts: open only
- Limit updates to 100 items per run to respect rate limits (prioritize highest severity/most recent first)
