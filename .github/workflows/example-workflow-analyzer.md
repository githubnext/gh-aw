---
on:
  schedule:
    - cron: "0 9 * * 1"  # Monday 9AM
  workflow_dispatch:
permissions:
  contents: read
  actions: read
engine: claude
tools:
  agentic-workflow:
  github:
    allowed: [get_workflow_run, list_workflow_runs]
safe-outputs:
  create-issue:
    title-prefix: "[workflow-analysis] "
    labels: [automation, ci-improvement]
timeout_minutes: 10
---

# Weekly Workflow Analysis

Analyze GitHub Actions workflow runs from the past week and identify improvement opportunities.

## Instructions

Use the agentic-workflow tool to:

1. **Check workflow status**: Use the `status` tool to see all workflows in the repository
2. **Download logs**: Use the `logs` tool with parameters like:
   - `workflow_name`: Specific workflow to analyze
   - `count`: Number of runs to analyze (e.g., 20)
   - `start_date`: Filter runs from last week (e.g., "-1w")
   - `engine`: Filter by AI engine if needed
3. **Audit failures**: Use the `audit` tool with `run_id` to investigate specific failed runs

## Analysis Tasks

Analyze the collected data and provide:

- **Failure Patterns**: Common errors across workflows
- **Performance Issues**: Slow steps or bottlenecks
- **Resource Usage**: Token usage and costs for AI-powered workflows
- **Reliability Metrics**: Success rates and error frequencies
- **Optimization Opportunities**: Suggestions for improving workflow efficiency

Create an issue with your findings and actionable recommendations for improving CI/CD reliability and performance.
