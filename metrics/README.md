# Metrics Collection System

## Overview

This directory contains daily performance metrics for all agentic workflows in the gh-aw ecosystem. The metrics are collected by the `metrics-collector` workflow and stored in a structured JSON format.

## Directory Structure

```
metrics/
├── README.md                    # This file
├── latest.json                  # Most recent metrics snapshot
└── daily/
    ├── 2026-01-06.json         # Daily metrics (YYYY-MM-DD format)
    ├── 2026-01-07.json
    └── ...
```

## Retention Policy

- **Daily metrics**: Kept for 30 days
- **Latest snapshot**: Always preserved
- **Cleanup**: Automatic removal of files older than 30 days

## Metrics Schema

### Top-Level Structure

```json
{
  "timestamp": "ISO 8601 datetime",
  "period": "daily",
  "collection_duration_seconds": 0,
  "collection_status": "complete|limited|failed",
  "workflows": {},
  "ecosystem": {}
}
```

### Workflow Metrics

Each workflow in the `workflows` object contains:

```json
{
  "workflow-name": {
    "safe_outputs": {
      "issues_created": 5,
      "prs_created": 2,
      "comments_added": 10,
      "discussions_created": 1
    },
    "workflow_runs": {
      "total": 7,
      "successful": 6,
      "failed": 1,
      "success_rate": 0.857,
      "avg_duration_seconds": 180,
      "total_tokens": 45000,
      "total_cost_usd": 0.45
    },
    "engagement": {
      "issue_reactions": 12,
      "pr_comments": 8,
      "discussion_replies": 3
    },
    "quality_indicators": {
      "pr_merge_rate": 0.75,
      "avg_issue_close_time_hours": 48.5,
      "avg_pr_merge_time_hours": 72.3
    }
  }
}
```

### Ecosystem Metrics

Aggregated metrics across all workflows:

```json
{
  "ecosystem": {
    "total_workflows": 126,
    "active_workflows": 85,
    "total_safe_outputs": 45,
    "overall_success_rate": 0.892,
    "total_tokens": 1250000,
    "total_cost_usd": 12.50
  }
}
```

## Data Sources

### Primary: Agentic Workflows Tool

The metrics collector uses the `gh-aw` CLI tool to gather data:

- `gh aw status` - Workflow inventory
- `gh aw logs --start-date "-1d"` - Last 24 hours of workflow runs

### Secondary: GitHub API

Supplemental data via GitHub MCP server:

- Issue/PR reactions
- Comment counts
- Discussion replies

## Collection Requirements

For full metrics collection, the workflow needs:

1. **GH_TOKEN environment variable** - GitHub API authentication
2. **gh-aw CLI extension** - Workflow data access
3. **Repository permissions**:
   - `contents: read`
   - `issues: read`
   - `pull-requests: read`

## Usage by Meta-Orchestrators

### Agent Performance Analyzer

Uses metrics to:
- Calculate quality scores for each workflow
- Identify performance trends
- Generate improvement recommendations

### Campaign Manager

Uses metrics to:
- Assess campaign progress
- Identify struggling workflows
- Prioritize resource allocation

### Workflow Health Manager

Uses metrics to:
- Monitor workflow success rates
- Detect failures and anomalies
- Track ecosystem health

## Querying Metrics

### Get Latest Metrics

```bash
cat /tmp/gh-aw/repo-memory/default/metrics/latest.json
```

### Get Specific Day

```bash
cat /tmp/gh-aw/repo-memory/default/metrics/daily/2026-01-06.json
```

### Check Workflow Status

```bash
cat /tmp/gh-aw/repo-memory/default/metrics/latest.json | \
  python3 -c "import sys, json; d=json.load(sys.stdin); print(d['collection_status'])"
```

### Count Active Workflows

```bash
cat /tmp/gh-aw/repo-memory/default/metrics/latest.json | \
  python3 -c "import sys, json; d=json.load(sys.stdin); print(d['ecosystem']['active_workflows'])"
```

## Troubleshooting

### Limited Collection Status

If `collection_status` is "limited":
- Check `limitations` array for specific issues
- Review `recommendations` for resolution steps
- Verify workflow configuration has proper permissions

### Missing Workflow Data

If a workflow is missing from `workflows` object:
- Workflow may have had no runs in collection period
- Check `workflow_inventory` to verify workflow exists
- Review workflow schedule and trigger configuration

### Zero Success Rate

If `overall_success_rate` is 0:
- May indicate collection limitation (no run data)
- Check individual workflow success rates
- Investigate failed workflows in GitHub Actions UI

## File Size Constraints

- **Max file size**: 10 MB per file
- **Max files**: 100 files per commit
- **Patterns**: Only `metrics/**` files allowed

If metrics exceed limits:
- Aggregate data more aggressively
- Reduce historical detail
- Archive old data to separate storage

## Integration

These metrics are automatically:
- **Collected**: Daily by metrics-collector workflow
- **Stored**: In `memory/meta-orchestrators` git branch
- **Pushed**: After workflow completion
- **Merged**: With "ours" strategy (current version wins)

---

**Last Updated**: 2026-01-06  
**Maintained By**: metrics-collector workflow
