# Metrics Collection

This directory contains daily performance metrics for all agentic workflows in the repository.

## Structure

```
metrics/
├── daily/
│   ├── 2026-01-16.json   # Daily metrics snapshot
│   ├── 2026-01-15.json
│   └── ...
└── latest.json           # Most recent metrics (quick access)
```

## Data Format

Each metrics file contains:

```json
{
  "timestamp": "2026-01-16T14:41:31.938498Z",
  "period": "daily",
  "collection_duration_seconds": 0,
  "workflows": {
    "workflow-name": {
      "safe_outputs": {
        "issues_created": 12,
        "prs_created": 2,
        "comments_added": 1,
        "discussions_created": 0
      },
      "workflow_runs": {
        "total": null,
        "successful": null,
        "failed": null,
        "success_rate": null,
        "avg_duration_seconds": null,
        "total_tokens": null,
        "total_cost_usd": null
      },
      "engagement": {
        "issue_reactions": 2,
        "pr_comments": 0,
        "discussion_replies": 0
      },
      "quality_indicators": {
        "pr_merge_rate": 0.0,
        "avg_issue_close_time_hours": null,
        "avg_pr_merge_time_hours": null
      }
    }
  },
  "ecosystem": {
    "total_workflows": 124,
    "active_workflows": 10,
    "total_safe_outputs": 36,
    "overall_success_rate": null,
    "total_tokens": null,
    "total_cost_usd": null
  }
}
```

## Data Sources

- **GitHub Issues API**: Issues created in last 24 hours
- **GitHub PRs API**: Pull requests created in last 24 hours
- **Workflow Inventory**: Total workflows in repository
- **Workflow Footers**: Parsed from issue/PR bodies to identify which workflow created them

## Data Limitations

**Currently Available:**
- Safe outputs (issues, PRs, comments created by workflows)
- Engagement metrics (reactions, comments on outputs)
- Workflow inventory and activity counts
- PR merge rates

**Currently Unavailable:**
- Workflow run statistics (success/failure rates)
- Execution duration and performance metrics
- Token usage and cost data
- These require direct access to GitHub Actions API or gh-aw CLI tools

## Retention Policy

- Daily metrics are retained for **30 days**
- Files older than 30 days are automatically deleted
- `latest.json` is always preserved

## Usage by Meta-Orchestrators

### Agent Performance Analyzer
Use this data to:
- Identify underperforming workflows
- Track workflow productivity trends
- Analyze safe output patterns
- Recommend optimizations

### Campaign Manager
Use this data to:
- Assess workflow health across campaigns
- Track campaign effectiveness metrics
- Identify resource allocation opportunities

### Workflow Health Manager
Use this data to:
- Monitor workflow activity levels
- Detect stale or inactive workflows
- Track engagement with workflow outputs
- Generate health reports

## File Constraints

- **Max file size**: 10KB per file
- **Max file count**: 100 files
- **Allowed patterns**: `metrics/**`

## Collection Schedule

Metrics are collected daily by the `metrics-collector` workflow.

## Last Updated

2026-01-16 by metrics-collector workflow
