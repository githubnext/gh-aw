---
name: "Example: GitHub Projects URL Workaround"
description: Demonstrates workarounds for obtaining issue/PR URLs from GitHub Projects
on:
  workflow_dispatch:

permissions:
  contents: read
  issues: read
  pull-requests: read

tools:
  github:
    toolsets: [projects, issues, pull_requests]
    github-token: ${{ secrets.GITHUB_TOKEN }}

safe-outputs:
  missing-data:
    create-issue: true
    title-prefix: "[data gap]"
    labels: [data-quality]
---

# GitHub Projects URL Workaround Example

This workflow demonstrates how to work with GitHub Projects when the `content.url` field is not available from the `list_project_items` tool.

## Your Task

Generate a report of all items in a GitHub Project board, including clickable links to each issue or PR.

**Project URL**: https://github.com/orgs/githubnext/projects/66

## Instructions

Since the GitHub MCP `list_project_items` tool does not return the `content.url` field, you need to use a workaround approach:

### Approach 1: Fetch Details Separately (Recommended)

1. **List project items** using the `list_project_items` tool
2. **For each item** that has `content_type` of "Issue" or "PullRequest":
   - Extract the `content_number` (issue/PR number)
   - Extract repository information (owner and repo name)
   - Use `issue_read` (with method="get") or `pull_request_read` (with method="get") to get full details
   - The full response will include the URL

3. **Generate your report** with the URLs you obtained

### Approach 2: Manual URL Construction (Fallback)

If you have the repository owner, repo name, and content number from the project item:

- Issues: `https://github.com/{owner}/{repo}/issues/{number}`
- Pull Requests: `https://github.com/{owner}/{repo}/pull/{number}`

Only use this if you can verify the repository information from the project item metadata.

### Approach 3: Report Missing Data

If you cannot obtain URLs using either approach above, report the data gap:

Use the `missing_data` safe output type with:

```json
{
  "type": "missing_data",
  "data_type": "project_item_urls",
  "reason": "GitHub MCP list_project_items does not return content.url field",
  "context": "Needed to generate report with clickable issue links for project board",
  "alternatives": "Attempted to fetch details separately using issue_read/pull_request_read"
}
```

## Expected Output

Generate a markdown report showing:

1. Project name and URL
2. List of all items with:
   - Item title
   - Item type (Issue/PR/Draft)
   - Status
   - **Full GitHub URL** (using one of the workaround approaches)
3. Summary statistics

## Notes

- This workflow demonstrates best practices for handling data gaps in GitHub MCP tools
- The `missing_data` safe output encourages honest reporting rather than hallucination
- Future versions of the GitHub MCP server may add the `content.url` field directly
