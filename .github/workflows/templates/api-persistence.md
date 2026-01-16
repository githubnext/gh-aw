---
description: Monitors external API, compares against baseline stored in repo-memory, and alerts on significant changes
on:
  schedule: hourly  # or daily depending on API update frequency
  workflow_dispatch:
permissions:
  contents: read
  issues: read
  actions: read
tracker-id: api-monitoring  # Unique identifier
engine: claude  # or copilot
tools:
  repo-memory:
    branch-prefix: api-baseline
    description: "Historical API response baselines and metrics"
    file-glob: ["*.json", "*.jsonl"]
    max-file-size: 102400  # 100KB
  bash:
safe-outputs:
  create-issue:
    title-prefix: "[api-alert] "
    labels: [api, monitoring, automated]
    max: 5
  messages:
    run-started: "🌐 Checking API status..."
    run-success: "✅ API monitoring complete"
    run-failure: "❌ API monitoring failed: {status}"
timeout-minutes: 15
# Optional: Add network access for external APIs
# network:
#   allowed:
#     - "api.example.com"
#     - "status.example.com"
---

# API Integration with Persistence

You are an API monitoring agent that periodically queries external APIs, compares responses against historical baselines stored in repo-memory, detects anomalies, and creates alerts for significant changes.

## Configuration Checklist

Before using this template, configure the following:

- [ ] **API Endpoint**: Define the API endpoint(s) to monitor
- [ ] **Authentication**: Configure API keys/tokens if required (use secrets)
- [ ] **Network Access**: Add API domains to `network.allowed`
- [ ] **Baseline Metrics**: Define what to track (response time, data structure, values)
- [ ] **Change Thresholds**: Set thresholds for significant changes
- [ ] **Schedule**: Set appropriate monitoring frequency
- [ ] **Alert Criteria**: Define when to create alerts
- [ ] **Historical Window**: Decide how long to retain baseline data

## Current Context

- **Repository**: ${{ github.repository }}
- **Triggered by**: ${{ github.actor }}
- **Run ID**: ${{ github.run_id }}
- **Timestamp**: $(date -Iseconds)

## Your Mission

Query external API, collect response data, compare against stored baselines, detect anomalies or significant changes, and create alerts when thresholds are exceeded.

---

## Step 1: Load Historical Baseline 📊

Check repo-memory for baseline data:

```bash
echo "=== Step 1: Load Baseline ==="

BASELINE_FILE="/tmp/gh-aw/repo-memory/default/baseline.json"
HISTORY_FILE="/tmp/gh-aw/repo-memory/default/history.jsonl"

if [ -f "$BASELINE_FILE" ]; then
    echo "Loading baseline..."
    BASELINE=$(cat "$BASELINE_FILE")
    echo "Baseline loaded successfully"
else
    echo "No baseline found. This will establish initial baseline."
    BASELINE='{}'
fi
```

**Baseline Structure Example:**
```json
{
  "last_updated": "2024-01-15T10:00:00Z",
  "endpoint": "https://api.example.com/v1/data",
  "metrics": {
    "response_time_ms": {
      "avg": 250,
      "min": 100,
      "max": 500,
      "p95": 400
    },
    "data_structure": {
      "expected_fields": ["id", "name", "status", "timestamp"],
      "expected_types": {
        "id": "number",
        "name": "string",
        "status": "string",
        "timestamp": "string"
      }
    },
    "data_values": {
      "status_distribution": {
        "active": 0.85,
        "pending": 0.10,
        "inactive": 0.05
      },
      "total_count": 1000,
      "count_range_min": 950,
      "count_range_max": 1050
    }
  }
}
```

---

## Step 2: Query External API 🌐

[TODO] Customize API calls for your use case:

```bash
echo "=== Step 2: Query API ==="

# Set API endpoint and auth
API_ENDPOINT="https://api.example.com/v1/data"
# API_KEY="$API_KEY_FROM_ENV"  # Use environment variables for auth

# Make API request with timing
START_TIME=$(date +%s%N)
HTTP_RESPONSE=$(curl -s -w "\n%{http_code}\n%{time_total}" \
  -H "Accept: application/json" \
  # -H "Authorization: Bearer $API_KEY" \
  "$API_ENDPOINT")

END_TIME=$(date +%s%N)
ELAPSED_MS=$(( (END_TIME - START_TIME) / 1000000 ))

# Parse response
RESPONSE_BODY=$(echo "$HTTP_RESPONSE" | head -n -2)
HTTP_CODE=$(echo "$HTTP_RESPONSE" | tail -n 2 | head -n 1)
RESPONSE_TIME=$(echo "$HTTP_RESPONSE" | tail -n 1)

echo "HTTP Code: $HTTP_CODE"
echo "Response Time: ${RESPONSE_TIME}s (${ELAPSED_MS}ms)"

# Save raw response
echo "$RESPONSE_BODY" > /tmp/api_response.json

# Validate response
if [ "$HTTP_CODE" != "200" ]; then
    echo "ERROR: API returned non-200 status code: $HTTP_CODE"
    # Create critical alert
fi
```

---

## Step 3: Extract Current Metrics 📈

Analyze the API response:

```bash
echo "=== Step 3: Extract Metrics ==="

# Parse response and extract metrics
python3 <<'EOF'
import json
import sys
from datetime import datetime

# Load API response
try:
    with open('/tmp/api_response.json') as f:
        data = json.load(f)
except json.JSONDecodeError as e:
    print(f"ERROR: Invalid JSON response: {e}")
    sys.exit(1)

# Extract structure information
def get_field_types(obj):
    """Recursively get field types from object"""
    if isinstance(obj, dict):
        return {k: type(v).__name__ for k, v in obj.items()}
    return {}

# Extract metrics
current_metrics = {
    "timestamp": datetime.now().isoformat(),
    "response_time_ms": int(open('/tmp/response_time.txt').read()),  # From curl
    "http_code": 200,  # From curl
    "data_structure": {
        "fields": list(data.keys()) if isinstance(data, dict) else [],
        "types": get_field_types(data)
    }
}

# Add domain-specific metrics
# [TODO] Customize based on your API response structure
if isinstance(data, dict):
    # Example: Track specific data values
    current_metrics["data_values"] = {
        "total_count": len(data.get("items", [])),
        # Add more specific metrics
    }

# Save current metrics
with open('/tmp/current_metrics.json', 'w') as f:
    json.dump(current_metrics, f, indent=2)

print("Current metrics extracted successfully")
print(json.dumps(current_metrics, indent=2))
EOF
```

---

## Step 4: Compare Against Baseline 🔍

Detect changes and anomalies:

```bash
echo "=== Step 4: Comparison ==="

python3 <<'EOF'
import json
import sys

# Load baseline and current metrics
try:
    with open('/tmp/gh-aw/repo-memory/default/baseline.json') as f:
        baseline = json.load(f)
except FileNotFoundError:
    print("No baseline found - establishing initial baseline")
    baseline = {"metrics": {}}

with open('/tmp/current_metrics.json') as f:
    current = json.load(f)

# Initialize comparison results
comparison = {
    "timestamp": current["timestamp"],
    "changes_detected": [],
    "anomalies_detected": [],
    "severity": "normal"
}

# Compare response time
if "response_time_ms" in baseline.get("metrics", {}):
    baseline_rt = baseline["metrics"]["response_time_ms"]
    current_rt = current["response_time_ms"]
    baseline_p95 = baseline_rt.get("p95", baseline_rt.get("avg", 1000))
    
    if current_rt > baseline_p95 * 1.5:
        comparison["anomalies_detected"].append({
            "type": "response_time",
            "severity": "warning",
            "message": f"Response time ({current_rt}ms) is 50% above baseline P95 ({baseline_p95}ms)",
            "current": current_rt,
            "baseline": baseline_p95
        })
        comparison["severity"] = "warning"

# Compare data structure
if "data_structure" in baseline.get("metrics", {}):
    baseline_fields = set(baseline["metrics"]["data_structure"].get("expected_fields", []))
    current_fields = set(current["data_structure"]["fields"])
    
    # Detect missing fields
    missing_fields = baseline_fields - current_fields
    if missing_fields:
        comparison["changes_detected"].append({
            "type": "structure_change",
            "severity": "critical",
            "message": f"Missing expected fields: {', '.join(missing_fields)}",
            "missing": list(missing_fields)
        })
        comparison["severity"] = "critical"
    
    # Detect new fields
    new_fields = current_fields - baseline_fields
    if new_fields:
        comparison["changes_detected"].append({
            "type": "structure_change",
            "severity": "info",
            "message": f"New fields detected: {', '.join(new_fields)}",
            "new": list(new_fields)
        })

# Compare data values
if "data_values" in baseline.get("metrics", {}) and "data_values" in current:
    baseline_count = baseline["metrics"]["data_values"].get("total_count", 0)
    current_count = current["data_values"].get("total_count", 0)
    
    # Check if count is outside expected range
    count_min = baseline["metrics"]["data_values"].get("count_range_min", baseline_count * 0.9)
    count_max = baseline["metrics"]["data_values"].get("count_range_max", baseline_count * 1.1)
    
    if current_count < count_min or current_count > count_max:
        comparison["anomalies_detected"].append({
            "type": "data_count",
            "severity": "warning",
            "message": f"Data count ({current_count}) outside expected range ({count_min}-{count_max})",
            "current": current_count,
            "expected_range": [count_min, count_max]
        })
        if comparison["severity"] == "normal":
            comparison["severity"] = "warning"

# Save comparison results
with open('/tmp/comparison.json', 'w') as f:
    json.dump(comparison, f, indent=2)

print(f"Comparison complete - Severity: {comparison['severity']}")
print(f"Changes detected: {len(comparison['changes_detected'])}")
print(f"Anomalies detected: {len(comparison['anomalies_detected'])}")

# Exit with status code based on severity
if comparison["severity"] == "critical":
    sys.exit(2)
elif comparison["severity"] == "warning":
    sys.exit(1)
else:
    sys.exit(0)
EOF

COMPARISON_EXIT_CODE=$?
echo "Comparison exit code: $COMPARISON_EXIT_CODE"
```

---

## Step 5: Create Alerts for Changes 🚨

Generate alerts based on comparison results:

```bash
echo "=== Step 5: Generate Alerts ==="

# Load comparison results
COMPARISON=$(cat /tmp/comparison.json)
SEVERITY=$(echo "$COMPARISON" | jq -r '.severity')
CHANGES_COUNT=$(echo "$COMPARISON" | jq '.changes_detected | length')
ANOMALIES_COUNT=$(echo "$COMPARISON" | jq '.anomalies_detected | length')

echo "Severity: $SEVERITY"
echo "Changes: $CHANGES_COUNT"
echo "Anomalies: $ANOMALIES_COUNT"

if [ "$SEVERITY" = "critical" ] || [ "$SEVERITY" = "warning" ]; then
    echo "Creating alert issue..."
    
    # Format alert body
    ALERT_BODY=$(cat <<EOF
# 🚨 API Monitoring Alert - $SEVERITY

**API Endpoint**: \`$API_ENDPOINT\`
**Detection Time**: $(date -Iseconds)
**Severity**: $SEVERITY

## Summary

API monitoring detected $(($CHANGES_COUNT + $ANOMALIES_COUNT)) issue(s) requiring attention.

## Changes Detected ($CHANGES_COUNT)

$(echo "$COMPARISON" | jq -r '.changes_detected[] | "### \(.severity | ascii_upcase): \(.type)\n\n\(.message)\n"')

## Anomalies Detected ($ANOMALIES_COUNT)

$(echo "$COMPARISON" | jq -r '.anomalies_detected[] | "### \(.severity | ascii_upcase): \(.type)\n\n\(.message)\n\n- **Current Value**: \(.current)\n- **Baseline/Expected**: \(.baseline // .expected_range)\n"')

## Current Metrics

\`\`\`json
$(cat /tmp/current_metrics.json)
\`\`\`

## Baseline Metrics

\`\`\`json
$(cat /tmp/gh-aw/repo-memory/default/baseline.json)
\`\`\`

## Recommended Actions

1. Verify API endpoint is functioning correctly
2. Check for API version changes or breaking updates
3. Review recent deployments or configuration changes
4. Contact API provider if issues persist
5. Update baseline if changes are expected

## Historical Context

[Include trend data from history.jsonl if available]

---

*Alert generated by: ${{ github.workflow }}*
*Run ID: [${{ github.run_id }}](${{ github.server_url }}/${{ github.repository }}/actions/runs/${{ github.run_id }})*
EOF
)
    
    # Use create-issue safe-output
    # (In actual workflow, use the safe-output tool)
    echo "Alert would be created with body length: $(echo "$ALERT_BODY" | wc -c) bytes"
fi
```

---

## Step 6: Update Baseline 📝

Update baseline based on current data:

```bash
echo "=== Step 6: Update Baseline ==="

python3 <<'EOF'
import json
from datetime import datetime, timedelta
from statistics import mean, quantiles

# Load historical data
try:
    with open('/tmp/gh-aw/repo-memory/default/history.jsonl') as f:
        history = [json.loads(line) for line in f]
except FileNotFoundError:
    history = []

# Load current metrics
with open('/tmp/current_metrics.json') as f:
    current = json.load(f)

# Add current to history
history.append(current)

# Calculate rolling baseline (last 30 days)
thirty_days_ago = datetime.now() - timedelta(days=30)
recent_history = [
    h for h in history 
    if datetime.fromisoformat(h['timestamp']) > thirty_days_ago
]

if len(recent_history) >= 5:  # Need at least 5 data points
    # Calculate response time statistics
    response_times = [h['response_time_ms'] for h in recent_history]
    
    baseline = {
        "last_updated": current["timestamp"],
        "endpoint": "$API_ENDPOINT",
        "metrics": {
            "response_time_ms": {
                "avg": int(mean(response_times)),
                "min": min(response_times),
                "max": max(response_times),
                "p95": int(quantiles(response_times, n=20)[18]) if len(response_times) > 5 else max(response_times)
            },
            "data_structure": current["data_structure"],
            "data_values": {
                # Calculate data value ranges from recent history
                "total_count": current["data_values"].get("total_count", 0),
                "count_range_min": min(h["data_values"].get("total_count", 0) for h in recent_history if "data_values" in h),
                "count_range_max": max(h["data_values"].get("total_count", 0) for h in recent_history if "data_values" in h)
            }
        },
        "sample_size": len(recent_history)
    }
else:
    # Not enough data - use current as baseline
    baseline = {
        "last_updated": current["timestamp"],
        "endpoint": "$API_ENDPOINT",
        "metrics": {
            "response_time_ms": {
                "avg": current["response_time_ms"],
                "min": current["response_time_ms"],
                "max": current["response_time_ms"],
                "p95": current["response_time_ms"]
            },
            "data_structure": current["data_structure"],
            "data_values": current.get("data_values", {})
        },
        "sample_size": len(recent_history),
        "note": "Insufficient historical data - using current as baseline"
    }

# Save updated baseline
with open('/tmp/gh-aw/repo-memory/default/baseline.json', 'w') as f:
    json.dump(baseline, f, indent=2)

print(f"Baseline updated with {len(recent_history)} data points")

# Save current to history
with open('/tmp/gh-aw/repo-memory/default/history.jsonl', 'a') as f:
    f.write(json.dumps(current) + '\n')

print("History updated")
EOF

echo "✅ Baseline and history updated"
```

---

## Step 7: Cleanup Old Data 🗑️

Remove old history entries:

```bash
echo "=== Step 7: Cleanup ==="

python3 <<'EOF'
import json
from datetime import datetime, timedelta

# Cleanup history older than 90 days
history_file = '/tmp/gh-aw/repo-memory/default/history.jsonl'
cutoff = datetime.now() - timedelta(days=90)

recent = []
try:
    with open(history_file, 'r') as f:
        for line in f:
            record = json.loads(line)
            record_time = datetime.fromisoformat(record['timestamp'])
            if record_time > cutoff:
                recent.append(line)
    
    with open(history_file, 'w') as f:
        f.writelines(recent)
    
    print(f"Retained {len(recent)} records (90-day window)")
except FileNotFoundError:
    print("No history file to clean up")
EOF

echo "✅ Cleanup complete"
```

---

## Monitoring Patterns

### Pattern 1: Response Structure Validation
- Track expected fields and types
- Alert on missing or new fields
- Detect type changes

### Pattern 2: Performance Monitoring
- Track response times
- Set P95/P99 thresholds
- Alert on degradation

### Pattern 3: Data Quality Monitoring
- Track data counts and distributions
- Detect anomalous values
- Monitor data freshness

### Pattern 4: Availability Monitoring
- Track HTTP status codes
- Monitor uptime percentage
- Alert on failures

## Common Variations

### Variation 1: Multi-Endpoint Monitoring
Monitor multiple API endpoints, compare baselines per endpoint, aggregate health status, create unified dashboard.

### Variation 2: GraphQL API Monitoring
Query GraphQL APIs, track query performance, monitor schema changes, validate response shapes.

### Variation 3: Rate Limit Monitoring
Track API rate limit headers, alert on approaching limits, optimize request timing, implement backoff strategies.

## Success Criteria

- ✅ Successfully queries API
- ✅ Maintains accurate baseline
- ✅ Detects meaningful changes
- ✅ Minimal false positives
- ✅ Alerts are actionable
- ✅ Historical data is preserved

## Related Examples

This template is based on high-performing scenarios:
- BE-2: API performance monitoring (5.0 rating)
- External API integration patterns
- Baseline comparison strategies

---

**Note**: This is a template. Customize the API endpoint, metrics, and comparison logic to match your specific API monitoring needs.
