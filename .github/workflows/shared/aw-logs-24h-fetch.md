---
# Pre-fetch last 24 hours of agentic workflow logs for analysis
# Saves logs to /tmp/gh-aw/aw-mcp/logs/

tools:
  agentic-workflows:
  cache-memory: true
  timeout: 300

steps:
  - name: Download logs from last 24 hours
    env:
      GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}
    run: ./gh-aw logs --start-date -1d -o /tmp/gh-aw/aw-mcp/logs
---

## Agentic Workflow Logs (Last 24h)

Workflow logs have been pre-downloaded to `/tmp/gh-aw/aw-mcp/logs/`.

**IMPORTANT**: Do NOT run `./gh-aw` or `gh aw` CLI commands directly — the binary is not authenticated in the agent environment. Use the `agentic-workflows` MCP server tools (`status`, `logs`, `audit`) instead for all additional queries.

### Log Directory Structure

```
/tmp/gh-aw/aw-mcp/logs/
└── run-(id)/           # One directory per workflow run
    ├── aw_info.json    # Run metadata (engine, workflow, status, tokens)
    ├── activation/     # Activation job logs
    └── agent/          # Agent job logs
```
