---
title: MemoryOps
description: Techniques for using cache-memory and repo-memory to build stateful workflows that track progress, share data, and compute trends
sidebar:
  badge: { text: 'Patterns', variant: 'note' }
---

MemoryOps enables workflows to persist state across runs using `cache-memory` and `repo-memory`. Build workflows that remember their progress, resume after interruptions, share data between workflows, and avoid API throttling.

Use MemoryOps for incremental processing, trend analysis, multi-step tasks, and workflow coordination.

## Memory Types

### Cache Memory

Fast, ephemeral storage using GitHub Actions cache (7 days retention):

```yaml
tools:
  cache-memory:
    key: my-workflow-state
```

**Use for**: Temporary state, session data, short-term caching  
**Location**: `/tmp/gh-aw/cache-memory/`

### Repository Memory

Persistent, version-controlled storage in a dedicated Git branch:

```yaml
tools:
  repo-memory:
    branch-name: memory/my-workflow
    file-glob: ["*.json", "*.jsonl"]
```

**Use for**: Historical data, trend tracking, permanent state  
**Location**: `/tmp/gh-aw/repo-memory/default/`

## Pattern 1: Exhaustive Processing

Track progress through large datasets with todo/done lists to ensure complete coverage across multiple runs.

**Concept**: Maintain a state file with items to process (`todo`) and completed items (`done`). After processing each item, immediately update the state so the workflow can resume if interrupted.

**Example structure**:
```json
{
  "todo": [123, 456, 789],
  "done": [101, 102],
  "errors": [],
  "last_run": 1705334400
}
```

**Workflow steps**:
1. Load existing state or initialize new state
2. Build todo list (exclude already-done items)
3. Process items one by one, marking each as done immediately
4. Generate progress report

**Real examples**: `.github/workflows/repository-quality-improver.md`, `.github/workflows/copilot-agent-analysis.md`

## Pattern 2: State Persistence

Save workflow checkpoints to resume long-running tasks that may timeout.

**Concept**: Store a checkpoint with the last processed position. Each run loads the checkpoint, processes a batch, then saves the new position.

**Example checkpoint**:
```json
{
  "last_processed_id": 1250,
  "batch_number": 13,
  "total_migrated": 1250,
  "status": "in_progress"
}
```

**Workflow steps**:
1. Load checkpoint (or start at 0)
2. Process next batch from checkpoint position
3. Save new checkpoint with updated position
4. Detect completion when no items remain

**Real examples**: `.github/workflows/daily-news.md`, `.github/workflows/cli-consistency-checker.md`

## Pattern 3: Shared Information

Share data between workflows using repo-memory branches.

**Concept**: One workflow (producer) collects data and stores it in repo-memory. Other workflows (consumers) read and analyze the shared data.

**Producer workflow**:
```yaml
tools:
  repo-memory:
    branch-name: memory/shared-data
```

Store data in JSON Lines format:
```bash
# Append new data point
echo '{"timestamp": 1705334400, "value": 42}' >> history.jsonl
```

**Consumer workflow**:
```yaml
tools:
  repo-memory:
    branch-name: memory/shared-data  # Same branch
```

Read shared data:
```bash
# Load historical data
cat /tmp/gh-aw/repo-memory/default/history.jsonl
```

**Real examples**: `.github/workflows/metrics-collector.md` (producer), trend analysis workflows (consumers)

## Pattern 4: Data Caching

Cache API responses to avoid rate limits and reduce workflow time.

**Concept**: Before making expensive API calls, check if cached data exists and is fresh. If cache is valid (based on TTL), use cached data. Otherwise, fetch fresh data and update cache.

**Example with 24-hour TTL**:
```bash
CACHE_FILE="/tmp/gh-aw/cache-memory/data.json"
CACHE_TIMESTAMP="/tmp/gh-aw/cache-memory/.timestamp"
CACHE_TTL=86400  # 24 hours

if [ -f "$CACHE_TIMESTAMP" ]; then
  CACHE_AGE=$(($(date +%s) - $(cat "$CACHE_TIMESTAMP")))
  if [ $CACHE_AGE -lt $CACHE_TTL ]; then
    echo "Using cached data"
    DATA=$(cat "$CACHE_FILE")
  fi
fi

# If no valid cache, fetch fresh data
if [ -z "$DATA" ]; then
  # Fetch from API
  date +%s > "$CACHE_TIMESTAMP"
fi
```

**Cache TTL guidelines**:
- Repository metadata: 24 hours
- Contributor lists: 12 hours
- Issues/PRs: 1 hour
- Workflow runs: 30 minutes

**Real examples**: `.github/workflows/daily-news.md`

## Pattern 5: Trend Computation

Store time-series data and compute trends, moving averages, and statistics.

**Concept**: Append new data points to a history file (JSON Lines format). Load historical data to compute trends, moving averages, and generate visualizations.

**Data collection**:
```bash
# Append today's metrics
echo '{"date": "2024-01-15", "value": 42}' >> history.jsonl
```

**Trend analysis**:
```python
import pandas as pd

# Load data
df = pd.read_json('history.jsonl', lines=True)

# Compute 7-day moving average
df['ma_7d'] = df['value'].rolling(window=7).mean()

# Compute 30-day moving average
df['ma_30d'] = df['value'].rolling(window=30).mean()
```

**Real examples**: `.github/workflows/daily-code-metrics.md`, `.github/workflows/shared/charts-with-trending.md`

## Pattern 6: Multiple Memory Stores

Use multiple memory instances for different purposes and retention policies.

**Concept**: Separate hot data (cache-memory) from historical data (repo-memory). Use different repo-memory branches for metrics vs. configuration vs. archives.

**Example configuration**:
```yaml
tools:
  cache-memory:
    key: session-data  # Fast, temporary
  
  repo-memory:
    - id: metrics
      branch-name: memory/metrics  # Time-series data
    
    - id: config
      branch-name: memory/config  # Schema and metadata
    
    - id: archive
      branch-name: memory/archive  # Compressed backups
```

**Access patterns**:
- **Session cache**: Intermediate calculations within a run
- **Metrics store**: Daily statistics and trends
- **Config store**: Data schemas and catalogs
- **Archive store**: Full data snapshots (compressed)

## Best Practices

### Use JSON Lines for Time-Series Data

Append-only format ideal for logs and metrics:

```bash
# Append without reading entire file
echo '{"date": "2024-01-15", "value": 42}' >> data.jsonl
```

### Include Metadata

Document your data structure:

```json
{
  "dataset": "performance-metrics",
  "schema": {
    "date": "YYYY-MM-DD",
    "value": "integer"
  },
  "retention": "90 days"
}
```

### Implement Data Rotation

Prevent unbounded growth:

```bash
# Keep only last 90 entries
tail -n 90 history.jsonl > history-trimmed.jsonl
mv history-trimmed.jsonl history.jsonl
```

### Validate State

Check integrity before processing:

```bash
if [ -f state.json ] && jq empty state.json 2>/dev/null; then
  echo "Valid state"
else
  echo "Corrupt state, reinitializing..."
  echo '{}' > state.json
fi
```

## Security Considerations

> [!CAUTION]
> **Sensitive Data**
>
> Memory stores are visible to anyone with repository access:
> - **Never store**: Credentials, API tokens, PII, secrets
> - **Store only**: Aggregate statistics, anonymized data
> - Consider encryption for sensitive but non-secret data

**Safe practices**:
```bash
# ✅ GOOD - Aggregate statistics
echo '{"open_issues": 42}' > metrics.json

# ❌ BAD - Individual user data
echo '{"user": "alice", "email": "alice@example.com"}' > users.json
```

## Troubleshooting

**Cache not persisting**: Verify cache key is consistent across runs

**Repo memory not updating**: Check `file-glob` patterns match your files and files are within `max-file-size` limit

**Out of memory errors**: Process data in chunks instead of loading entirely, implement data rotation

**Merge conflicts**: Use JSON Lines format (append-only), separate branches per workflow, or add run ID to filenames

## Related Documentation

- [MCP Servers](/gh-aw/guides/mcps/) - Memory MCP server configuration
- [Deterministic Patterns](/gh-aw/guides/deterministic-agentic-patterns/) - Data preprocessing
- [Safe Outputs](/gh-aw/guides/custom-safe-outputs/) - Storing workflow outputs
- [Frontmatter Reference](/gh-aw/reference/frontmatter/) - Configuration options
