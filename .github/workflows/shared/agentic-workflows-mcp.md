---
# Agentic Workflows MCP Server
# Enables the agentic-workflows tool that provides status/logs/audit access
#
# Usage:
#   imports:
#     - shared/agentic-workflows-mcp.md

tools:
  agentic-workflows:
---

## Agentic Workflows MCP Server

The `agentic-workflows` MCP server is configured for this workflow.

**⚠️ IMPORTANT**: Do NOT run `./gh-aw` or `gh aw` commands directly in bash — the binary is NOT authenticated in the agent environment. Always use the MCP server tools below instead.

### Available Tools

- **`status`** — List all agentic workflows in the repository with their last run status
- **`logs`** — Download workflow run logs. Key parameters:
  - `workflow_name` — Specific workflow (or omit for all)
  - `count` — Number of runs (default: 20)
  - `start_date` — Filter by date (e.g., `"-1d"`, `"-7d"`, `"-30d"`)
  - `engine` — Filter by AI engine (`"copilot"`, `"claude"`, `"codex"`)
  - `json` — Return structured JSON summary
- **`audit`** — Deep-dive investigation of a specific run by `run_id`

### Common Patterns

```
# List all workflow statuses
Use the status tool

# Get logs from last 24 hours
Use the logs tool with start_date: "-1d"

# Investigate a specific failed run
Use the audit tool with run_id: "12345"
```
