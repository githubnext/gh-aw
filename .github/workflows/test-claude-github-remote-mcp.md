---
on:
  workflow_dispatch:
  reaction: eyes

engine: 
  id: claude

safe-outputs:
  create-issue:

network: {}

tools:
  github-remote:
    mcp:
      type: http
      url: "https://api.githubcopilot.com/mcp/"
      headers:
        Authorization: "Bearer ${{ secrets.GITHUB_TOKEN }}"
        Content-Type: "application/json"
    allowed: ["*"]
---

**First, use the GitHub remote MCP tools to get information about this repository and current user.**

Try to use available GitHub remote MCP tools to:
1. Get current user information
2. Get repository information 
3. List recent issues or pull requests

Then create an issue with title "Hello from Claude via GitHub Remote MCP" and include in the body:
- What tools were available from the remote MCP
- What repository information you were able to retrieve
- Whether you were successful in using the remote GitHub MCP tools

### AI Attribution

Include this footer in your issue description:

```markdown
> AI-generated content by [${{ github.workflow }}](https://github.com/${{ github.repository }}/actions/runs/${{ github.run_id }}) may contain mistakes.
```