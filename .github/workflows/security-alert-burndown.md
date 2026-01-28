---
name: Security Alert Burndown
description: Discovers Dependabot PRs and assigns them to Copilot for review
on:
  #schedule:
  #  - cron: "0 * * * *"
  workflow_dispatch:
permissions:
  issues: read
  pull-requests: read
  contents: read
tools:
  github:
    toolsets: [repos, issues, pull_requests]
safe-outputs:
  noop:
    max: 1
  update-project:
    max: 10
project:
  url: https://github.com/orgs/githubnext/projects/144
---

# Security Alert Burndown - Simple Discovery Workflow

This workflow discovers Dependabot-created pull requests for JavaScript dependencies in the githubnext/gh-aw repository and updates the project board with their status.

## Task

You need to discover and update Dependabot pull requests on the project board. Follow these steps:

### Step 1: Discover Dependabot PRs

Use the GitHub MCP server to search for pull requests in the `githubnext/gh-aw` repository with:
- Author: `app/dependabot`
- Labels: `dependencies`, `javascript`
- State: open

Example search query:
```
repo:githubnext/gh-aw is:pr author:app/dependabot label:dependencies label:javascript is:open
```

### Step 2: Check for Work

If no pull requests are found:
- Call the `noop` tool with message: "No Dependabot JavaScript PRs found to process"
- Exit successfully

### Step 3: Update Project Board

For each discovered PR (up to 10):
- Add or update the PR on the project board: https://github.com/orgs/githubnext/projects/144
- Use the `update-project` safe output tool
- Set fields:
  - `campaign_id`: "security-alert-burndown"
  - `status`: "Todo" (for open PRs)
  - `target_repo`: "githubnext/gh-aw"
  - `worker_workflow`: "unknown"
  - `priority`: "Medium"
  - `size`: "Small"
  - `start_date`: PR created date (YYYY-MM-DD format)
  - `end_date`: Today's date (YYYY-MM-DD format)

### Step 4: Report

Summarize how many PRs were discovered and added/updated on the project board.

## Important

- Always use the `update-project` tool for project board updates
- If no work is found, call `noop` to indicate successful completion with no actions
- Focus only on open PRs - closed/merged PRs should be ignored
- Limit updates to 10 PRs per run to respect rate limits
