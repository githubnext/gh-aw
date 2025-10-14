---
description: Agentic Workflows Tool for Workflow Analysis
applyTo: ".github/workflows/*.md,.github/workflows/**/*.md"
---

# Agentic Workflows Tool

The `agentic-workflows` tool is a builtin MCP server that enables AI agents to introspect and analyze GitHub Actions workflow execution history. This tool is particularly useful for creating workflows that monitor, analyze, and improve other workflows based on their execution traces.

## Overview

When you include `agentic-workflows:` in your workflow's tools section, the AI agent gains access to powerful workflow analysis capabilities through the `gh aw mcp-server` command.

## Available Tools

The agentic-workflows MCP server provides four main tools:

### 1. `status`
Show the compilation status and GitHub Actions state of all workflow files in the repository.

**Use cases:**
- Check which workflows are properly compiled
- Identify workflows with compilation errors
- Get an overview of all agentic workflows in the repository

### 2. `compile`
Programmatically compile markdown workflow files to YAML.

**Use cases:**
- Validate workflow syntax
- Test workflow compilation after making changes
- Ensure workflows are properly formatted

### 3. `logs`
Download and analyze workflow run logs with advanced filtering options.

**Parameters:**
- `workflow_name`: Filter by specific workflow
- `count`: Number of runs to analyze
- `start_date`: Filter runs from a specific date (supports delta time like "-1w")
- `end_date`: Filter runs until a specific date
- `engine`: Filter by AI engine (claude, copilot, codex)
- `branch`: Filter by git branch

**Use cases:**
- Analyze recent workflow failures
- Track performance over time
- Compare execution across different engines
- Monitor token usage and costs

### 4. `audit`
Investigate specific workflow run failures and generate detailed diagnostic reports.

**Parameters:**
- `run_id`: The workflow run ID to audit

**Use cases:**
- Deep dive into failed runs
- Identify root causes of failures
- Generate comprehensive failure reports
- Understand error patterns

## Configuration

### Basic Usage

```yaml
tools:
  agentic-workflows:  # or agentic-workflows: true
```

This is all you need! The tool has no additional configuration options.

### Automatic Installation

When the `agentic-workflows` tool is enabled, the generated workflow automatically includes a step to install the gh-aw extension:

```yaml
- name: Install gh-aw extension
  run: |
    echo "Installing gh-aw extension..."
    gh extension install githubnext/gh-aw || gh extension upgrade githubnext/gh-aw
    gh aw --version
```

This ensures the MCP server has access to the gh-aw CLI commands.

## Common Use Cases

### 1. Weekly Workflow Health Report

Create a workflow that runs weekly and analyzes all workflow executions:

```yaml
---
on:
  schedule:
    - cron: "0 9 * * 1"  # Monday 9AM
tools:
  agentic-workflows:
  github:
    allowed: [get_workflow_run, list_workflow_runs]
safe-outputs:
  create-issue:
    title-prefix: "[workflow-health] "
    labels: [automation, ci-analysis]
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

### 2. Failure Investigation Bot

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
    max: 1
---

# Workflow Failure Investigator

When a workflow fails, analyze the failure and provide insights.

If the workflow run failed:
1. Use `audit` with the run ID to get detailed diagnostics
2. Identify the root cause from error logs
3. Add a comment to the associated issue/PR with findings and suggestions
```

### 3. Performance Monitoring

Track workflow performance over time:

```yaml
---
on:
  schedule:
    - cron: "0 */6 * * *"  # Every 6 hours
tools:
  agentic-workflows:
safe-outputs:
  create-issue:
    title-prefix: "[performance] "
    labels: [performance, monitoring]
    max: 1
---

# Workflow Performance Monitor

Monitor workflow execution performance and resource usage.

Use the agentic-workflows tool to:
1. Download logs for the last 50 runs
2. Analyze execution times and token usage
3. Identify performance regressions
4. Track cost trends

Create an issue if performance degrades by >20% or costs spike.
```

## Best Practices

1. **Combine with GitHub tools**: Use `agentic-workflows` together with the `github` tool to cross-reference workflow data with issues, PRs, and commits.

2. **Use safe-outputs**: Always use `safe-outputs` for creating issues or comments instead of requiring write permissions.

3. **Set appropriate schedules**: Don't run analysis workflows too frequently to avoid unnecessary API usage.

4. **Focus on actionable insights**: Have the AI agent provide specific, actionable recommendations rather than just reporting data.

5. **Filter strategically**: Use date filters and workflow name filters to focus analysis on relevant runs.

## Permissions

The agentic-workflows tool requires minimal permissions:

```yaml
permissions:
  contents: read  # To read workflow files
  actions: read   # To access workflow run data
```

Note: The tool itself only needs read access. If you want the agent to create issues or comments based on the analysis, use `safe-outputs` configuration which handles write permissions separately.

## Troubleshooting

### Tool not available
If the agentic-workflows tools are not available to the agent, check that:
1. The tool is properly configured in the frontmatter: `agentic-workflows:`
2. The installation step completed successfully (check workflow logs)
3. The gh-aw extension installed correctly

### Authentication errors
The tool uses `${{ secrets.GITHUB_TOKEN }}` by default. Ensure the token has appropriate permissions for the repository and workflow run access.

## Examples

See `.github/workflows/example-workflow-analyzer.md` for a complete working example of a workflow that uses the agentic-workflows tool to analyze workflow health and create improvement recommendations.
