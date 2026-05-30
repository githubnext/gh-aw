## Issue Deduplication

**Always use the `create_issue` safe-output tool (not the GitHub MCP `create_issue` tool) when reporting failures or problems.**

The `create_issue` safe-output tool has built-in deduplication:

- **`group-by-day`**: If an open issue with the same title was already created today, it posts the new content as a comment on the existing issue instead of creating a duplicate
- **`close-older-issues`**: Automatically closes older issues with the same title prefix, keeping the issue queue clean

### When to create a failure issue

Only create an issue if you encounter a genuine problem that a maintainer should know about (e.g., data fetch failure, unexpected workflow error). Do not create an issue for normal "no changes needed" runs — use the `noop` safe-output for those.

### How to report a failure

Use the `create_issue` safe-output tool (JSON format):

```json
{"create_issue": {"title": "Brief description of the failure", "body": "### What failed\n\nData fetch or processing step failed with an unexpected error.\n\n### Steps to investigate\n\n1. Check the workflow run logs for the exact error message\n2. Verify that the data sources (community_issues.json, pull_requests.json) are accessible\n3. Review recent changes to the workflow or data sources"}}
```

The built-in `close-older-issues` and `group-by-day` settings ensure that repeated failures on the same problem are grouped into a single issue thread rather than creating new duplicate issues each run.
