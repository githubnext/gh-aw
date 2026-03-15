<agentic-workflows-guide>
## Using the agentic-workflows MCP Server

**⚠️ CRITICAL**: The `status`, `logs`, `audit`, and `compile` operations are MCP server tools,
NOT shell commands. Do NOT run `gh aw` directly — it is not authenticated in this context.
Do not attempt to download or build the `gh aw` extension. If the MCP server fails, give up.
Call all operations as MCP tools with JSON parameters.

- Run the `status` tool to verify configuration and list all workflows
- Use the `logs` tool to download run logs (saves to `/tmp/gh-aw/aw-mcp/logs/`)
- Use the `audit` tool with a run ID or URL to investigate specific runs

### Tool Parameters

#### `status` — Verify MCP server configuration and list workflows

#### `logs` — Download workflow run logs
- `workflow_name`: filter to a specific workflow (leave empty for all)
- `count`: number of runs (default: 100)
- `start_date`: filter runs after this date (YYYY-MM-DD or relative like `-1d`, `-7d`, `-30d`)
- `end_date`: filter runs before this date
- `engine`: filter by AI engine (`copilot`, `claude`, `codex`)
- `branch`: filter by branch name
- `firewall` / `no_firewall`: filter by firewall status
- `after_run_id` / `before_run_id`: paginate by run database ID
- Logs are saved to `/tmp/gh-aw/aw-mcp/logs/`

#### `audit` — Inspect a specific run
- `run_id_or_url`: numeric run ID, run URL, job URL, or job URL with step anchor
</agentic-workflows-guide>
