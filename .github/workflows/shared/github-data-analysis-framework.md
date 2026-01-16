---
# GitHub Data Analysis Framework
# Provides standardized infrastructure for workflows that analyze GitHub data
# with persistent storage and trend tracking
#
# Usage:
#   imports:
#     - shared/github-data-analysis-framework.md
#
# This import provides:
# - Structured directory setup for data analysis
# - Historical data loading (last 30 days)
# - Trend calculation helper functions
# - Standardized metrics storage format
# - Cross-platform date handling utilities

tools:
  repo-memory:
    description: "Shared data analysis storage"
    file-glob: ["*.json", "*.jsonl", "*.csv", "*.md"]
    max-file-size: 102400
  bash:
    - "jq *"
    - "date *"
    - "mkdir *"
    - "cp *"
    - "cat *"
    - "bc *"
    - "find *"
    - "ls *"
    - "wc *"

steps:
  - name: Setup analysis environment
    run: |
      # Create structured directories for data analysis
      mkdir -p /tmp/gh-aw/analysis/{data,historical,output}
      mkdir -p /tmp/gh-aw/repo-memory/default/metrics/daily
      echo "Analysis environment ready"
      echo "Current run: $(date -u +%Y-%m-%dT%H:%M:%SZ)"
      
  - name: Load historical context
    run: |
      # Load last 30 days of historical data for trend analysis
      HISTORY_DIR="/tmp/gh-aw/repo-memory/default/metrics/daily"
      if [ -d "$HISTORY_DIR" ]; then
        # Copy last 30 days of data
        find "$HISTORY_DIR" -name "*.json" -mtime -30 \
          -exec cp {} /tmp/gh-aw/analysis/historical/ \; 2>/dev/null || true
        
        HIST_COUNT=$(ls -1 /tmp/gh-aw/analysis/historical/ 2>/dev/null | wc -l)
        echo "Loaded $HIST_COUNT days of historical data"
      else
        echo "No historical data found (first run)"
      fi
---

# GitHub Data Analysis Framework

This shared component provides a standardized framework for workflows that analyze GitHub data with persistent storage and trend tracking.

## Features

- **Persistent storage** with repo-memory for historical data
- **Structured directories** for organized data management
- **Historical data loading** (last 30 days automatically)
- **Trend calculation helpers** for 7-day and 30-day comparisons
- **Standardized metrics storage** format

## Directory Structure

```
/tmp/gh-aw/analysis/
├── data/          # Current run data and inputs
├── historical/    # Last 30 days for comparison
└── output/        # Analysis results and reports

/tmp/gh-aw/repo-memory/default/
└── metrics/
    └── daily/     # Daily metrics stored as YYYY-MM-DD.json
```

## Trend Calculation Helpers

The following bash functions are available for calculating trends:

### calculate_trend

Calculate percentage change between two values:

```bash
# Calculate percentage change between two values
calculate_trend() {
  local current=$1
  local previous=$2
  
  if [ -z "$previous" ] || [ "$previous" = "0" ] || [ "$previous" = "null" ]; then
    echo "N/A"
    return
  fi
  
  local change=$(echo "scale=2; (($current - $previous) / $previous) * 100" | bc)
  printf "%.1f" "$change"
}

# Usage example:
current_value=100
previous_value=80
trend=$(calculate_trend "$current_value" "$previous_value")
echo "Trend: ${trend}%"  # Output: Trend: 25.0%
```

### get_trend_indicator

Get trend indicator emoji based on percentage change:

```bash
# Get trend indicator emoji
get_trend_indicator() {
  local change=$1
  
  if [ "$change" = "N/A" ]; then
    echo "➡️"
  elif (( $(echo "$change > 10" | bc -l) )); then
    echo "⬆️"
  elif (( $(echo "$change < -10" | bc -l) )); then
    echo "⬇️"
  else
    echo "➡️"
  fi
}

# Usage example:
trend_7d=25.0
indicator=$(get_trend_indicator "$trend_7d")
echo "7-day trend: ${indicator} ${trend_7d}%"  # Output: 7-day trend: ⬆️ 25.0%
```

### get_historical_value

Get value from N days ago from historical data:

```bash
# Get value from N days ago
get_historical_value() {
  local metric_path=$1
  local days_ago=$2
  
  # Cross-platform date handling (GNU date first, BSD fallback)
  local target_date=$(date -d "$days_ago days ago" '+%Y-%m-%d' 2>/dev/null || \
                      date -v-${days_ago}d '+%Y-%m-%d')
  local hist_file="/tmp/gh-aw/analysis/historical/${target_date}.json"
  
  if [ -f "$hist_file" ]; then
    jq -r "$metric_path // null" "$hist_file"
  else
    echo "null"
  fi
}

# Usage example:
value_7d_ago=$(get_historical_value '.metrics.total_count' 7)
value_30d_ago=$(get_historical_value '.metrics.total_count' 30)
echo "7 days ago: $value_7d_ago"
echo "30 days ago: $value_30d_ago"
```

## Storage Pattern

Store daily metrics in standardized format:

```bash
# Store current run metrics
TODAY=$(date +%Y-%m-%d)
METRICS_FILE="/tmp/gh-aw/repo-memory/default/metrics/daily/${TODAY}.json"

# Example: Create metrics JSON with jq
jq -n \
  --arg date "$TODAY" \
  --arg timestamp "$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
  --argjson metrics "$metrics_json" \
  '{
    date: $date,
    timestamp: $timestamp,
    metrics: $metrics
  }' > "$METRICS_FILE"

echo "Metrics stored: $METRICS_FILE"
```

### Metrics JSON Structure

All metrics should follow this structure:

```json
{
  "date": "2026-01-16",
  "timestamp": "2026-01-16T12:00:00Z",
  "metrics": {
    "total_count": 100,
    "open_count": 45,
    "closed_count": 55,
    "custom_metric_1": 42,
    "custom_metric_2": "some_value"
  }
}
```

## Trend Analysis Pattern

Complete example of calculating trends:

```bash
# Calculate 7-day and 30-day trends
current_value=100
value_7d_ago=$(get_historical_value '.metrics.total_count' 7)
value_30d_ago=$(get_historical_value '.metrics.total_count' 30)

trend_7d=$(calculate_trend "$current_value" "$value_7d_ago")
trend_30d=$(calculate_trend "$current_value" "$value_30d_ago")

indicator_7d=$(get_trend_indicator "$trend_7d")
indicator_30d=$(get_trend_indicator "$trend_30d")

echo "Current: $current_value"
echo "7-day change: ${indicator_7d} ${trend_7d}%"
echo "30-day change: ${indicator_30d} ${trend_30d}%"
```

## Complete Usage Example

Here's a complete workflow using the framework:

```yaml
---
imports:
  - shared/github-data-analysis-framework.md
  - shared/issues-data-fetch.md
  
tools:
  github:
    toolsets: [default]

timeout-minutes: 20
---

# Your workflow prompt

## Step 1: Load and Analyze Data

# Analysis environment is already set up by the framework
# Load your current data
cp /tmp/gh-aw/issues-data/issues.json /tmp/gh-aw/analysis/data/

# Perform analysis
current_total=$(jq 'length' /tmp/gh-aw/analysis/data/issues.json)
current_open=$(jq '[.[] | select(.state == "OPEN")] | length' /tmp/gh-aw/analysis/data/issues.json)

## Step 2: Calculate Trends

# Get historical values
total_7d_ago=$(get_historical_value '.metrics.total_issues' 7)
total_30d_ago=$(get_historical_value '.metrics.total_issues' 30)

# Calculate trends
trend_7d=$(calculate_trend "$current_total" "$total_7d_ago")
trend_30d=$(calculate_trend "$current_total" "$total_30d_ago")

# Get indicators
indicator_7d=$(get_trend_indicator "$trend_7d")
indicator_30d=$(get_trend_indicator "$trend_30d")

## Step 3: Store Results

TODAY=$(date +%Y-%m-%d)
METRICS_FILE="/tmp/gh-aw/repo-memory/default/metrics/daily/${TODAY}.json"

jq -n \
  --arg date "$TODAY" \
  --arg timestamp "$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
  --argjson total "$current_total" \
  --argjson open "$current_open" \
  --arg trend_7d "$trend_7d" \
  --arg trend_30d "$trend_30d" \
  '{
    date: $date,
    timestamp: $timestamp,
    metrics: {
      total_issues: $total,
      open_issues: $open,
      trend_7d: $trend_7d,
      trend_30d: $trend_30d
    }
  }' > "$METRICS_FILE"

echo "Metrics stored to: $METRICS_FILE"

## Step 4: Generate Report

echo "# Daily Report - $TODAY"
echo ""
echo "## Metrics"
echo "- Total issues: $current_total"
echo "- Open issues: $current_open"
echo "- 7-day trend: ${indicator_7d} ${trend_7d}%"
echo "- 30-day trend: ${indicator_30d} ${trend_30d}%"
```

## Best Practices

1. **Consistent metric names**: Use same field names across runs for trend tracking
2. **Date-based files**: Always use YYYY-MM-DD.json format for daily metrics
3. **Null handling**: Check for null/missing historical data before calculations
4. **Cross-platform**: Use compatible date commands (GNU date with BSD fallback)
5. **Validation**: Verify historical files exist before loading
6. **bc availability**: Ensure `bc` is installed for floating-point calculations
7. **Error handling**: Gracefully handle missing historical data (first run)

## Cross-Platform Compatibility

### Date Commands

Always use the cross-platform date pattern:

```bash
# GNU date (Linux) first, then BSD date (macOS) fallback
TARGET_DATE=$(date -d "7 days ago" '+%Y-%m-%d' 2>/dev/null || \
              date -v-7d '+%Y-%m-%d')
```

### bc Commands

For floating-point calculations, use `bc` with proper scale:

```bash
# Calculate with 2 decimal places
result=$(echo "scale=2; 100 / 3" | bc)
# Output: 33.33

# Format output with printf
formatted=$(printf "%.1f" "$result")
```

## Helper Functions in Action

Copy this into your workflow for easy access to all helpers:

```bash
# Calculate percentage change between two values
calculate_trend() {
  local current=$1
  local previous=$2
  
  if [ -z "$previous" ] || [ "$previous" = "0" ] || [ "$previous" = "null" ]; then
    echo "N/A"
    return
  fi
  
  local change=$(echo "scale=2; (($current - $previous) / $previous) * 100" | bc)
  printf "%.1f" "$change"
}

# Get trend indicator emoji
get_trend_indicator() {
  local change=$1
  
  if [ "$change" = "N/A" ]; then
    echo "➡️"
  elif (( $(echo "$change > 10" | bc -l) )); then
    echo "⬆️"
  elif (( $(echo "$change < -10" | bc -l) )); then
    echo "⬇️"
  else
    echo "➡️"
  fi
}

# Get value from N days ago
get_historical_value() {
  local metric_path=$1
  local days_ago=$2
  
  local target_date=$(date -d "$days_ago days ago" '+%Y-%m-%d' 2>/dev/null || \
                      date -v-${days_ago}d '+%Y-%m-%d')
  local hist_file="/tmp/gh-aw/analysis/historical/${target_date}.json"
  
  if [ -f "$hist_file" ]; then
    jq -r "$metric_path // null" "$hist_file"
  else
    echo "null"
  fi
}
```

## Troubleshooting

### Missing Historical Data

On first run or when historical data is unavailable:

```bash
value=$(get_historical_value '.metrics.count' 7)
if [ "$value" = "null" ]; then
  echo "No historical data available for comparison"
  # Skip trend calculation or use fallback value
fi
```

### bc Not Found

If `bc` is not available (rare):

```bash
if ! command -v bc &> /dev/null; then
  echo "bc is not installed, skipping trend calculations"
  # Use alternative approach or skip trends
fi
```

### Date Command Differences

The framework uses cross-platform date handling. If you need custom date operations:

```bash
# Always try GNU date first, then BSD date
custom_date=$(date -d "3 months ago" '+%Y-%m-%d' 2>/dev/null || \
              date -v-3m '+%Y-%m-%d')
```

## Integration with Other Shared Components

This framework works well with:

- **shared/issues-data-fetch.md** - Fetch issues data for analysis
- **shared/copilot-pr-data-fetch.md** - Fetch PR data for analysis
- **shared/trends.md** - Advanced visualization with Python
- **shared/python-dataviz.md** - Create charts from metrics
- **shared/reporting.md** - Format reports with metrics

Example combining multiple shared components:

```yaml
imports:
  - shared/github-data-analysis-framework.md
  - shared/issues-data-fetch.md
  - shared/python-dataviz.md
  - shared/reporting.md
```
