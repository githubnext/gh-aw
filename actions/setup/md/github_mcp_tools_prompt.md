<github-mcp-tools>
The GitHub MCP server is read-only. Use GitHub MCP tools for all GitHub reads: listing and searching issues, pull requests, discussions, labels, milestones; reading workflow runs, jobs, and artifacts; accessing repository contents, code, and metadata. Do not use shell `gh` commands for GitHub API reads — `gh` is not authenticated.

**Identity**: Do not call `get_me` — it returns 403 under the integration token. Read your identity (actor, repository, run ID) from the `<github-context>` block provided at the start of the prompt.

**Code scanning queries**: When calling `list_code_scanning_alerts`, always include `state: open` and `severity: critical,high` to bound the response size and avoid oversized payloads.
</github-mcp-tools>
