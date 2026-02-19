---
# Token Cost Analysis
# Shared module for analyzing token consumption and costs
#
# Usage:
#   imports:
#     - shared/token-cost-analysis.md
#
# This import provides:
# - Python environment for data analysis
# - Token aggregation patterns
# - Cost calculation methods
# - Historical tracking patterns

imports:
  - shared/python-dataviz.md
---

# Token Cost Analysis Patterns

Patterns for processing and analyzing token consumption data from Copilot workflows.

## Data Processing Patterns

### Extract Per-Workflow Metrics

Create aggregated statistics by workflow:

```python
#!/usr/bin/env python3
"""Process Copilot workflow logs and calculate per-workflow statistics"""
import json
import os
from collections import defaultdict

# Load the logs
with open('/tmp/gh-aw/copilot-logs.json', 'r') as f:
    runs = json.load(f)

print(f"Processing {len(runs)} workflow runs...")

# Aggregate by workflow
workflow_stats = defaultdict(lambda: {
    'total_tokens': 0,
    'total_cost': 0.0,
    'total_turns': 0,
    'run_count': 0,
    'total_duration_seconds': 0,
    'runs': []
})

for run in runs:
    workflow_name = run.get('WorkflowName', 'unknown')
    tokens = run.get('TokenUsage', 0)
    cost = run.get('EstimatedCost', 0.0)
    turns = run.get('Turns', 0)
    duration = run.get('Duration', 0)  # in nanoseconds
    created_at = run.get('CreatedAt', '')
    
    workflow_stats[workflow_name]['total_tokens'] += tokens
    workflow_stats[workflow_name]['total_cost'] += cost
    workflow_stats[workflow_name]['total_turns'] += turns
    workflow_stats[workflow_name]['run_count'] += 1
    workflow_stats[workflow_name]['total_duration_seconds'] += duration / 1e9
    
    workflow_stats[workflow_name]['runs'].append({
        'date': created_at[:10],
        'tokens': tokens,
        'cost': cost,
        'turns': turns,
        'run_id': run.get('DatabaseID', run.get('Number', 0))
    })

# Calculate averages and save
output = []
for workflow, stats in workflow_stats.items():
    count = stats['run_count']
    output.append({
        'workflow': workflow,
        'total_tokens': stats['total_tokens'],
        'total_cost': stats['total_cost'],
        'total_turns': stats['total_turns'],
        'run_count': count,
        'avg_tokens': stats['total_tokens'] / count if count > 0 else 0,
        'avg_cost': stats['total_cost'] / count if count > 0 else 0,
        'avg_turns': stats['total_turns'] / count if count > 0 else 0,
        'avg_duration_seconds': stats['total_duration_seconds'] / count if count > 0 else 0,
        'runs': stats['runs']
    })

# Sort by total cost (highest first)
output.sort(key=lambda x: x['total_cost'], reverse=True)

# Save processed data
os.makedirs('/tmp/gh-aw/python/data', exist_ok=True)
with open('/tmp/gh-aw/python/data/workflow_stats.json', 'w') as f:
    json.dump(output, f, indent=2)

print(f"✅ Processed {len(output)} unique workflows")
```

### Store Historical Data

Append today's metrics to persistent cache for trend tracking:

```python
#!/usr/bin/env python3
"""Store today's metrics in cache memory for historical tracking"""
import json
import os
from datetime import datetime

# Load processed workflow stats
with open('/tmp/gh-aw/python/data/workflow_stats.json', 'r') as f:
    workflow_stats = json.load(f)

# Prepare today's summary
today = datetime.now().strftime('%Y-%m-%d')
today_summary = {
    'date': today,
    'timestamp': datetime.now().isoformat(),
    'workflows': {}
}

# Aggregate totals
total_tokens = 0
total_cost = 0.0
total_runs = 0

for workflow in workflow_stats:
    workflow_name = workflow['workflow']
    today_summary['workflows'][workflow_name] = {
        'tokens': workflow['total_tokens'],
        'cost': workflow['total_cost'],
        'runs': workflow['run_count'],
        'avg_tokens': workflow['avg_tokens'],
        'avg_cost': workflow['avg_cost']
    }
    total_tokens += workflow['total_tokens']
    total_cost += workflow['total_cost']
    total_runs += workflow['run_count']

today_summary['totals'] = {
    'tokens': total_tokens,
    'cost': total_cost,
    'runs': total_runs
}

# Ensure memory directory exists
memory_dir = '/tmp/gh-aw/repo-memory-default/memory/default'
os.makedirs(memory_dir, exist_ok=True)

# Append to history (JSON Lines format)
history_file = f'{memory_dir}/history.jsonl'
with open(history_file, 'a') as f:
    f.write(json.dumps(today_summary) + '\n')

print(f"✅ Stored metrics for {today}")
print(f"📈 Total tokens: {total_tokens:,}")
print(f"💰 Total cost: ${total_cost:.2f}")
print(f"🔄 Total runs: {total_runs}")
```

### Prepare Data for Visualization

Create CSV files for trend chart generation:

```python
#!/usr/bin/env python3
"""Prepare CSV data for trend charts"""
import json
import os
import pandas as pd
from datetime import datetime

# Load historical data from repo memory
memory_dir = '/tmp/gh-aw/repo-memory-default/memory/default'
history_file = f'{memory_dir}/history.jsonl'

historical_data = []
if os.path.exists(history_file):
    with open(history_file, 'r') as f:
        for line in f:
            if line.strip():
                historical_data.append(json.loads(line))

# Load today's data if needed
if not historical_data:
    with open('/tmp/gh-aw/python/data/workflow_stats.json', 'r') as f:
        workflow_stats = json.load(f)
    
    today = datetime.now().strftime('%Y-%m-%d')
    historical_data = [{
        'date': today,
        'totals': {
            'tokens': sum(w['total_tokens'] for w in workflow_stats),
            'cost': sum(w['total_cost'] for w in workflow_stats),
            'runs': sum(w['run_count'] for w in workflow_stats)
        }
    }]

# Create daily aggregates DataFrame
daily_data = []
for entry in historical_data:
    daily_data.append({
        'date': entry['date'],
        'tokens': entry['totals']['tokens'],
        'cost': entry['totals']['cost'],
        'runs': entry['totals']['runs']
    })

df = pd.DataFrame(daily_data)
df.to_csv('/tmp/gh-aw/python/data/daily_trends.csv', index=False)

print(f"✅ Prepared trend data: {len(df)} days")
```

## Chart Generation Patterns

### Token Usage Trends Chart

```python
import matplotlib.pyplot as plt
import seaborn as sns
import pandas as pd

# Load data
df = pd.read_csv('/tmp/gh-aw/python/data/daily_trends.csv')
df['date'] = pd.to_datetime(df['date'])

# Create chart
fig, ax1 = plt.subplots(figsize=(12, 7), dpi=300)
sns.set_style("whitegrid")

# Plot tokens
ax1.plot(df['date'], df['tokens'], marker='o', color='#4ECDC4', 
         linewidth=2.5, label='Token Usage')
ax1.set_xlabel('Date', fontsize=12, fontweight='bold')
ax1.set_ylabel('Tokens', fontsize=12, fontweight='bold')
ax1.tick_params(axis='y')

# Create secondary axis for cost
ax2 = ax1.twinx()
ax2.plot(df['date'], df['cost'], marker='s', color='#FF6B6B', 
         linewidth=2.5, label='Cost (USD)')
ax2.set_ylabel('Cost (USD)', fontsize=12, fontweight='bold')

# Add title and legend
plt.title('Copilot Token Usage and Cost Trends', fontsize=16, fontweight='bold')
lines1, labels1 = ax1.get_legend_handles_labels()
lines2, labels2 = ax2.get_legend_handles_labels()
ax1.legend(lines1 + lines2, labels1 + labels2, loc='upper left')

# Format x-axis
plt.xticks(rotation=45)
plt.tight_layout()

# Save
plt.savefig('/tmp/gh-aw/python/charts/token_trends.png', 
            dpi=300, bbox_inches='tight', facecolor='white')
```

### Top Workflows Bar Chart

```python
# Load workflow stats
with open('/tmp/gh-aw/python/data/workflow_stats.json', 'r') as f:
    workflows = json.load(f)

# Get top 10 workflows by cost
top_workflows = workflows[:10]
names = [w['workflow'][:30] for w in top_workflows]  # Truncate long names
costs = [w['total_cost'] for w in top_workflows]

# Create bar chart
fig, ax = plt.subplots(figsize=(12, 8), dpi=300)
bars = ax.barh(names, costs, color=sns.color_palette("husl", len(names)))

# Customize
ax.set_xlabel('Total Cost (USD)', fontsize=12, fontweight='bold')
ax.set_title('Top 10 Workflows by Token Cost', fontsize=16, fontweight='bold')
ax.grid(True, alpha=0.3, axis='x')

# Add value labels
for bar in bars:
    width = bar.get_width()
    ax.text(width, bar.get_y() + bar.get_height()/2, 
            f'${width:.2f}', ha='left', va='center', fontweight='bold')

plt.tight_layout()
plt.savefig('/tmp/gh-aw/python/charts/top_workflows.png',
            dpi=300, bbox_inches='tight', facecolor='white')
```

## Usage Workflow

1. **Process logs**: Run per-workflow metrics extraction
2. **Store history**: Append today's data to cache
3. **Prepare visualization data**: Create CSV files
4. **Generate charts**: Create trend and comparison charts
5. **Upload assets**: Use upload-asset tool to publish charts
6. **Create report**: Include charts in discussion/issue

## Key Metrics

- **Token Usage**: Total and per-workflow token consumption
- **Cost**: Estimated costs based on token usage
- **Turns**: Number of agent turns (conversation rounds)
- **Duration**: Time spent in workflow execution
- **Runs**: Number of workflow executions

## Historical Tracking

Store data in JSON Lines format for efficient append operations:
- **Location**: `/tmp/gh-aw/repo-memory-default/memory/default/history.jsonl`
- **Format**: One JSON object per line, one entry per day
- **Retention**: Keep full history (managed by repo-memory tool)
