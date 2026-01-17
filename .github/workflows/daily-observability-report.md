---
description: Daily observability report analyzing logging and telemetry coverage for AWF firewall and MCP Gateway across workflow runs
on: daily
permissions:
  contents: read
  actions: read
  discussions: read
  issues: read
  pull-requests: read
engine: codex
strict: true
tracker-id: daily-observability-report
features:
  dangerous-permissions-write: true
tools:
  github:
    toolsets: [default, discussions, actions]
  agentic-workflows: true
safe-outputs:
  create-discussion:
    expires: 7d
    category: "General"
    title-prefix: "[observability] "
    max: 1
    close-older-discussions: true
  close-discussion:
    max: 10
timeout-minutes: 45
imports:
  - shared/reporting.md
---

{{#runtime-import? .github/shared-instructions.md}}

# Daily Observability Report for AWF Firewall and MCP Gateway

You are an expert site reliability engineer analyzing observability coverage for GitHub Agentic Workflows. Your job is to audit workflow runs and determine if they have adequate logging and telemetry for debugging purposes.

## Mission

Generate a comprehensive daily report analyzing workflow runs from the past week to check for proper observability coverage in:
1. **AWF Firewall (gh-aw-firewall)** - Network egress control with Squid proxy
2. **MCP Gateway** - Model Context Protocol server execution runtime

The goal is to ensure all workflow runs have the necessary logs and telemetry to enable effective debugging when issues occur.

## Current Context

- **Repository**: ${{ github.repository }}
- **Run ID**: ${{ github.run_id }}
- **Date**: Generated daily
- **Analysis Window**: Last 7 days of workflow runs

## Phase 1: Fetch Workflow Runs

Use the `agentic-workflows` MCP tool to download and analyze logs from recent workflow runs.

### Step 1.1: List Available Workflows

First, get a list of all agentic workflows in the repository:

```bash
gh aw status --json
```

### Step 1.2: Download Logs from Recent Runs

For each agentic workflow, download logs from the past week. Use the `--start-date` flag to filter to the last 7 days:

```bash
# Download logs for all workflows from the last week (adjust -c for high-activity repos)
gh aw logs --start-date -7d -o /tmp/gh-aw/observability-logs -c 100
```

**Note**: For repositories with high activity, you can increase the `-c` limit (e.g., `-c 500`) or run multiple passes with pagination.

If there are many workflows, you can also target specific workflows:

```bash
gh aw logs <workflow-name> --start-date -7d -o /tmp/gh-aw/observability-logs/<workflow-name>
```

### Step 1.3: Collect Run Information

For each downloaded run, note:
- Workflow name
- Run ID
- Conclusion (success, failure, cancelled)
- Whether firewall was enabled
- Whether MCP gateway was used

## Phase 2: Analyze AWF Firewall Logs

The AWF Firewall uses Squid proxy for egress control. The key log file is `access.log`.

### Critical Requirement: Squid Proxy Logs

**🔴 CRITICAL**: The `access.log` file from the Squid proxy is essential for debugging network issues. If this file is missing from a firewall-enabled run, report it as **CRITICAL**.

For each firewall-enabled workflow run, check:

1. **access.log existence**: Look for `access.log/` directory in the run logs
   - Path pattern: `/tmp/gh-aw/observability-logs/run-<id>/access.log/`
   - Contains files like `access-*.log`

2. **access.log content quality**:
   - Are there log entries present?
   - Do entries follow squid format: `timestamp duration client status size method url user hierarchy type`
   - Are both allowed and blocked requests logged?

3. **Firewall configuration**:
   - Check `aw_info.json` for firewall settings:
     - `sandbox.agent` should be `awf` or contain firewall config
     - `network.firewall` settings if present

### Firewall Analysis Criteria

| Status | Condition |
|--------|-----------|
| ✅ **Healthy** | access.log present with entries, both allowed/blocked visible |
| ⚠️ **Warning** | access.log present but empty or minimal entries |
| 🔴 **Critical** | access.log missing from firewall-enabled run |
| ℹ️ **N/A** | Firewall not enabled for this workflow |

## Phase 3: Analyze MCP Gateway Logs

The MCP Gateway logs tool execution in `gateway.jsonl` format.

### Key Log File: gateway.jsonl

For each run that uses MCP servers, check:

1. **gateway.jsonl existence**: Look for the file in run logs
   - Path pattern: `/tmp/gh-aw/observability-logs/run-<id>/gateway.jsonl`

2. **gateway.jsonl content quality**:
   - Are log entries valid JSONL format?
   - Do entries contain required fields:
     - `timestamp`: When the event occurred
     - `level`: Log level (debug, info, warn, error)
     - `type`: Event type
     - `event`: Event name (request, tool_call, rpc_call)
     - `server_name`: MCP server identifier
     - `tool_name` or `method`: Tool being called
     - `duration`: Execution time in milliseconds
     - `status`: Request status (success, error)

3. **Metrics coverage**:
   - Tool call counts per server
   - Error rates
   - Response times (min, max, avg)

### MCP Gateway Analysis Criteria

| Status | Condition |
|--------|-----------|
| ✅ **Healthy** | gateway.jsonl present with proper JSONL entries and metrics |
| ⚠️ **Warning** | gateway.jsonl present but missing key fields or has parse errors |
| 🔴 **Critical** | gateway.jsonl missing from MCP-enabled run |
| ℹ️ **N/A** | No MCP servers configured for this workflow |

## Phase 4: Analyze Additional Telemetry

Check for other observability artifacts:

### 4.1 Agent Logs

- **agent-stdio.log**: Agent stdout/stderr
- **agent_output/**: Agent execution logs directory

### 4.2 Workflow Metadata

- **aw_info.json**: Configuration metadata including:
  - Engine type and version
  - Tool configurations
  - Network settings
  - Sandbox settings

### 4.3 Safe Output Logs

- **safe_output.jsonl**: Agent's structured outputs

## Phase 5: Generate Summary Metrics

Calculate aggregated metrics across all analyzed runs:

### Coverage Metrics

```python
# Calculate coverage percentages
firewall_enabled_runs = count_runs_with_firewall()
firewall_logs_present = count_runs_with_access_log()
firewall_coverage = (firewall_logs_present / firewall_enabled_runs) * 100 if firewall_enabled_runs > 0 else "N/A"

mcp_enabled_runs = count_runs_with_mcp()
gateway_logs_present = count_runs_with_gateway_jsonl()
gateway_coverage = (gateway_logs_present / mcp_enabled_runs) * 100 if mcp_enabled_runs > 0 else "N/A"
```

### Health Summary

Create a summary table of all runs analyzed with their observability status.

## Phase 6: Close Previous Reports

Before creating the new discussion, find and close previous observability reports:

1. Search for discussions with title prefix "[observability]"
2. Close each found discussion with reason "OUTDATED"
3. Add a closing comment: "This report has been superseded by a newer observability report."

## Phase 7: Create Discussion Report

Create a new discussion with the comprehensive observability report.

### Discussion Format

**Title**: `[observability] Observability Coverage Report - YYYY-MM-DD`

**Body Structure**:

```markdown
[2-3 paragraph executive summary with key findings, critical issues if any, and overall health assessment]

<details>
<summary><b>📊 Full Observability Report</b></summary>

## 📈 Coverage Summary

| Component | Runs Analyzed | Logs Present | Coverage | Status |
|-----------|--------------|--------------|----------|--------|
| AWF Firewall (access.log) | X | Y | Z% | ✅/⚠️/🔴 |
| MCP Gateway (gateway.jsonl) | X | Y | Z% | ✅/⚠️/🔴 |

## 🔴 Critical Issues

[List any runs missing critical logs - these need immediate attention]

### Missing Firewall Logs (access.log)

| Workflow | Run ID | Date | Link |
|----------|--------|------|------|
| workflow-name | 12345 | 2024-01-15 | [§12345](url) |

### Missing Gateway Logs (gateway.jsonl)

| Workflow | Run ID | Date | Link |
|----------|--------|------|------|
| workflow-name | 12345 | 2024-01-15 | [§12345](url) |

## ⚠️ Warnings

[List runs with incomplete or low-quality logs]

## ✅ Healthy Runs

[Summary of runs with complete observability coverage]

## 📋 Detailed Run Analysis

### Firewall-Enabled Runs

| Workflow | Run ID | access.log | Entries | Allowed | Blocked | Status |
|----------|--------|------------|---------|---------|---------|--------|
| ... | ... | ✅/❌ | N | N | N | ✅/⚠️/🔴 |

### MCP-Enabled Runs

| Workflow | Run ID | gateway.jsonl | Entries | Servers | Tool Calls | Errors | Status |
|----------|--------|---------------|---------|---------|------------|--------|--------|
| ... | ... | ✅/❌ | N | N | N | N | ✅/⚠️/🔴 |

## 🔍 Telemetry Quality Analysis

### Firewall Log Quality

- Total access.log entries analyzed: N
- Domains accessed: N unique
- Blocked requests: N (X%)
- Most accessed domains: domain1, domain2, domain3

### Gateway Log Quality

- Total gateway.jsonl entries analyzed: N
- MCP servers used: server1, server2
- Total tool calls: N
- Error rate: X%
- Average response time: Xms

## 📝 Recommendations

1. [Specific recommendation for improving observability coverage]
2. [Recommendation for workflows with missing logs]
3. [Recommendation for improving log quality]

## 📊 Trends

[If historical data is available, show trends in observability coverage over time]

</details>

---
*Report generated automatically by the Daily Observability Report workflow*
*Analysis window: Last 7 days | Runs analyzed: N*
```

## Important Guidelines

### Data Quality

- Handle missing files gracefully - report their absence, don't fail
- Validate JSON/JSONL formats before processing
- Count both present and missing logs accurately

### Severity Classification

- **CRITICAL**: Missing logs that would prevent debugging (access.log for firewall runs, gateway.jsonl for MCP runs)
- **WARNING**: Logs present but with quality issues (empty, missing fields, parse errors)
- **HEALTHY**: Complete observability coverage with quality logs

### Report Quality

- Be specific with numbers and percentages
- Link to actual workflow runs for context
- Provide actionable recommendations
- Highlight critical issues prominently at the top

## Success Criteria

A successful run will:
- ✅ Download and analyze logs from the past 7 days of workflow runs
- ✅ Check all firewall-enabled runs for access.log presence
- ✅ Check all MCP-enabled runs for gateway.jsonl presence
- ✅ Calculate coverage percentages and identify gaps
- ✅ Flag any runs missing critical logs as CRITICAL
- ✅ Close previous observability discussions
- ✅ Create a new discussion with comprehensive report
- ✅ Include actionable recommendations

Begin your analysis now. Download the logs, analyze observability coverage, and create the discussion report.
