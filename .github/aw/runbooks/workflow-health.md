# Workflow Health Monitoring Runbook

## Common Workflow Failure Patterns

### Missing Tool Configurations

Errors like "missing-tool" or "tool not found"; agent cannot perform GitHub operations. Cause: GitHub MCP server not configured in frontmatter, missing/incorrect `toolsets:`.

### Authentication and Permission Errors

HTTP 403, "Resource not accessible", or token scope errors. Cause: missing or insufficient `permissions:` block; GITHUB_TOKEN not passed to custom actions.

### Input/Secret Validation Failures

MCP Scripts action fails; env var unavailable; template expression errors. Cause: MCP Scripts not configured, missing required secrets, or incorrect secret references.

## Investigation Steps

### Step 1: Analyze Workflow Logs

Download and analyze with `gh aw logs`:

> **Note**: Run these commands locally or from a Copilot coding agent session. If you include `gh aw logs` / `gh aw audit` as workflow steps, add `actions: read` to `permissions:` and install the extension with the `setup-cli` action first — see [Logs and Metrics](../github-agentic-workflows.md#logs-and-metrics).

```bash
# Download logs from last 24 hours
gh aw logs --start-date -1d -o /tmp/workflow-logs

# Download logs for a specific workflow run
gh aw logs --run-id <run-id> -o /tmp/workflow-logs

# Analyze logs for a specific workflow
gh aw logs --workflow <workflow-name> --start-date -7d
```

Look in the "Run AI Agent" step for missing-tool errors, HTTP codes (401/403/404/500), and stack traces.

### Step 2: Identify Missing-Tool Errors

Missing-tool errors look like:

```
Error: Tool 'github:read_issue' not found
Error: missing tool configuration for mcpscripts-gh
```

Check the workflow's `tools:` section, compare with similar working workflows, and verify the tool is configured in frontmatter.

### Step 3: Verify MCP Server Configurations

```aw
---
tools:
  github:
    toolsets: [default]   # Enables repos, issues, pull_requests
---
```

Verify with:

```bash
gh aw mcp inspect <workflow-name>   # Inspect MCP servers for a workflow
gh aw mcp list                       # List all workflows with MCP servers
```

### Step 4: Check Permissions Configuration

```aw
---
permissions:
  contents: read       # repository files
  issues: write        # create/update issues
  pull-requests: write # create/update PRs
  actions: read        # access workflow runs
---
```

Match the permission scope to the operation: read for queries, write for create/update.

## Resolution Procedures

### Adding GitHub MCP Server to Workflows

Add `tools.github.toolsets` to the workflow frontmatter, then `gh aw compile <workflow-name>.md` and `gh aw mcp inspect <workflow-name>` to verify.

**Available toolsets**: `default` (repos + issues + pull_requests + common), `repos`, `issues`, `pull_requests`, `actions`.

**Example**: Dev workflow with GitHub MCP server

```aw
---
description: Development workflow with GitHub integration
on:
  workflow_dispatch:
permissions:
  contents: read
  issues: read
  pull-requests: read
engine: copilot
tools:
  github:
    toolsets: [default]
---

# Development Agent

Analyze repository issues and provide insights.
```

### Configuring MCP Scripts and Safe-Outputs

MCP Scripts pass GitHub context to the agent as env vars:

```aw
---
mcp-scripts:
  issue:
    title: ${{ github.event.issue.title }}
    body: ${{ github.event.issue.body }}
    number: ${{ github.event.issue.number }}
---
```

Safe-outputs let the agent create GitHub resources:

```aw
---
safe-outputs:
  create-issue:
    labels: ["ai-generated"]
  create-pull-request:
    labels: ["ai-generated"]
  create-discussion:
    category: "general"
---
```

### Testing Workflow Fixes

```bash
gh aw compile <workflow-name>.md                              # compile
gh workflow run <workflow-name>.lock.yml                      # trigger (if workflow_dispatch)
gh run list --workflow=<workflow-name>.lock.yml --limit 1     # get run ID
gh run watch <run-id>                                         # monitor
gh aw logs --run-id <run-id>                                  # logs on failure
```

Verify: no missing-tool errors, agent completes, expected resources created.

## Case Study: DeepReport Incident Response

Three failures fixed:

- **Weekly Issue Summary** — missing `actions: read` permission.
- **Dev Workflow** — "Tool 'github:read_issue' not found": added `tools.github.toolsets: [default]`.
- **Daily Copilot PR Merged** — "missing tool configuration for mcpscripts-gh": added `mcp-scripts.pull_request` with `number`/`title` from `github.event.pull_request`.

## Common Configuration Patterns

**Basic GitHub integration**:
```aw
---
permissions:
  contents: read
  issues: read
tools:
  github:
    toolsets: [default]
---
```

**Issue-triggered workflow with mcp-scripts**:
```aw
---
on:
  issues:
    types: [opened]
permissions:
  contents: read
  issues: write
mcp-scripts:
  issue:
    title: ${{ github.event.issue.title }}
    body: ${{ github.event.issue.body }}
tools:
  github:
    toolsets: [default]
---
```

**Workflow with safe-outputs**:
```aw
---
permissions:
  contents: read
  issues: write
  discussions: write
safe-outputs:
  create-issue:
    labels: ["ai-generated"]
  create-discussion:
    category: "general"
tools:
  github:
    toolsets: [default]
---
```

