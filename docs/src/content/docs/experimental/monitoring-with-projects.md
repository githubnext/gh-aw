---
title: Monitoring with Projects
description: Use GitHub Projects + safe-outputs to track and monitor workflow work items and progress.
---

:::caution[Experimental]
The monitoring with projects pattern is experimental and subject to change.
:::

Use this pattern when you want a durable “source of truth” for what your agentic workflows discovered, decided, and did.

## What this pattern is

- **Projects** are the dashboard: a GitHub Projects v2 board holds issues/PRs and custom fields.
- **Monitoring** is the behavior: workflows continuously add/update items, and periodically post status updates.

## Building blocks

### 1) Track items with `update-project`

Enable the safe output and point it at your project URL:

```yaml
safe-outputs:
  update-project:
    project: https://github.com/orgs/myorg/projects/123
    max: 10
    github-token: ${{ secrets.GH_AW_PROJECT_GITHUB_TOKEN }}
```

Adds issues/PRs to the board and updates custom fields, and can create views and custom fields when configured. See the [`update-project` reference](/gh-aw/reference/safe-outputs/#project-board-updates-update-project).

### 2) Post run summaries with `create-project-status-update`

Use project status updates to communicate progress and next steps:

```yaml
safe-outputs:
  create-project-status-update:
    project: https://github.com/orgs/myorg/projects/123
    max: 1
    github-token: ${{ secrets.GH_AW_PROJECT_GITHUB_TOKEN }}
```

This is useful for scheduled (daily/weekly) or orchestrator workflows. See the [`create-project-status-update` reference](/gh-aw/reference/safe-outputs/#project-status-updates-create-project-status-update).

### 3) Correlate work with a Tracker Id field

If you want to correlate multiple runs, add a custom field like **Tracker Id** (text) and populate it from your workflow prompt/output (for example, a run ID, issue number, or “initiative” key).

## Run failure issues

When a workflow run fails, the system automatically posts a failure notification on the triggering issue or pull request. To track failures as searchable GitHub issues, enable `create-issue` in `safe-outputs`:

```yaml wrap
safe-outputs:
  create-issue:
    title-prefix: "[failed] "
    labels: [automation, failed]
```

The issue body includes the workflow name, run URL, and failure status, making it easy to find and triage recurring failures.

### Grouping failures as sub-issues

When multiple workflow runs fail, the `group-reports` option links each failure report as a sub-issue under a shared parent issue titled "[aw] Failed runs". This is useful for scheduled or high-frequency workflows where failures can accumulate.

```yaml wrap
safe-outputs:
  create-issue:
    title-prefix: "[failed] "
    labels: [automation, failed]
  group-reports: true   # Group failure reports under a shared parent issue (default: false)
```

When enabled, a parent "[aw] Failed runs" issue is automatically created and managed, and each failure report is linked as a sub-issue under it (up to 64 per parent). See the [`group-reports` reference](/gh-aw/reference/safe-outputs/#group-reports-group-reports).

## No-op run reports

When an agent determines that no action is needed (for example, no issues were found), it outputs a no-op message. By default, this message is posted as a comment on the triggering issue or pull request, keeping a visible record of runs that intentionally did nothing.

To disable posting no-op messages as issue comments:

```yaml wrap
safe-outputs:
  create-issue:
  noop:
    report-as-issue: false  # Disable posting noop messages as issue comments
```

No-op messages still appear in the workflow step summary even when `report-as-issue` is `false`. To disable the no-op output entirely:

```yaml wrap
safe-outputs:
  create-issue:
  noop: false   # Disable noop output completely
```

See the [no-op logging reference](/gh-aw/reference/safe-outputs/#no-op-logging-noop).

## Operational monitoring

Use `gh aw status` to see which workflows are enabled and their latest run state.

For deeper investigation, the audit commands are the primary monitoring tool for agentic workflows:

- `gh aw audit <run-id>` — single-run report with tool usage, MCP failures, firewall activity, and cost metrics
- `gh aw audit <run-id-1> <run-id-2>` — compare two runs to detect behavioral regressions or new network accesses (pass additional IDs to compare base against multiple runs)
- `gh aw logs my-workflow --format markdown --count 10` — cross-run security and performance report for trend monitoring

> [!TIP]
> Use `gh aw logs --format markdown` inside a scheduled workflow agent to automate trend monitoring and surface cost or security regressions without manual intervention.

See [Audit Commands](/gh-aw/reference/audit/) for full flag documentation, and [CLI Reference](/gh-aw/setup/cli/) for all available commands.
