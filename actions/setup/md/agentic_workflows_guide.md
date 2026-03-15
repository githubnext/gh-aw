<agentic-workflows-guide>
## Using the agentic-workflows MCP Server

**⚠️ CRITICAL**: The `status`, `logs`, `audit`, `list`, and `compile` operations are MCP server tools,
NOT shell commands. Do NOT run `gh aw` directly — it is not authenticated in this context.
Do not attempt to download or build the `gh aw` extension. If the MCP server fails, give up.
Call all operations as MCP tools with JSON parameters.

- Run the `status` tool to verify configuration and list all workflows
- Use the `logs` tool to download run logs (saves to `/tmp/gh-aw/aw-mcp/logs/`)
- Use the `audit` tool with a `run_id` to investigate specific runs
- Use the `list` tool to enumerate workflows in the repository

### Tool Parameters

#### `status` — Verify MCP server configuration and list workflows

#### `logs` — Download workflow run logs
- `workflow_name`: filter to a specific workflow (leave empty for all)
- `count`: number of runs (default 30, max 5000)
- `start_date`: relative date, e.g. `-1d`, `-7d`, `-30d`
- `engine`: filter by AI engine (`copilot`, `claude`, `codex`)
- Logs are saved to `/tmp/gh-aw/aw-mcp/logs/`

#### `audit` — Inspect a specific run
- `run_id`: GitHub Actions run ID (numeric)
</agentic-workflows-guide>
