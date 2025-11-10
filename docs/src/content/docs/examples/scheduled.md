---
title: Scheduled Workflows
description: Workflows that run automatically on a schedule using cron expressions - daily reports, weekly research, and continuous improvement patterns
sidebar:
  order: 1
---

Scheduled workflows run automatically at specified times using cron expressions. They're perfect for recurring tasks like daily status updates, weekly research reports, continuous code improvements, and automated maintenance.

## When to Use Scheduled Workflows

- **Regular reporting**: Daily team status, weekly summaries
- **Continuous improvement**: Incremental code quality improvements (DailyOps)
- **Research & monitoring**: Weekly industry research, dependency updates
- **Maintenance tasks**: Cleaning up stale issues, archiving old discussions

## Patterns in This Section

- **[DailyOps](/gh-aw/examples/scheduled/dailyops/)** - Make incremental improvements through small daily changes
- **[Research & Planning](/gh-aw/examples/scheduled/research-planning/)** - Automated research, status reports, and planning

## Example Schedule Triggers

```yaml
on:
  schedule:
    - cron: "0 9 * * 1"      # Every Monday at 9 AM
    - cron: "0 0 * * *"      # Daily at midnight
    - cron: "0 */6 * * *"    # Every 6 hours
```

## Quick Start

Add a scheduled workflow to your repository:

```bash
gh aw add githubnext/agentics/weekly-research
gh aw add githubnext/agentics/daily-team-status
```
