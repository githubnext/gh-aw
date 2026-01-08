# Metrics Collection System

This directory contains daily performance metrics for the gh-aw agentic workflow ecosystem.

## Directory Structure

```
metrics/
├── daily/           # Daily metrics snapshots (YYYY-MM-DD.json)
│   ├── 2026-01-06.json
│   ├── 2026-01-07.json
│   └── 2026-01-08.json
├── latest.json      # Most recent metrics (quick access)
└── README.md        # This file
```

## Retention Policy

- Daily metrics are retained for 30 days
- Files older than 30 days are automatically deleted during collection
- `latest.json` is always preserved

## Metrics Schema

Each metrics file contains:

```json
{
  "timestamp": "ISO 8601 timestamp",
  "period": "daily",
  "collection_duration_seconds": 0,
  "collection_method": "filesystem_analysis or api_access",
  "limitations": ["array of known limitations"],
  "filesystem_analysis": {
    "total_workflow_files": 0,
    "shared_workflow_files": 0,
    "compiled_lock_files": 0,
    "campaign_workflows": 0,
    "compilation_rate": 0.0
  },
  "workflows": {},
  "ecosystem": {
    "total_workflows": 0,
    "active_workflows": null,
    "total_safe_outputs": null,
    "overall_success_rate": null,
    "total_tokens": null,
    "total_cost_usd": null
  }
}
```

## Collection Status

**Current Method**: Filesystem analysis only

**Available Metrics**:
- ✓ Total workflow count
- ✓ Shared workflow count  
- ✓ Compiled workflow count
- ✓ Campaign workflow count
- ✓ Compilation rate

**Unavailable Metrics** (requires GitHub API access):
- ✗ Workflow run statistics
- ✗ Safe output metrics
- ✗ Token usage and costs
- ✗ Engagement metrics
- ✗ Quality indicators

## Usage

**Meta-orchestrators** can read these metrics for:
- Historical trend analysis (Agent Performance Analyzer)
- Campaign health assessment (Campaign Manager)
- Workflow health monitoring (Workflow Health Manager)
- Data-driven optimization decisions

**Access latest metrics**:
```bash
cat /tmp/gh-aw/repo-memory/default/metrics/latest.json
```

**Access historical metrics**:
```bash
cat /tmp/gh-aw/repo-memory/default/metrics/daily/2026-01-08.json
```

## Future Improvements

To enable comprehensive metrics collection:
1. Install gh-aw extension
2. Set GITHUB_TOKEN in workflow environment
3. Grant workflow permissions for run metadata
4. Configure GitHub MCP server with credentials
