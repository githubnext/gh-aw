---
description: agentic-workflows MCP server tool reference for workflows that call status, logs, audit, or compile
---

# agentic-workflows MCP Server Tools

**⚠️ CRITICAL**: `status`, `logs`, `audit`, and `compile` are MCP server tools — NOT shell commands. Do NOT run `gh aw` directly. If the MCP server fails, give up.

## Tools

### `status`

Verify MCP server configuration and list all workflows.

No required parameters.

### `logs`

Download workflow run logs to `/tmp/gh-aw/aw-mcp/logs/`.

| Parameter | Description |
|---|---|
| `workflow_name` | Filter to a specific workflow (leave empty for all) |
| `count` | Number of runs (default: 100) |
| `start_date` | Filter runs after this date — `YYYY-MM-DD` or relative like `-1d`, `-7d`, `-30d` |
| `end_date` | Filter runs before this date |
| `engine` | Filter by AI engine: `copilot`, `claude`, `codex` |
| `branch` | Filter by branch name |
| `firewall` / `no_firewall` | Filter by firewall status |
| `filtered_integrity` | Only runs with DIFC integrity-filtered events in gateway logs |
| `after_run_id` / `before_run_id` | Paginate by run database ID |

### `audit`

Inspect a specific run in detail (missing tools, safe outputs, metrics).

| Parameter | Description |
|---|---|
| `run_id_or_url` | Run ID or run/job URL (including step anchors), as string or number |

### `compile`

Recompile workflow `.md` files into `.lock.yml` files.

**MCP equivalent of**: `gh aw compile`
