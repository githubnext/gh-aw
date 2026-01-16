---
description: Test workflow to validate the GitHub Data Analysis Framework
on: workflow_dispatch
permissions:
  contents: read
engine: copilot
timeout-minutes: 5
imports:
  - shared/github-data-analysis-framework.md
---

# Test GitHub Data Analysis Framework

This is a test workflow to validate the shared GitHub Data Analysis Framework component.

## Test 1: Verify Directory Setup

Check that the analysis directories were created:

```bash
echo "Checking directory structure..."
ls -la /tmp/gh-aw/analysis/
ls -la /tmp/gh-aw/repo-memory/default/metrics/
echo "✓ Directory structure verified"
```

## Test 2: Test Helper Functions

Define and test the helper functions:

```bash
# Define helper functions
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

# Test trend calculation
echo "Testing trend calculations..."
trend1=$(calculate_trend 100 80)
echo "Trend (100 vs 80): ${trend1}% (expected: 25.0%)"

trend2=$(calculate_trend 80 100)
echo "Trend (80 vs 100): ${trend2}% (expected: -20.0%)"

trend3=$(calculate_trend 100 0)
echo "Trend (100 vs 0): ${trend3} (expected: N/A)"

# Test indicators
indicator1=$(get_trend_indicator "25.0")
echo "Indicator for +25%: ${indicator1} (expected: ⬆️)"

indicator2=$(get_trend_indicator "-25.0")
echo "Indicator for -25%: ${indicator2} (expected: ⬇️)"

indicator3=$(get_trend_indicator "5.0")
echo "Indicator for +5%: ${indicator3} (expected: ➡️)"

echo "✓ Helper functions tested"
```

## Test 3: Test Metrics Storage

Create and store sample metrics:

```bash
echo "Testing metrics storage..."

TODAY=$(date +%Y-%m-%d)
METRICS_FILE="/tmp/gh-aw/repo-memory/default/metrics/daily/${TODAY}.json"

# Create sample metrics
jq -n \
  --arg date "$TODAY" \
  --arg timestamp "$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
  --argjson total 42 \
  --argjson active 15 \
  '{
    date: $date,
    timestamp: $timestamp,
    metrics: {
      total_items: $total,
      active_items: $active
    }
  }' > "$METRICS_FILE"

echo "Metrics stored to: $METRICS_FILE"

# Verify stored metrics
if [ -f "$METRICS_FILE" ]; then
  echo "✓ Metrics file created successfully"
  jq '.' "$METRICS_FILE"
else
  echo "✗ Failed to create metrics file"
  exit 1
fi
```

## Test 4: Test Historical Data Loading

Test loading historical data:

```bash
echo "Testing historical data loading..."

# Create some historical data for testing (simulate 3 days ago)
for days_ago in 3 7 15; do
  hist_date=$(date -d "$days_ago days ago" '+%Y-%m-%d' 2>/dev/null || \
              date -v-${days_ago}d '+%Y-%m-%d')
  hist_file="/tmp/gh-aw/repo-memory/default/metrics/daily/${hist_date}.json"
  
  jq -n \
    --arg date "$hist_date" \
    --argjson total "$((40 + days_ago))" \
    '{
      date: $date,
      metrics: {
        total_items: $total
      }
    }' > "$hist_file"
  
  echo "Created historical data: $hist_file"
done

# Trigger historical data loading (re-run the load step conceptually)
HISTORY_DIR="/tmp/gh-aw/repo-memory/default/metrics/daily"
if [ -d "$HISTORY_DIR" ]; then
  find "$HISTORY_DIR" -name "*.json" -mtime -30 \
    -exec cp {} /tmp/gh-aw/analysis/historical/ \; 2>/dev/null || true
  
  HIST_COUNT=$(ls -1 /tmp/gh-aw/analysis/historical/ 2>/dev/null | wc -l)
  echo "Loaded $HIST_COUNT days of historical data"
  ls -l /tmp/gh-aw/analysis/historical/
  echo "✓ Historical data loading verified"
fi
```

## Test 5: Complete Integration Test

Run a complete analysis workflow:

```bash
echo "Running complete integration test..."

# Define helper functions
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

# Current metrics
current_total=42

# Get historical values
total_7d_ago=$(get_historical_value '.metrics.total_items' 7)
echo "7 days ago: $total_7d_ago"

# Calculate trend
if [ "$total_7d_ago" != "null" ]; then
  trend_7d=$(calculate_trend "$current_total" "$total_7d_ago")
  indicator_7d=$(get_trend_indicator "$trend_7d")
  
  echo ""
  echo "=== Analysis Results ==="
  echo "Current total: $current_total"
  echo "7 days ago: $total_7d_ago"
  echo "7-day trend: ${indicator_7d} ${trend_7d}%"
  echo "✓ Complete integration test passed"
else
  echo "Note: No historical data from 7 days ago (expected on first run)"
fi
```

## Success Criteria

All tests should pass:
- ✓ Directory structure created
- ✓ Helper functions work correctly
- ✓ Metrics storage successful
- ✓ Historical data loading works
- ✓ Complete integration test passes

If any test fails, the framework needs adjustment.
