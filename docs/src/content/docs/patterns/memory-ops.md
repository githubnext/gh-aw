---
title: MemoryOps
description: Design patterns for using memory to build stateful workflows that track progress, share data, and compute trends
sidebar:
  badge: { text: 'Patterns', variant: 'note' }
---

MemoryOps is a set of design patterns using [Cache Memory](/gh-aw/reference/cache-memory/) and [Repo Memory](/gh-aw/reference/repo-memory/) to persist state across workflow runs. Use memory to build workflows that record their progress, resume after interruptions, share data between workflows, incremental processing, trend analysis, multi-step tasks, and workflow coordination.

```mermaid
flowchart LR
    r1([Run 1]) --> s1[memory\nstate 1]
    s1 --> r2([Run 2])
    r2 --> s2[memory\nstate 2]
    s2 --> r3([Run 3])
```

When writing prompting for using memory, you can usually use surprisingly high level descriptions of the information to be stored. You often don't even need to write a schema for the memory. The first run will write data with an appropriate schema, later runs will read it and see the implicit schema. The agent can adapt to changes in the schema over time as long as the data is still there. This makes memory a very flexible tool for stateful workflows without needing to define rigid schemas upfront.

## Memory Types

Two types of memory stores are available. Each has different use cases and access patterns.

[Cache Memory](/gh-aw/reference/cache-memory/) gives fast, ephemeral storage using GitHub Actions cache (7 days retention):

```yaml
tools:
  cache-memory:
    key: my-workflow-state
```

Use for temporary state, session data, short-term caching. The memory is available at `/tmp/gh-aw/cache-memory/`.

[Repo Memory](/gh-aw/reference/repo-memory/) gives persistent, version-controlled storage in a dedicated Git branch:

```yaml
tools:
  repo-memory:
    branch-name: memory/my-workflow
    file-glob: ["*.json", "*.jsonl"]
```

Use for historical data, trend tracking, permanent state. By default the memory is available at `/tmp/gh-aw/repo-memory/default/`.

## Pattern 1: Exhaustive Processing

Track progress through large datasets with todo/done lists to ensure complete coverage across multiple runs.

```markdown
Analyze all open issues in the repository. Track your progress in cache-memory
so you can resume if the workflow times out. Mark each issue as done after
processing it. Generate a final report with statistics.
```

The agent maintains a state file with items to process and completed items, updating it after each item so the workflow can resume if interrupted:

```json
{
  "todo": [123, 456, 789],
  "done": [101, 102],
  "errors": [],
  "last_run": 1705334400
}
```

Real examples: `.github/workflows/repository-quality-improver.md`, `.github/workflows/copilot-agent-analysis.md`

## Pattern 2: State Persistence

Save workflow checkpoints to resume long-running tasks that may timeout.

```markdown
Migrate 10,000 records from the old format to the new format. Process 500
records per run and save a checkpoint. Each run should resume from the last
checkpoint until all records are migrated.
```

The agent stores a checkpoint with the last processed position and resumes from it each run:

```json
{
  "last_processed_id": 1250,
  "batch_number": 13,
  "total_migrated": 1250,
  "status": "in_progress"
}
```

Real examples: `.github/workflows/daily-news.md`, `.github/workflows/cli-consistency-checker.md`

## Pattern 3: Shared Information

Share data between workflows using [repo-memory](/gh-aw/reference/repo-memory/) branches. A producer workflow stores data; consumers read it using the same branch name.

*Producer workflow:*
```markdown
Every 6 hours, collect repository metrics (issues, PRs, stars) and store them
in repo-memory so other workflows can analyze the data later.
```

*Consumer workflow:*
```markdown
Load the historical metrics from repo-memory and compute weekly trends.
Generate a trend report with visualizations.
```

Both workflows reference the same branch:

```yaml
tools:
  repo-memory:
    branch-name: memory/shared-data
```

Real examples: `.github/workflows/metrics-collector.md` (producer), trend analysis workflows (consumers)

## Pattern 4: Data Caching

Cache API responses to avoid rate limits and reduce workflow time. The agent checks for fresh cached data before making API calls, using suggested TTLs: repository metadata (24h), contributor lists (12h), issues/PRs (1h), workflow runs (30m).

```markdown
Fetch repository metadata and contributor lists. Cache the data for 24 hours
to avoid repeated API calls. If the cache is fresh, use it. Otherwise, fetch
new data and update the cache.
```

Real examples: `.github/workflows/daily-news.md`

## Pattern 5: Trend Computation

Store time-series data and compute trends, moving averages, and statistics. The agent appends new data points to a JSON Lines history file and computes trends using Python.

```markdown
Collect daily build times and test times. Store them in repo-memory as
time-series data. Compute 7-day and 30-day moving averages. Generate trend
charts showing whether performance is improving or declining over time.
```

Real examples: `.github/workflows/daily-code-metrics.md`, `.github/workflows/shared/charts-with-trending.md`

## Pattern 6: Multiple Memory Stores

Use multiple memory instances for different lifecycles — cache-memory for temporary session data, separate repo-memory branches for metrics, configuration, and archives.

```markdown
Use cache-memory for temporary API responses during this run. Store daily
metrics in one repo-memory branch for trend analysis. Keep data schemas in
another branch. Archive full snapshots in a third branch with compression.
```

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

## Security Considerations

Memory stores are visible to anyone with repository access. Never store credentials, API tokens, PII, or secrets — only aggregate statistics and anonymized data.

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

- [Cache Memory](/gh-aw/reference/cache-memory/) — Full cache-memory reference
- [Repository Memory](/gh-aw/reference/repo-memory/) — Full repo-memory reference
- [MCP Servers](/gh-aw/guides/mcps/) - Memory MCP server configuration
- [DeterministicOps](/gh-aw/patterns/deterministic-ops/) - Data preprocessing and extraction
- [Safe Outputs](/gh-aw/reference/custom-safe-outputs/) - Storing workflow outputs
- [Frontmatter Reference](/gh-aw/reference/frontmatter/) - Configuration options
