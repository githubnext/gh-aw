<agentic-workflows-guide>
**⚠️ CRITICAL**: Use `status`, `logs`, `audit`, `compile` as MCP tools — NOT shell commands. Do NOT run `gh aw` directly. If the MCP server fails, give up.

- `status`: verify configuration and list workflows
- `logs`: download run logs to `/tmp/gh-aw/aw-mcp/logs/`; params: `workflow_name`, `count`, `start_date`, `end_date`, `engine`, `branch`
- `audit`: inspect a specific run; param: `run_id_or_url`
</agentic-workflows-guide>
