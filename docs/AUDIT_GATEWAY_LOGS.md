# Auditing Gateway Logs in Smoke-Copilot Workflow Runs

This guide explains how to audit smoke-copilot workflow runs and validate that MCP gateway logs are properly stored in artifacts.

## Overview

The smoke-copilot workflow is a critical testing workflow that validates the Copilot engine functionality. It uses various tools including the GitHub MCP server, which may route traffic through an MCP gateway. Gateway logs are essential for:

- Debugging MCP server communication issues
- Understanding request/response patterns
- Identifying authentication or network problems
- Analyzing performance bottlenecks

## Prerequisites

1. **GitHub CLI** (`gh`) installed and authenticated
2. **gh-aw** binary built and available in your PATH
3. Access to the `githubnext/gh-aw` repository

## Step-by-Step Audit Process

### Step 1: Find Recent Smoke-Copilot Runs

List recent runs of the smoke-copilot workflow:

```bash
gh run list \
  --repo githubnext/gh-aw \
  --workflow=smoke-copilot.md \
  --limit=10 \
  --json databaseId,status,conclusion,createdAt,displayTitle
```

Expected output:
```json
[
  {
    "createdAt": "2026-01-09T04:00:00Z",
    "databaseId": 12345678,
    "displayTitle": "Smoke Copilot",
    "conclusion": "success",
    "status": "completed"
  },
  ...
]
```

### Step 2: Select a Run to Audit

Choose a run ID from the list above (e.g., `12345678`).

### Step 3: Run the Audit Command

Use the `gh-aw audit` command to download artifacts and analyze the run:

```bash
./gh-aw audit 12345678 --output .github/aw/logs --parse -v
```

This command will:
- Download all artifacts from the specified run
- Extract logs and analyze them
- Generate a report in `.github/aw/logs/run-12345678/`
- Parse agent and firewall logs (with `--parse` flag)

### Step 4: Validate Gateway Logs Exist

Check that MCP logs (including gateway logs) were downloaded:

```bash
RUN_ID=12345678
RUN_DIR=".github/aw/logs/run-${RUN_ID}"

# Check for MCP logs directory
if [ -d "${RUN_DIR}/mcp-logs" ]; then
  echo "✓ MCP logs directory found"
  ls -la "${RUN_DIR}/mcp-logs"
else
  echo "✗ MCP logs directory NOT found"
fi
```

Expected artifacts structure:
```
.github/aw/logs/run-12345678/
├── agent-stdio.log          # Agent execution logs
├── aw_info.json             # Workflow metadata
├── mcp-logs/                # MCP server logs directory
│   ├── github-*.log         # GitHub MCP server logs
│   ├── gateway-*.log        # Gateway logs (if gateway is used)
│   └── ...
├── sandbox/
│   └── firewall/
│       └── logs/            # Network firewall logs
└── summary.json             # Audit summary
```

### Step 5: Analyze Gateway Logs

If gateway logs are present, examine them for errors:

```bash
# Find gateway log files
find "${RUN_DIR}/mcp-logs" -name "*gateway*" -type f

# Check for errors in gateway logs
for log in ${RUN_DIR}/mcp-logs/gateway-*.log; do
  if [ -f "$log" ]; then
    echo "Analyzing: $log"
    
    # Look for error patterns
    grep -i "error\|fatal\|fail\|panic\|exception" "$log" || echo "No errors found"
    
    # Check log size
    ls -lh "$log"
    
    # Show first/last 20 lines
    echo "First 20 lines:"
    head -20 "$log"
    echo ""
    echo "Last 20 lines:"
    tail -20 "$log"
  fi
done
```

### Step 6: Common Issues to Look For

When analyzing gateway logs, look for:

1. **Connection Errors**
   ```
   ERROR: Failed to connect to MCP server
   ERROR: Connection timeout
   ERROR: Connection refused
   ```

2. **Authentication Failures**
   ```
   ERROR: Invalid API key
   ERROR: Authentication failed
   ERROR: Unauthorized access
   ```

3. **Protocol Errors**
   ```
   ERROR: Invalid JSON-RPC request
   ERROR: Method not found
   ERROR: Invalid parameters
   ```

4. **Performance Issues**
   ```
   WARN: Request took longer than 5s
   WARN: Queue depth exceeds threshold
   ```

## Automated Audit Script

Use the provided script for automated auditing:

```bash
./scripts/audit-smoke-copilot.sh
```

This script automates all the steps above and provides a comprehensive report.

## Expected Gateway Log Format

Gateway logs should contain structured JSON entries like:

```json
{
  "timestamp": "2026-01-09T04:00:00Z",
  "level": "info",
  "msg": "Request received",
  "method": "tools/list",
  "server": "github",
  "request_id": "abc-123",
  "duration_ms": 42
}
```

## Validation Checklist

- [ ] MCP logs directory exists in artifacts
- [ ] Gateway log files are present (if gateway is configured)
- [ ] Gateway logs contain structured entries
- [ ] No critical errors in gateway logs
- [ ] Request/response patterns look normal
- [ ] Authentication succeeded
- [ ] Response times are reasonable (< 5s for most requests)

## Troubleshooting

### Gateway Logs Missing

If gateway logs are not found in artifacts:

1. **Check workflow configuration**: Verify that the workflow uses an MCP gateway
   ```bash
   # Check if smoke-copilot uses MCP gateway
   grep -i "gateway\|mcp.*gateway" .github/workflows/smoke-copilot.md
   ```

2. **Check artifact upload configuration**: Verify that MCP logs are uploaded
   ```bash
   # Look for artifact upload steps
   grep -A 5 "upload.*artifact" .github/workflows/smoke-copilot.lock.yml | grep mcp
   ```

3. **Check for feature flags**: Gateway may require a feature flag
   ```bash
   # Check for mcp-gateway feature flag
   grep -i "mcp-gateway.*feature" .github/workflows/smoke-copilot.md
   ```

### Gateway Errors Found

If errors are found in gateway logs:

1. **Authentication issues**: Check that secrets are properly configured
2. **Network issues**: Verify network permissions in workflow configuration
3. **Server startup failures**: Check container logs and initialization errors
4. **Protocol mismatches**: Verify MCP server compatibility

## Related Documentation

- [MCP Gateway Specification](../docs/src/content/docs/reference/mcp-gateway.md)
- [Artifact File Locations](../specs/artifacts.md)
- [MCP Logs Guardrails](../specs/mcp_logs_guardrails.md)
- [Workflow Health Monitoring Runbook](../specs/runbooks/workflow-health-monitoring.md)

## GitHub MCP Server Tools

You can also use GitHub MCP server tools to query workflow runs programmatically:

```javascript
// Using GitHub MCP server tools
const tools = {
  list_workflow_runs: {
    owner: "githubnext",
    repo: "gh-aw",
    resource_id: "smoke-copilot.md",
    method: "list_workflow_runs"
  },
  
  download_workflow_run_artifact: {
    owner: "githubnext",
    repo: "gh-aw",
    method: "download_workflow_run_artifact",
    run_id: "12345678",
    output_directory: ".github/aw/logs/run-12345678"
  }
}
```

These tools are available when using the GitHub MCP server in agentic workflows.
