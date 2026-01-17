---
name: Test Pattern Implementation
description: Test workflow demonstrating proven patterns from research
on:
  workflow_dispatch:

permissions:
  contents: read

engine: copilot

tracker-id: test-patterns

# Pattern 1: repo-memory for baselines (performance tracking)
# Pattern 2: noop for healthy states (prevent spam)
# Pattern 3: smart duplicate prevention (via cache-memory)
tools:
  repo-memory:
    branch-name: memory/test-baselines
    create-orphan: true
  cache-memory: true
  bash:
    - "echo"
    - "cat"
    - "date"
    - "jq"

safe-outputs:
  create-issue:
    title-prefix: "[test-patterns] "
    labels: [test, automated]
    max: 2
  # noop is automatically enabled with create-issue

timeout-minutes: 10
strict: true
---

# Test Pattern Implementation

This workflow demonstrates the proven patterns from the research:

## Instructions

### 1. Load Baseline from Repo-Memory (Pattern 1)

Check if baseline exists in repo-memory:

```bash
BASELINE_FILE="/tmp/gh-aw/repo-memory/default/metrics.json"

if [ -f "$BASELINE_FILE" ]; then
  echo "Found existing baseline"
  BASELINE_VALUE=$(cat "$BASELINE_FILE" | jq -r '.value')
else
  echo "No baseline found - first run"
  BASELINE_VALUE=100
fi

echo "Baseline value: $BASELINE_VALUE"
```

### 2. Simulate Current Measurement

```bash
# Simulate a measurement (random value between 90-110)
CURRENT_VALUE=$((RANDOM % 21 + 90))
echo "Current measurement: $CURRENT_VALUE"
```

### 3. Smart Duplicate Prevention (Pattern 5)

Check cache to avoid creating duplicate issues:

```bash
CACHE_FILE="/tmp/gh-aw/cache-memory/issue-hashes.json"

# Initialize cache if missing
if [ ! -f "$CACHE_FILE" ]; then
  echo '{"issues": []}' > "$CACHE_FILE"
fi

# Clean old entries (>30 days)
NOW=$(date +%s)
jq --arg now "$NOW" '
  .issues |= map(select(
    ($now | tonumber) - (.created | tonumber) < 2592000
  ))
' "$CACHE_FILE" > "$CACHE_FILE.tmp" && mv "$CACHE_FILE.tmp" "$CACHE_FILE"
```

### 4. Compare and Decide

```bash
# Check if value exceeds baseline by 20% (regression)
THRESHOLD=$(echo "$BASELINE_VALUE * 1.2" | bc)

if (( $(echo "$CURRENT_VALUE > $THRESHOLD" | bc -l) )); then
  echo "⚠️ Regression detected: $CURRENT_VALUE > $THRESHOLD"
  
  # Hash issue content for deduplication
  ISSUE_CONTENT="Regression detected: $CURRENT_VALUE vs $BASELINE_VALUE"
  HASH=$(echo -n "$ISSUE_CONTENT" | sha256sum | cut -d' ' -f1)
  
  # Check if already reported
  if jq -e --arg hash "$HASH" '.issues[] | select(.hash == $hash)' "$CACHE_FILE" > /dev/null 2>&1; then
    echo "⏭️ Skipping duplicate issue (hash: ${HASH:0:8})"
  else
    echo "✅ Creating new issue"
    echo "{
      \"type\": \"create-issue\",
      \"title\": \"[test-patterns] Regression Detected\",
      \"body\": \"$ISSUE_CONTENT\"
    }"
    
    # Add to cache
    jq --arg hash "$HASH" --arg now "$NOW" '
      .issues += [{
        "hash": $hash,
        "created": ($now | tonumber)
      }]
    ' "$CACHE_FILE" > "$CACHE_FILE.tmp" && mv "$CACHE_FILE.tmp" "$CACHE_FILE"
  fi
else
  # Pattern 2: Use noop for healthy state
  echo "✓ Measurement within acceptable range"
  echo "{
    \"type\": \"noop\",
    \"message\": \"Metrics healthy: $CURRENT_VALUE (baseline: $BASELINE_VALUE, threshold: $THRESHOLD)\"
  }"
fi
```

### 5. Update Baseline in Repo-Memory

```bash
# Store current measurement as new baseline
echo "{
  \"value\": $CURRENT_VALUE,
  \"updated\": \"$(date -I)\",
  \"previous\": $BASELINE_VALUE
}" > "$BASELINE_FILE"

echo "Updated baseline to $CURRENT_VALUE"
```

## Summary

This workflow demonstrates:
- ✅ **Pattern 1**: repo-memory for baseline tracking
- ✅ **Pattern 2**: noop for healthy states
- ✅ **Pattern 5**: Smart duplicate prevention with 30-day cache

The workflow will only create an issue if:
1. A regression is detected (>20% increase)
2. The issue hasn't been reported in the last 30 days
