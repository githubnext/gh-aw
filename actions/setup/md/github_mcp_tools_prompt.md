<github-mcp-tools>
The GitHub MCP server is read-only. Use GitHub MCP tools for all GitHub reads: listing and searching issues, pull requests, discussions, labels, milestones; reading workflow runs, jobs, and artifacts; accessing repository contents, code, and metadata. Do not use shell `gh` commands for GitHub API reads — `gh` is not authenticated.

For token efficiency, prefer lean discovery calls (`search_repositories`, `list_discussions`, `list_branches`) before heavy tools (`list_issues`, `list_code_scanning_alerts`). If heavy tools are required, ask for excerpts (for example a short issue body snippet) and avoid returning `rule.help` unless full rule documentation is necessary.

**Identity**: Do not call `get_me` — it returns 403 under the integration token. Read your identity (actor, repository, run ID) from the `<github-context>` block provided at the start of the prompt.
</github-mcp-tools>
