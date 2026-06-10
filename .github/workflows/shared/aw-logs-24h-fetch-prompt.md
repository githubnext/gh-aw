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
