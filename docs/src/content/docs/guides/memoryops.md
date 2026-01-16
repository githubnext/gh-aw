---
title: MemoryOps
description: Techniques for using cache-memory and repo-memory to build stateful, persistent workflows that track progress, share data, and compute trends
sidebar:
  badge: { text: 'Patterns', variant: 'note' }
---

MemoryOps enables workflows to persist state, track progress, share information, and compute trends across runs using `cache-memory` and `repo-memory`. Build workflows that remember their state, resume from interruptions, coordinate across multiple runs, and avoid API throttling through intelligent caching.

Use MemoryOps for incremental processing, trend analysis, data collection, coordination between workflows, and any task that benefits from maintaining state across workflow runs.

## Memory Types

GitHub Agentic Workflows provides two types of persistent storage:

### Cache Memory

**cache-memory** provides fast, ephemeral storage that persists across workflow runs using GitHub Actions cache:

```yaml
tools:
  cache-memory:
    key: my-workflow-state-${{ github.workflow }}
```

**Features:**
- Survives across workflow runs
- Faster access than repo-memory
- Stored in GitHub Actions cache (7 days retention)
- Ideal for temporary state, session data, and short-term caching
- Located at `/tmp/gh-aw/cache-memory/`

### Repository Memory

**repo-memory** provides persistent, version-controlled storage in a dedicated branch:

```yaml
tools:
  repo-memory:
    branch-name: memory/my-workflow
    description: "Historical workflow data"
    file-glob: ["*.json", "*.jsonl", "*.csv"]
    max-file-size: 102400  # 100KB
```

**Features:**
- Permanent storage in Git repository
- Version-controlled history
- Survives workflow run cleanup
- Ideal for historical data, trend tracking, and long-term state
- Located at `/tmp/gh-aw/repo-memory/default/`
- Access previous versions via Git history

## Pattern 1: Exhaustive Processing with Todo/Done Tracking

Track progress through large datasets by maintaining todo and done lists, ensuring complete coverage even across multiple runs or failures.

### When to Use This Pattern

- Processing large collections (all issues, all PRs, all files)
- Tasks that may timeout before completion
- Workflows that need to resume after failure
- Ensuring no items are missed or duplicated

### Implementation

```yaml wrap title=".github/workflows/issue-analyzer.md"
---
on:
  schedule: daily
engine: copilot
tools:
  cache-memory:
    key: issue-analyzer-state-${{ github.workflow }}
  github:
    toolsets: [default]
safe-outputs:
  create-discussion:
timeout-minutes: 20
---

# Issue Analyzer with Progress Tracking

Analyze all open issues in the repository with exhaustive progress tracking.

## Phase 1: Initialize State

Load or create the tracking state from cache-memory:

```bash
# Create state directory
mkdir -p /tmp/gh-aw/cache-memory/state

# Check if state file exists
if [ -f /tmp/gh-aw/cache-memory/state/progress.json ]; then
  echo "Loading existing state..."
  cat /tmp/gh-aw/cache-memory/state/progress.json
else
  echo "Initializing new state..."
  cat > /tmp/gh-aw/cache-memory/state/progress.json << 'EOF'
{
  "todo": [],
  "done": [],
  "errors": [],
  "last_run": null,
  "total_processed": 0
}
EOF
fi
```

## Phase 2: Build Todo List

Fetch all issues and compare against already-processed items:

```bash
# Fetch all open issues
gh issue list --repo ${{ github.repository }} \
  --state open --limit 1000 --json number,title > /tmp/issues.json

# Load current state
DONE_ISSUES=$(jq -r '.done[]' /tmp/gh-aw/cache-memory/state/progress.json)

# Build todo list (exclude already processed)
jq --argjson done "$DONE_ISSUES" '
  [.[] | select(.number as $num | $done | index($num) | not)]
' /tmp/issues.json > /tmp/todo.json

# Update state with new todo list
jq --slurpfile todo /tmp/todo.json '
  .todo = ($todo[0] | map(.number))
' /tmp/gh-aw/cache-memory/state/progress.json > /tmp/state-updated.json

mv /tmp/state-updated.json /tmp/gh-aw/cache-memory/state/progress.json
```

## Phase 3: Process with Checkpoints

Process items with periodic state saves to survive interruptions:

For each issue number in the todo list:

1. **Analyze the issue** using GitHub MCP tools
2. **Extract insights** (sentiment, priority, category)
3. **Mark as done** immediately after processing:

```bash
# After successfully analyzing issue #123
ISSUE_NUM=123

jq --arg num "$ISSUE_NUM" '
  .done += [$num | tonumber] |
  .todo -= [$num | tonumber] |
  .total_processed += 1 |
  .last_run = now
' /tmp/gh-aw/cache-memory/state/progress.json > /tmp/state-checkpoint.json

mv /tmp/state-checkpoint.json /tmp/gh-aw/cache-memory/state/progress.json
```

4. **On error**, add to errors list:

```bash
# If issue analysis fails
ISSUE_NUM=456
ERROR_MSG="Rate limit exceeded"

jq --arg num "$ISSUE_NUM" --arg err "$ERROR_MSG" '
  .errors += [{"issue": ($num | tonumber), "error": $err, "timestamp": now}] |
  .todo -= [$num | tonumber]
' /tmp/gh-aw/cache-memory/state/progress.json > /tmp/state-error.json

mv /tmp/state-error.json /tmp/gh-aw/cache-memory/state/progress.json
```

## Phase 4: Generate Report

Create a summary report with progress statistics:

- Total issues analyzed: `jq '.total_processed' state/progress.json`
- Remaining in queue: `jq '.todo | length' state/progress.json`
- Failed analyses: `jq '.errors | length' state/progress.json`
- Completion percentage
- Estimated time to complete (based on current rate)

Post the report to a discussion with the create-discussion tool.

## Phase 5: Cleanup (Optional)

Once all items are processed:

```bash
# Check if todo list is empty
TODO_COUNT=$(jq '.todo | length' /tmp/gh-aw/cache-memory/state/progress.json)

if [ "$TODO_COUNT" -eq 0 ]; then
  echo "All issues processed! Resetting state..."
  cat > /tmp/gh-aw/cache-memory/state/progress.json << 'EOF'
{
  "todo": [],
  "done": [],
  "errors": [],
  "last_run": $(date +%s),
  "total_processed": 0
}
EOF
fi
```
```

> [!TIP]
> **Process in Batches**
>
> If you have thousands of items, process them in batches:
> - Take first N items from todo list (e.g., 100)
> - Process the batch with checkpoint after each item
> - Next run continues with remaining items
> - Add `batch_size` to state for tracking

### Real-World Examples

**`.github/workflows/repository-quality-improver.md`** - Tracks focus area history across runs to ensure diverse coverage of repository quality aspects

**`.github/workflows/copilot-agent-analysis.md`** - Processes all Copilot PRs exhaustively, tracking which PRs have been analyzed

## Pattern 2: Save State and Resume

Persist workflow state to enable resumption after timeouts, failures, or manual interruption, creating sequences of workflow runs that build on previous work.

### When to Use This Pattern

- Long-running tasks that exceed timeout limits
- Multi-stage processing pipelines
- Tasks that depend on external system availability
- Workflows triggered by manual approval gates

### Implementation

```yaml wrap title=".github/workflows/migration-task.md"
---
on:
  schedule: daily
  workflow_dispatch:
engine: copilot
tools:
  cache-memory:
    key: migration-state-${{ github.workflow }}
safe-outputs:
  create-pull-request:
timeout-minutes: 15
---

# Database Migration with Resumable State

Migrate database records in batches, resuming from last successful checkpoint.

## Phase 1: Load Checkpoint

Load the previous state or initialize if first run:

```bash
mkdir -p /tmp/gh-aw/cache-memory/migration

# Load checkpoint
if [ -f /tmp/gh-aw/cache-memory/migration/checkpoint.json ]; then
  echo "Resuming from checkpoint..."
  CHECKPOINT=$(cat /tmp/gh-aw/cache-memory/migration/checkpoint.json)
  
  LAST_ID=$(echo "$CHECKPOINT" | jq -r '.last_processed_id')
  BATCH_NUM=$(echo "$CHECKPOINT" | jq -r '.batch_number')
  TOTAL_MIGRATED=$(echo "$CHECKPOINT" | jq -r '.total_migrated')
  
  echo "Last processed ID: $LAST_ID"
  echo "Current batch: $BATCH_NUM"
  echo "Total migrated: $TOTAL_MIGRATED"
else
  echo "Starting fresh migration..."
  LAST_ID=0
  BATCH_NUM=0
  TOTAL_MIGRATED=0
fi
```

## Phase 2: Process Current Batch

Process records starting from the checkpoint:

```bash
BATCH_SIZE=100

# Query records after LAST_ID
# Process batch of records
# Transform and migrate each record

# Example: Process files from a directory
FILES_TO_MIGRATE=$(find /path/to/data -type f | sort | \
  awk -v last="$LAST_ID" 'NR > last' | head -n "$BATCH_SIZE")

CURRENT_ID=$LAST_ID
for FILE in $FILES_TO_MIGRATE; do
  # Process file
  echo "Processing: $FILE"
  
  # Your migration logic here
  # ...
  
  CURRENT_ID=$((CURRENT_ID + 1))
  TOTAL_MIGRATED=$((TOTAL_MIGRATED + 1))
done

BATCH_NUM=$((BATCH_NUM + 1))
```

## Phase 3: Save Checkpoint

Update checkpoint with progress:

```bash
cat > /tmp/gh-aw/cache-memory/migration/checkpoint.json << EOF
{
  "last_processed_id": $CURRENT_ID,
  "batch_number": $BATCH_NUM,
  "total_migrated": $TOTAL_MIGRATED,
  "timestamp": $(date +%s),
  "status": "in_progress"
}
EOF
```

## Phase 4: Detect Completion

Check if migration is complete:

```bash
# Count remaining records
REMAINING=$(find /path/to/data -type f | sort | \
  awk -v last="$CURRENT_ID" 'NR > last' | wc -l)

if [ "$REMAINING" -eq 0 ]; then
  echo "Migration complete!"
  
  # Update checkpoint to mark completion
  jq '.status = "completed" | .completion_time = now' \
    /tmp/gh-aw/cache-memory/migration/checkpoint.json > /tmp/final.json
  
  mv /tmp/final.json /tmp/gh-aw/cache-memory/migration/checkpoint.json
  
  # Create completion report
  echo "Total records migrated: $TOTAL_MIGRATED"
  echo "Total batches processed: $BATCH_NUM"
else
  echo "Migration in progress. Remaining: $REMAINING"
  echo "Next run will process from ID: $CURRENT_ID"
fi
```

Use the create-pull-request tool to commit migrated files if this is a code migration.
```

> [!TIP]
> **Multiple Checkpoints**
>
> Create multiple checkpoint files for complex state:
> - `checkpoint.json` - Main progress tracker
> - `errors.jsonl` - Failed items (append-only)
> - `statistics.json` - Running statistics
> - `metadata.json` - Configuration and context

### Real-World Examples

**`.github/workflows/daily-news.md`** - Caches fetched GitHub data with timestamp, reuses if less than 24 hours old to avoid re-fetching

**`.github/workflows/cli-consistency-checker.md`** - Maintains state across runs to track consistency improvements over time

## Pattern 3: Shared Information with Repo Memory

Share data between workflows using repo-memory for coordination, handoffs, and data exchange.

### When to Use This Pattern

- Workflows that depend on data from other workflows
- Building incremental datasets from multiple sources
- Coordinating related workflows
- Creating shared knowledge bases

### Implementation

```yaml wrap title=".github/workflows/data-producer.md"
---
on:
  schedule:
    - cron: "0 */6 * * *"  # Every 6 hours
engine: copilot
tools:
  repo-memory:
    branch-name: memory/shared-data
    file-glob: ["metrics/*.jsonl", "reports/*.json"]
  github:
    toolsets: [default]
timeout-minutes: 10
---

# Data Producer Workflow

Collect metrics and store in shared repo-memory for consumer workflows.

## Collect Current Metrics

Use GitHub MCP tools to gather current repository metrics:

```bash
# Create directory structure
mkdir -p /tmp/gh-aw/repo-memory/default/metrics

# Fetch current issues
gh api repos/${{ github.repository }}/issues \
  --paginate \
  --jq '[.[] | {number, title, state, created_at, updated_at, labels: [.labels[].name]}]' \
  > /tmp/issues.json

# Append to historical data (JSON Lines format)
TIMESTAMP=$(date +%s)
jq --arg ts "$TIMESTAMP" '. as $data | {timestamp: $ts, date: (now | strftime("%Y-%m-%d")), issues: $data}' \
  /tmp/issues.json \
  >> /tmp/gh-aw/repo-memory/default/metrics/issues.jsonl

# Also save latest snapshot for quick access
jq --arg ts "$TIMESTAMP" '{timestamp: $ts, date: (now | strftime("%Y-%m-%d")), issues: .}' \
  /tmp/issues.json \
  > /tmp/gh-aw/repo-memory/default/metrics/issues-latest.json
```

## Store Metadata

Create metadata file for consumers:

```bash
cat > /tmp/gh-aw/repo-memory/default/metrics/metadata.json << EOF
{
  "producer": "data-producer",
  "last_updated": $(date +%s),
  "last_run": "${{ github.run_id }}",
  "format": "jsonl",
  "schema": {
    "timestamp": "unix_timestamp",
    "date": "YYYY-MM-DD",
    "issues": "array of issue objects"
  },
  "retention": "90 days",
  "description": "Historical issue data collected every 6 hours"
}
EOF
```

Repo-memory automatically commits and pushes these files to the `memory/shared-data` branch.
```

Now create a consumer workflow:

```yaml wrap title=".github/workflows/trend-analyzer.md"
---
on:
  schedule: daily
engine: copilot
tools:
  repo-memory:
    branch-name: memory/shared-data
    file-glob: ["metrics/*.jsonl", "reports/*.json"]
safe-outputs:
  create-discussion:
imports:
  - shared/python-dataviz.md
timeout-minutes: 15
---

# Trend Analyzer Workflow

Analyze trends from shared data produced by data-producer workflow.

## Load Historical Data

Load data from repo-memory (already available at mount point):

```bash
# Check if data exists
if [ ! -f /tmp/gh-aw/repo-memory/default/metrics/issues.jsonl ]; then
  echo "Error: No historical data found. Has data-producer run?"
  exit 1
fi

# Load metadata
cat /tmp/gh-aw/repo-memory/default/metrics/metadata.json

# Count data points
DATA_POINTS=$(wc -l < /tmp/gh-aw/repo-memory/default/metrics/issues.jsonl)
echo "Found $DATA_POINTS historical data points"

# Copy to analysis directory
cp /tmp/gh-aw/repo-memory/default/metrics/issues.jsonl /tmp/gh-aw/python/data/
```

## Compute Trends

Use Python to analyze trends:

```python
import json
import pandas as pd
import matplotlib.pyplot as plt

# Load historical data
data = []
with open('/tmp/gh-aw/python/data/issues.jsonl', 'r') as f:
    for line in f:
        data.append(json.loads(line))

# Convert to DataFrame
df = pd.DataFrame(data)
df['date'] = pd.to_datetime(df['date'])
df['issue_count'] = df['issues'].apply(len)

# Calculate 7-day moving average
df['ma_7d'] = df['issue_count'].rolling(window=7).mean()

# Generate trend chart
plt.figure(figsize=(12, 6))
plt.plot(df['date'], df['issue_count'], label='Daily Issues', alpha=0.5)
plt.plot(df['date'], df['ma_7d'], label='7-Day Average', linewidth=2)
plt.xlabel('Date')
plt.ylabel('Open Issues')
plt.title('Issue Trend Analysis')
plt.legend()
plt.savefig('/tmp/gh-aw/python/charts/issue-trend.png')
```

Post analysis to discussion using create-discussion tool with embedded chart.
```

> [!WARNING]
> **Access Control**
>
> Repo-memory branches are part of your repository:
> - Anyone with read access can view the data
> - Do not store sensitive information (credentials, PII)
> - Use for aggregate metrics and non-sensitive state
> - Consider using encrypted cache-memory for sensitive data

### Real-World Examples

**`.github/workflows/metrics-collector.md`** - Stores daily performance metrics in `memory/meta-orchestrators` for analysis by other workflows

**`.github/workflows/daily-code-metrics.md`** - Uses `repo-memory` to maintain historical code quality metrics and trend data

## Pattern 4: Data Caching to Avoid Throttling

Cache expensive or rate-limited API calls to improve performance and avoid throttling issues.

### When to Use This Pattern

- Workflows that make many API calls
- Data that changes infrequently
- Avoiding GitHub API rate limits
- Reducing workflow execution time
- Minimizing external service dependencies

### Implementation

```yaml wrap title=".github/workflows/report-generator.md"
---
on:
  schedule: daily
engine: copilot
tools:
  cache-memory:
    key: api-cache-${{ github.workflow }}
  github:
    toolsets: [default]
safe-outputs:
  create-discussion:
timeout-minutes: 20
---

# Report Generator with Smart Caching

Generate reports using cached data when fresh enough.

## Phase 1: Check Cache

Determine if cached data is still valid:

```bash
mkdir -p /tmp/gh-aw/cache-memory/data

CACHE_FILE="/tmp/gh-aw/cache-memory/data/repository-info.json"
CACHE_TIMESTAMP_FILE="/tmp/gh-aw/cache-memory/data/.timestamp"

# Cache validity period (in seconds)
CACHE_TTL=86400  # 24 hours

CACHE_VALID=false

if [ -f "$CACHE_TIMESTAMP_FILE" ] && [ -f "$CACHE_FILE" ]; then
  CACHE_AGE=$(($(date +%s) - $(cat "$CACHE_TIMESTAMP_FILE")))
  
  if [ $CACHE_AGE -lt $CACHE_TTL ]; then
    echo "✅ Cache is valid (age: ${CACHE_AGE}s < ${CACHE_TTL}s)"
    CACHE_VALID=true
  else
    echo "⚠️  Cache is stale (age: ${CACHE_AGE}s > ${CACHE_TTL}s)"
  fi
else
  echo "ℹ️  No cache found"
fi
```

## Phase 2: Use Cache or Fetch

Use cached data or fetch fresh data:

```bash
if [ "$CACHE_VALID" = true ]; then
  echo "Using cached data..."
  DATA=$(cat "$CACHE_FILE")
else
  echo "Fetching fresh data..."
  
  # Make expensive API calls
  gh api repos/${{ github.repository }} \
    --jq '{
      name, 
      description, 
      stargazers_count, 
      forks_count, 
      open_issues_count,
      language,
      topics
    }' > "$CACHE_FILE"
  
  # Additional data that doesn't change often
  gh api repos/${{ github.repository }}/contributors \
    --paginate > /tmp/contributors.json
  
  jq --slurpfile contrib /tmp/contributors.json \
    '. + {contributors: $contrib[0]}' \
    "$CACHE_FILE" > /tmp/combined.json
  
  mv /tmp/combined.json "$CACHE_FILE"
  
  # Update timestamp
  date +%s > "$CACHE_TIMESTAMP_FILE"
  
  echo "✅ Cache updated"
  
  DATA=$(cat "$CACHE_FILE")
fi
```

## Phase 3: Fetch Dynamic Data

Fetch data that must be fresh:

```bash
# These are fetched every time (recent activity)
gh api repos/${{ github.repository }}/issues \
  --field state=open \
  --field since=$(date -d '7 days ago' --iso-8601) \
  --paginate > /tmp/recent-issues.json

gh api repos/${{ github.repository }}/pulls \
  --field state=open \
  --paginate > /tmp/open-pulls.json
```

## Phase 4: Generate Report

Combine cached and fresh data for the report.

> [!TIP]
> **Cache Invalidation Strategies**
>
> Choose TTL based on data volatility:
> - Repository metadata: 24 hours (changes rarely)
> - Contributor list: 12 hours (changes occasionally)  
> - Issue/PR data: 1 hour (changes frequently)
> - Workflow run data: 30 minutes (very dynamic)
>
> Use different cache keys for different data types:
> ```yaml
> cache-memory:
>   - id: repo-metadata
>     key: metadata-${{ github.repository }}
>   - id: activity-data  
>     key: activity-${{ github.repository }}-${{ github.run_id }}
> ```
```

### Real-World Examples

**`.github/workflows/daily-news.md`** - Caches GitHub data fetch with 24-hour TTL to avoid re-fetching on multiple runs

**`.github/workflows/ubuntu-image-analyzer.md`** - Caches Docker image analysis data to avoid expensive re-computation

## Pattern 5: Trend Computation

Compute trends, moving averages, and statistical analysis from historical data stored in memory.

### When to Use This Pattern

- Tracking metrics over time
- Detecting anomalies or changes
- Forecasting future values
- Visualizing historical patterns
- Computing performance trends

### Implementation

```yaml wrap title=".github/workflows/performance-tracker.md"
---
on:
  schedule: daily
engine: copilot
tools:
  repo-memory:
    branch-name: memory/performance
    description: "Historical performance metrics"
    file-glob: ["history/*.jsonl", "stats/*.json"]
  bash:
imports:
  - shared/python-dataviz.md
safe-outputs:
  create-discussion:
timeout-minutes: 15
---

# Performance Trend Tracker

Track and visualize performance trends over time.

## Phase 1: Collect Current Metrics

Gather current performance data:

```bash
mkdir -p /tmp/gh-aw/repo-memory/default/history

# Collect current metrics
BUILD_TIME=$(make build 2>&1 | grep -oP 'real\s+\K\d+' || echo "0")
TEST_TIME=$(make test 2>&1 | grep -oP 'real\s+\K\d+' || echo "0")
BINARY_SIZE=$(stat -f%z ./gh-aw 2>/dev/null || stat -c%s ./gh-aw)
LOC=$(find . -name '*.go' -exec wc -l {} + | tail -1 | awk '{print $1}')

# Create data point
cat > /tmp/current-metrics.json << EOF
{
  "timestamp": $(date +%s),
  "date": "$(date +%Y-%m-%d)",
  "build_time_sec": $BUILD_TIME,
  "test_time_sec": $TEST_TIME,
  "binary_size_bytes": $BINARY_SIZE,
  "lines_of_code": $LOC,
  "git_sha": "${{ github.sha }}"
}
EOF
```

## Phase 2: Append to History

Add to historical dataset:

```bash
# Append to history file (JSON Lines format)
cat /tmp/current-metrics.json >> /tmp/gh-aw/repo-memory/default/history/metrics.jsonl

# Keep only last 90 days
tail -n 90 /tmp/gh-aw/repo-memory/default/history/metrics.jsonl > /tmp/trimmed.jsonl
mv /tmp/trimmed.jsonl /tmp/gh-aw/repo-memory/default/history/metrics.jsonl
```

## Phase 3: Compute Trends with Python

Calculate trends, moving averages, and statistics:

```python
import json
import pandas as pd
import numpy as np
from scipy import stats
import matplotlib.pyplot as plt
import seaborn as sns

# Load historical data
data = []
with open('/tmp/gh-aw/repo-memory/default/history/metrics.jsonl', 'r') as f:
    for line in f:
        data.append(json.loads(line))

df = pd.DataFrame(data)
df['date'] = pd.to_datetime(df['date'])
df = df.sort_values('date')

# Compute trends for each metric
metrics = ['build_time_sec', 'test_time_sec', 'binary_size_bytes', 'lines_of_code']

trends = {}
for metric in metrics:
    # 7-day moving average
    df[f'{metric}_ma7'] = df[metric].rolling(window=7, min_periods=1).mean()
    
    # 30-day moving average
    df[f'{metric}_ma30'] = df[metric].rolling(window=30, min_periods=1).mean()
    
    # Linear trend (slope)
    x = np.arange(len(df))
    y = df[metric].values
    slope, intercept, r_value, p_value, std_err = stats.linregress(x, y)
    
    # Percent change from 30 days ago
    if len(df) >= 30:
        change_30d = ((df[metric].iloc[-1] - df[metric].iloc[-30]) / 
                      df[metric].iloc[-30] * 100)
    else:
        change_30d = 0
    
    trends[metric] = {
        'current': float(df[metric].iloc[-1]),
        'avg_7d': float(df[f'{metric}_ma7'].iloc[-1]),
        'avg_30d': float(df[f'{metric}_ma30'].iloc[-1]),
        'slope': float(slope),
        'trend': 'improving' if slope < 0 else 'declining',
        'change_30d_percent': float(change_30d),
        'r_squared': float(r_value ** 2)
    }

# Save computed trends
with open('/tmp/gh-aw/repo-memory/default/stats/latest-trends.json', 'w') as f:
    json.dump(trends, f, indent=2)
```

## Phase 4: Generate Visualizations

Create trend charts:

```python
# Create 2x2 subplot grid
fig, axes = plt.subplots(2, 2, figsize=(16, 12))
fig.suptitle('Performance Trends', fontsize=16, fontweight='bold')

metrics_config = [
    ('build_time_sec', 'Build Time (seconds)', axes[0, 0]),
    ('test_time_sec', 'Test Time (seconds)', axes[0, 1]),
    ('binary_size_bytes', 'Binary Size (MB)', axes[1, 0]),
    ('lines_of_code', 'Lines of Code', axes[1, 1])
]

for metric, title, ax in metrics_config:
    # Plot actual values
    ax.plot(df['date'], df[metric], 
            label='Actual', alpha=0.6, marker='o', markersize=3)
    
    # Plot 7-day moving average
    ax.plot(df['date'], df[f'{metric}_ma7'], 
            label='7-day MA', linewidth=2)
    
    # Plot 30-day moving average
    ax.plot(df['date'], df[f'{metric}_ma30'], 
            label='30-day MA', linewidth=2, linestyle='--')
    
    # Add trend line
    x_numeric = np.arange(len(df))
    slope = trends[metric]['slope']
    intercept = df[metric].iloc[0]
    trend_line = slope * x_numeric + intercept
    ax.plot(df['date'], trend_line, 
            label='Trend', linewidth=1, linestyle=':', color='red')
    
    ax.set_title(title, fontweight='bold')
    ax.set_xlabel('Date')
    ax.legend(loc='best')
    ax.grid(True, alpha=0.3)
    
    # Convert to MB for binary size
    if metric == 'binary_size_bytes':
        ax.set_ylabel('Size (MB)')
        vals = ax.get_yticks()
        ax.set_yticklabels([f'{int(v/1024/1024)}' for v in vals])
    else:
        ax.set_ylabel(title)

plt.tight_layout()
plt.savefig('/tmp/gh-aw/python/charts/performance-trends.png', dpi=150)
```

## Phase 5: Generate Report

Create discussion with:
- Current metrics vs. historical averages
- Trend direction (improving/declining)
- 30-day change percentages
- Anomaly detection (values beyond 2σ from mean)
- Embedded trend charts

Post using create-discussion tool.
```

> [!TIP]
> **Advanced Trend Analysis**
>
> Implement sophisticated analytics:
> - **Seasonal decomposition** - Separate trends from seasonal patterns
> - **Change point detection** - Identify when trends shift significantly
> - **Forecasting** - Predict future values using ARIMA or Prophet
> - **Correlation analysis** - Find relationships between metrics
> - **Anomaly detection** - Flag unusual values using statistical methods

### Real-World Examples

**`.github/workflows/daily-code-metrics.md`** - Stores metrics in JSONL format and computes 7/30-day trends with Python visualizations

**`.github/workflows/shared/charts-with-trending.md`** - Provides complete setup for trend analysis with cache-memory integration

## Pattern 6: Multiple Memory Stores

Use multiple cache-memory or repo-memory instances to create complex storage architectures with different retention and access patterns.

### When to Use This Pattern

- Separating concerns (state vs. data vs. metadata)
- Different retention requirements
- Access control separation
- Performance optimization (hot vs. cold data)

### Implementation

```yaml wrap title=".github/workflows/complex-analytics.md"
---
on:
  schedule: daily
engine: copilot
tools:
  # Short-term session state
  cache-memory:
    - id: session
      key: session-${{ github.workflow }}-${{ github.run_id }}
  
  # Long-term metric history
  repo-memory:
    - id: metrics
      branch-name: memory/metrics
      file-glob: ["*.jsonl"]
      max-file-size: 204800  # 200KB
  
  # Configuration and metadata
  repo-memory:
    - id: config
      branch-name: memory/config
      file-glob: ["*.json", "*.yaml"]
      max-file-size: 51200  # 50KB
  
  # Raw data archive
  repo-memory:
    - id: archive
      branch-name: memory/archive
      file-glob: ["data/**/*.json.gz"]
      max-file-size: 524288  # 512KB

safe-outputs:
  create-discussion:
timeout-minutes: 20
---

# Complex Analytics with Multiple Memory Stores

Workflow demonstrating multiple memory stores for different purposes.

## Memory Store Organization

This workflow uses 4 memory stores:

1. **session** (cache-memory) - Temporary processing state
   - Location: `/tmp/gh-aw/cache-memory/`
   - Retention: 7 days (GitHub Actions cache)
   - Use: Intermediate calculations, API responses, temp files

2. **metrics** (repo-memory) - Historical metrics
   - Location: `/tmp/gh-aw/repo-memory/metrics/`
   - Retention: Permanent (Git history)
   - Use: Time-series data, computed statistics

3. **config** (repo-memory) - Configuration and metadata
   - Location: `/tmp/gh-aw/repo-memory/config/`
   - Retention: Permanent (Git history)
   - Use: Schema definitions, data catalogs, settings

4. **archive** (repo-memory) - Compressed raw data
   - Location: `/tmp/gh-aw/repo-memory/archive/`
   - Retention: Permanent (Git history)
   - Use: Full data backups, audit trail

## Working with Multiple Stores

### Session Cache (Fast, Temporary)

```bash
# Store intermediate API responses
gh api repos/${{ github.repository }}/issues > \
  /tmp/gh-aw/cache-memory/issues-raw.json

# Process data
jq '[.[] | select(.state == "open")]' \
  /tmp/gh-aw/cache-memory/issues-raw.json > \
  /tmp/gh-aw/cache-memory/issues-processed.json
```

### Metrics Store (Structured Time-Series)

```bash
# Append computed metrics
cat >> /tmp/gh-aw/repo-memory/metrics/daily-stats.jsonl << EOF
{"date":"$(date +%Y-%m-%d)","open_issues":$(jq length /tmp/gh-aw/cache-memory/issues-processed.json),"timestamp":$(date +%s)}
EOF
```

### Config Store (Metadata and Schema)

```bash
# Update data catalog
cat > /tmp/gh-aw/repo-memory/config/data-catalog.json << 'EOF'
{
  "datasets": {
    "daily-stats": {
      "location": "metrics/daily-stats.jsonl",
      "format": "jsonl",
      "schema": {
        "date": "YYYY-MM-DD",
        "open_issues": "integer",
        "timestamp": "unix_timestamp"
      },
      "updated": "$(date --iso-8601)"
    }
  }
}
EOF
```

### Archive Store (Compressed Backups)

```bash
# Archive full data snapshot
mkdir -p /tmp/gh-aw/repo-memory/archive/data/$(date +%Y/%m)

gzip -c /tmp/gh-aw/cache-memory/issues-raw.json > \
  /tmp/gh-aw/repo-memory/archive/data/$(date +%Y/%m)/issues-$(date +%Y%m%d).json.gz
```

## Access Patterns

**Fast reads**: Use session cache for frequently accessed data within a run

**Historical queries**: Query metrics store for trends and time-series analysis

**Schema lookup**: Check config store for data formats and metadata

**Deep dive**: Decompress archive store for detailed historical investigation
```

> [!WARNING]
> **Memory Limits**
>
> Be mindful of storage limits:
> - Cache-memory: Limited by GitHub Actions cache (10GB per repo)
> - Repo-memory: Limited by repository size quotas
> - Use `max-file-size` limits to prevent large files
> - Implement rotation/cleanup for growing datasets
> - Compress archived data with gzip

## Best Practices

### 1. Use JSON Lines for Append-Only Data

JSON Lines (`.jsonl`) format is ideal for time-series and log-style data:

```bash
# Append new data without reading entire file
echo '{"date":"2024-01-15","value":42}' >> data.jsonl

# Process with standard tools
jq -s '.' data.jsonl  # Convert to JSON array
tail -n 100 data.jsonl  # Get last 100 entries
```

### 2. Include Metadata with Datasets

Always store schema and documentation:

```json
{
  "dataset": "performance-metrics",
  "version": "1.0",
  "created": "2024-01-15T00:00:00Z",
  "schema": {
    "timestamp": "unix timestamp (seconds)",
    "build_time": "seconds (integer)",
    "test_time": "seconds (integer)"
  },
  "retention": "90 days",
  "producer": "performance-tracker workflow"
}
```

### 3. Implement Data Rotation

Prevent unbounded growth:

```bash
# Keep only last N days
MAX_DAYS=90
tail -n "$MAX_DAYS" history.jsonl > history-trimmed.jsonl
mv history-trimmed.jsonl history.jsonl

# Delete old archives
find /tmp/gh-aw/repo-memory/default/archive -type f -mtime +90 -delete
```

### 4. Use Timestamps Consistently

Always include timestamps for temporal data:

```json
{
  "timestamp": 1705334400,
  "date": "2024-01-15",
  "iso_date": "2024-01-15T14:30:00Z"
}
```

### 5. Validate Memory State

Check data integrity before processing:

```bash
# Verify file exists and is valid JSON
if [ -f state.json ] && jq empty state.json 2>/dev/null; then
  echo "State is valid"
else
  echo "State is corrupt, reinitializing..."
  echo '{}' > state.json
fi
```

### 6. Document Memory Layout

Document your memory structure in workflow or shared imports:

```markdown
## Memory Layout

### `/tmp/gh-aw/cache-memory/`
- `state/progress.json` - Todo/done tracking
- `data/cache-*.json` - Cached API responses (24h TTL)

### `/tmp/gh-aw/repo-memory/default/`  
- `history/metrics.jsonl` - Time-series metrics (90 days)
- `metadata.json` - Data schema and descriptions
```

## Privacy and Security Considerations

> [!CAUTION]
> **Sensitive Data**
>
> Memory stores are part of your repository:
> - **repo-memory** - Visible to anyone with repo read access
> - **cache-memory** - Stored in GitHub Actions cache (org-level access)
> - **Never store**: Credentials, API tokens, PII, secrets
> - **Aggregate only**: Store statistics, not raw sensitive data
> - **Consider encryption**: For sensitive but non-secret data

### Safe Data Practices

```bash
# ✅ GOOD - Aggregate statistics
echo '{"open_issues": 42, "avg_response_time": 3.5}' > metrics.json

# ❌ BAD - Individual user data
echo '{"user": "alice", "email": "alice@example.com"}' > users.json

# ✅ GOOD - Anonymized data
echo '{"user_id_hash": "a1b2c3", "activity_score": 85}' > activity.json
```

## Troubleshooting

### Cache Not Persisting

**Problem**: Cache-memory data disappears between runs

**Solutions**:
- Verify `cache-memory` is configured in frontmatter
- Check cache key is consistent across runs
- Ensure data is written before workflow completes
- Note: Cache expires after 7 days of inactivity

### Repo Memory Not Updating

**Problem**: Changes to repo-memory don't appear in branch

**Solutions**:
- Verify `repo-memory` configuration in frontmatter
- Check `file-glob` patterns match your files
- Ensure files are within `max-file-size` limit
- Verify workflow has permissions to push to branches

### Out of Memory Errors

**Problem**: Workflow fails with memory errors when loading data

**Solutions**:
- Process data in chunks instead of loading entirely
- Use streaming with `jq` or line-by-line processing
- Implement data rotation to limit file sizes
- Consider pagination for large datasets

### Merge Conflicts in Memory Branch

**Problem**: Concurrent workflows create conflicts in repo-memory branch

**Solutions**:
- Use JSON Lines format (append-only, no conflicts)
- Separate memory branches per workflow
- Add workflow run ID to filenames for uniqueness
- Use `concurrency` groups to serialize workflows

## Related Documentation

- [MCP Servers Guide](/gh-aw/guides/mcps/) - Using memory MCP server
- [Deterministic Patterns](/gh-aw/guides/deterministic-agentic-patterns/) - Data preprocessing patterns  
- [Safe Outputs](/gh-aw/guides/custom-safe-outputs/) - Storing output data
- [Python Data Visualization](/gh-aw/reference/imports/) - Shared imports for trends
- [Frontmatter Reference](/gh-aw/reference/frontmatter/) - Configuration options

## Real Workflow Examples

Explore these workflows in `.github/workflows/` for complete implementations:

- **`daily-news.md`** - Cache with 24h TTL, shared data fetch
- **`metrics-collector.md`** - Repo-memory for persistent metrics
- **`daily-code-metrics.md`** - Trend computation with Python
- **`repository-quality-improver.md`** - Focus area rotation tracking
- **`shared/mcp/server-memory.md`** - Memory MCP server configuration
- **`shared/charts-with-trending.md`** - Complete trending setup
