# Metrics Collector Workflow

The Metrics Collector is an infrastructure workflow that collects daily performance metrics for the entire agentic workflow ecosystem and stores them in a structured format for analysis by meta-orchestrators.

## Overview

- **Location**: `.github/workflows/metrics-collector.md`
- **Schedule**: Daily (fuzzy schedule to distribute load)
- **Engine**: Copilot
- **Purpose**: Centralized metrics collection for historical trend analysis

## What It Collects

The Metrics Collector gathers comprehensive performance data across all workflows:

### Per-Workflow Metrics

For each workflow in the repository, the collector tracks:

1. **Safe Outputs**
   - Issues created
   - Pull requests created
   - Comments added
   - Discussions created

2. **Workflow Run Statistics**
   - Total runs in last 24 hours
   - Successful runs
   - Failed runs
   - Success rate (calculated)

3. **Engagement Metrics**
   - Reactions on issues
   - Comments on pull requests
   - Replies on discussions

4. **Quality Indicators**
   - PR merge rate (merged PRs / total PRs)
   - Average issue close time (in hours)
   - Average PR merge time (in hours)

### Ecosystem-Level Metrics

Aggregated data across the entire workflow ecosystem:

- Total number of workflows
- Number of active workflows (ran in last 24 hours)
- Total safe outputs created
- Overall success rate

## Storage Location

Metrics are stored in repo-memory under the meta-orchestrators branch:

```
/tmp/gh-aw/repo-memory-default/memory/meta-orchestrators/metrics/
├── daily/
│   ├── 2024-12-24.json
│   ├── 2024-12-25.json
│   └── ... (last 30 days)
└── latest.json (most recent snapshot)
```

### Storage Format

Metrics are stored in JSON format following this schema:

```json
{
  "timestamp": "2024-12-24T00:00:00Z",
  "period": "daily",
  "collection_duration_seconds": 45,
  "workflows": {
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
        "success_rate": 0.857
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
  },
  "ecosystem": {
    "total_workflows": 120,
    "active_workflows": 85,
    "total_safe_outputs": 45,
    "overall_success_rate": 0.892
  }
}
```

## Data Retention

- **Daily metrics**: Kept for 30 days
- **Latest snapshot**: Always available at `metrics/latest.json`
- **Cleanup**: Automated cleanup runs during each collection

## Consuming Metrics

Meta-orchestrators and other workflows can access metrics data:

### Latest Metrics

```bash
# Read most recent metrics
cat /tmp/gh-aw/repo-memory-default/memory/meta-orchestrators/metrics/latest.json
```

### Historical Metrics

```bash
# Read specific day
cat /tmp/gh-aw/repo-memory-default/memory/meta-orchestrators/metrics/daily/2024-12-24.json

# List all available days
ls /tmp/gh-aw/repo-memory-default/memory/meta-orchestrators/metrics/daily/
```

### In Workflow Files

Meta-orchestrators automatically load metrics using the repo-memory tool:

```yaml
tools:
  repo-memory:
    branch-name: memory/meta-orchestrators
    file-glob: "metrics/**/*"
```

## Integration with Meta-Orchestrators

The following workflows consume metrics data:

### Agent Performance Analyzer

Uses metrics to:
- Track historical performance trends
- Compare current vs. historical success rates
- Calculate week-over-week and month-over-month changes
- Avoid redundant API queries (metrics already collected)

### Campaign Manager

Uses metrics to:
- Assess campaign health via workflow success rates
- Calculate velocity trends from safe output volume
- Detect performance degradation early
- Predict completion dates based on velocity

### Workflow Health Manager

Uses metrics to:
- Identify failing workflows without repeated API queries
- Track quality trends using historical data
- Calculate 7-day and 30-day success rate trends
- Compute mean time between failures (MTBF)

## Manual Testing

To manually test the metrics collector:

1. **Trigger a manual run**:
   ```bash
   gh workflow run metrics-collector.md
   ```

2. **Check the run status**:
   ```bash
   gh run list --workflow=metrics-collector.lock.yml
   ```

3. **Verify metrics were stored**:
   ```bash
   # Check if latest.json exists and is valid JSON
   cat /tmp/gh-aw/repo-memory-default/memory/meta-orchestrators/metrics/latest.json | jq .
   
   # Check daily metrics
   ls -lh /tmp/gh-aw/repo-memory-default/memory/meta-orchestrators/metrics/daily/
   ```

4. **Validate data structure**:
   ```bash
   # Verify required fields exist
   jq '.timestamp, .workflows | length, .ecosystem.total_workflows' \
     /tmp/gh-aw/repo-memory-default/memory/meta-orchestrators/metrics/latest.json
   ```

## Benefits

The shared metrics infrastructure enables:

✅ **Historical Trend Analysis**: Compare performance week-over-week and month-over-month
✅ **Performance Benchmarking**: Compare individual workflows to ecosystem averages
✅ **Anomaly Detection**: Identify sudden drops in success rate or output volume
✅ **Evidence-Based Prioritization**: Use data to prioritize improvements
✅ **Reduced API Load**: Meta-orchestrators query pre-collected metrics instead of GitHub API
✅ **Coordinated Insights**: All meta-orchestrators work from the same data foundation

## Configuration

The workflow requires:

- `actions: read` permission for GitHub Actions API access
- GitHub MCP server with `actions` toolset enabled
- Repo-memory tool configured for meta-orchestrators branch

No additional secrets or configuration needed.

## Troubleshooting

### Metrics Not Collecting

1. Check workflow run logs for errors
2. Verify `actions: read` permission is granted
3. Ensure GitHub MCP server is accessible
4. Check repo-memory branch exists

### Invalid JSON Format

1. Workflow validates JSON with `jq` before storing
2. Check workflow logs for validation errors
3. Verify all required fields are present

### Missing Historical Data

1. Workflow only keeps last 30 days
2. Check if cleanup removed older files
3. Verify daily collection is running successfully

## Future Enhancements

Potential improvements to consider:

- Weekly and monthly aggregates for longer-term trends
- Alert thresholds for anomaly detection
- Dashboard visualization of trends
- Export to external analytics platforms
- Custom metric definitions per workflow type
