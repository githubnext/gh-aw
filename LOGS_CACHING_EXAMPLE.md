# Logs Command Caching Feature

## Overview

The logs command now automatically caches run summaries to avoid re-downloading and re-processing already analyzed workflow runs. Each run folder gets a `run_summary.json` file that acts as a marker and contains all extracted metadata.

## How It Works

### First Run (Initial Download)
```bash
gh aw logs weekly-research -c 5
```

1. Downloads artifacts from GitHub
2. Parses logs and extracts metrics
3. Analyzes access logs, missing tools, MCP failures
4. Creates `run_summary.json` in each run folder
5. Displays results

**Output:**
```
logs/
├── run-12345/
│   ├── aw_info.json
│   ├── agent-stdio.log
│   ├── safe_output.jsonl
│   └── run_summary.json    ← New cache file
└── run-12346/
    ├── aw_info.json
    ├── agent-stdio.log
    └── run_summary.json    ← New cache file
```

### Subsequent Runs (Cached)
```bash
gh aw logs weekly-research -c 5
```

1. Checks for `run_summary.json` in each run folder
2. Validates CLI version matches
3. Loads cached data (skips download and parsing)
4. Displays results immediately

**Performance:** ~10-100x faster than re-downloading and re-parsing

### Version Update (Cache Invalidation)

When you update the CLI:
```bash
gh extension upgrade aw
gh aw logs weekly-research -c 5
```

1. Detects version mismatch in `run_summary.json`
2. Re-processes affected runs with new CLI version
3. Updates `run_summary.json` with new version
4. Ensures you get latest bug fixes and improvements

## Run Summary File Structure

Each `run_summary.json` contains:

```json
{
  "cli_version": "1.0.0",
  "run_id": 12345,
  "processed_at": "2025-10-21T03:38:55Z",
  "run": {
    "databaseId": 12345,
    "workflowName": "Weekly Research",
    "status": "completed",
    "conclusion": "success",
    ...
  },
  "metrics": {
    "token_usage": 5000,
    "estimated_cost": 0.25,
    "turns": 3,
    "errors": [...]
  },
  "access_analysis": {
    "allowed_domains": ["github.com", "api.github.com"],
    "denied_domains": []
  },
  "missing_tools": [],
  "mcp_failures": [],
  "artifacts_list": [
    "aw_info.json",
    "agent-stdio.log",
    "safe_output.jsonl"
  ],
  "job_details": [...]
}
```

## Benefits

1. **Performance**: Much faster subsequent runs (no re-downloading or re-parsing)
2. **Bandwidth**: Saves network bandwidth by avoiding redundant downloads
3. **Reliability**: Consistent results across runs
4. **Versioning**: Automatic cache invalidation ensures bug fixes are applied
5. **Debugging**: Human-readable summary files for manual inspection

## Manual Cache Management

### View a cached summary
```bash
cat logs/run-12345/run_summary.json | jq .
```

### Force reprocessing
```bash
# Delete summary file to force reprocessing
rm logs/run-12345/run_summary.json
gh aw logs weekly-research -c 5
```

### Clear all cached summaries
```bash
# Remove all summary files
find logs -name "run_summary.json" -delete
gh aw logs weekly-research -c 5
```

## Verbose Mode

See caching in action:
```bash
gh aw logs weekly-research -c 5 -v
```

**Output:**
```
ℹ Processing run 12345 (completed)...
ℹ Loaded cached run summary for run 12345 (processed at 2025-10-21T02:30:00Z)
✓ Using cached artifacts for run 12345 at logs/run-12345
```

Or when cache is invalid:
```
ℹ Processing run 12345 (completed)...
ℹ Run summary version mismatch (cached: 0.9.0, current: 1.0.0), will reprocess
ℹ Downloading artifacts for run 12345...
✓ Saved run summary to logs/run-12345/run_summary.json
```
