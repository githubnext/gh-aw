---
title: MCP Server
description: Use the gh-aw MCP server to expose CLI tools to AI agents via Model Context Protocol, enabling secure workflow management.
sidebar:
  order: 400
---

The `gh aw mcp-server` command exposes `gh aw` CLI tools (status, compile, logs, audit) to AI agents through the Model Context Protocol. The MCP server enables AI agents to:
- Check workflow status and compile workflows
- Download and analyze workflow logs
- Investigate workflow run failures

Start the server for local CLI usage:

```bash
gh aw mcp-server
```

Or configure in for any host:
```yaml
command: gh
args: [aw, mcp-server]
```

## Configuration Options

### Using a Custom Command Path

Use the `--cmd` flag to specify a custom path to the gh-aw binary instead of using the default `gh aw` command:

```bash
gh aw mcp-server --cmd ./gh-aw
```

This is useful when:
- Running a local build of gh-aw for development
- Using a specific version of gh-aw in CI/CD workflows
- Running the MCP server in environments where the gh CLI extension is not available

Example in an agentic workflow:
```yaml
steps:
  - name: Build gh-aw
    run: make build
  - name: Start MCP server
    run: |
      set -e
      ./gh-aw mcp-server --cmd ./gh-aw --port 8765 &
      MCP_PID=$!
      sleep 2
      if ! kill -0 $MCP_PID 2>/dev/null; then
        echo "MCP server failed to start"
        exit 1
      fi
```

### HTTP Server Mode

Use the `--port` flag to run the server with HTTP/SSE transport instead of stdio:

```bash
gh aw mcp-server --port 8080
```

## Available Tools

The MCP server provides these tools:

- **status** - List workflows with optional pattern filter
- **compile** - Compile workflows to GitHub Actions YAML
- **logs** - Download workflow logs (saved to `/tmp/gh-aw/aw-mcp/logs`)
- **audit** - Generate detailed workflow run report (saved to `/tmp/gh-aw/aw-mcp/logs`)

## Example Prompt

```markdown
Check all workflows in this repository:

1. Use `status` to list workflows
2. Use `logs` to get recent runs (last 5 for each workflow)
3. Use `audit` to investigate any failures
4. Generate a summary report

```

## Using as Agentic Workflows Tool

The `gh aw mcp-server` is available as a builtin tool called `agentic-workflows` in agentic workflows. This enables AI agents to analyze GitHub Actions workflow traces and improve workflows based on execution history.

### Configuration

Add the `agentic-workflows` tool to your workflow frontmatter:

```yaml
---
on:
  schedule:
    - cron: "0 9 * * 1"  # Weekly analysis
tools:
  agentic-workflows:  # Enable workflow introspection
  github:
    allowed: [get_workflow_run, list_workflow_runs]
safe-outputs:
  create-issue:
    title-prefix: "[workflow-analysis] "
    labels: [automation, ci-analysis]
permissions:
  contents: read
  actions: read
---

# Weekly Workflow Health Report

Analyze all workflow runs from the past week and create a health report.

Use the agentic-workflows tool to:
1. Download logs from the last 7 days using `logs` with `--start-date -1w`
2. Audit any failed runs to understand failure patterns
3. Check workflow status to ensure all are properly compiled

Create an issue with:
- Summary of workflow execution statistics
- Common failure patterns and root causes
- Recommendations for improving reliability
- Token usage and cost analysis
```

### Automatic Installation

When the `agentic-workflows` tool is enabled, the generated workflow automatically includes a step to install the gh-aw extension:

```yaml
- name: Install gh-aw extension
  run: |
    echo "Installing gh-aw extension..."
    gh extension install githubnext/gh-aw || gh extension upgrade githubnext/gh-aw
    gh aw --version
```

### Common Use Cases

#### 1. Weekly Workflow Health Reports

Monitor all workflows weekly and create summary issues:

```yaml
---
on:
  schedule:
    - cron: "0 9 * * 1"
tools:
  agentic-workflows:
  github:
safe-outputs:
  create-issue:
    title-prefix: "[health-report] "
---

# Workflow Health Monitor

Analyze the past week's workflow executions and report on:
- Success/failure rates
- Common error patterns
- Performance trends
- Cost analysis
```

#### 2. Failure Investigation

Automatically investigate workflow failures:

```yaml
---
on:
  workflow_run:
    types: [completed]
tools:
  agentic-workflows:
  github:
safe-outputs:
  add-comment:
---

# Failure Investigator

When a workflow fails:
1. Use `audit` to analyze the failure
2. Identify root cause
3. Add diagnostic comment to related issue/PR
```

#### 3. Performance Monitoring

Track workflow performance over time:

```yaml
---
on:
  schedule:
    - cron: "0 */6 * * *"
tools:
  agentic-workflows:
safe-outputs:
  create-issue:
    title-prefix: "[performance] "
    max: 1
---

# Performance Monitor

Monitor execution times, token usage, and costs.
Create alerts if performance degrades >20% or costs spike.
```

### Best Practices

- **Use with GitHub tools**: Combine `agentic-workflows` with the `github` tool to cross-reference workflow data with issues and PRs
- **Use safe-outputs**: Always use `safe-outputs` for creating issues or comments instead of write permissions
- **Set appropriate schedules**: Don't run analysis workflows too frequently to avoid API usage
- **Focus on actionable insights**: Have agents provide specific recommendations, not just raw data
- **Filter strategically**: Use date filters and workflow name filters to focus on relevant runs

### Permissions

The tool requires minimal permissions:

```yaml
permissions:
  contents: read  # To read workflow files
  actions: read   # To access workflow run data
```

If creating issues or comments, use `safe-outputs` which handles write permissions in a separate job.

### Examples

See the complete example workflow at [`.github/workflows/example-workflow-analyzer.md`](https://github.com/githubnext/gh-aw/blob/main/.github/workflows/example-workflow-analyzer.md) in the repository.

