---
description: Monitors external APIs or data sources on a schedule, detects issues, and creates GitHub issues for alerting
on:
  schedule: daily  # Options: hourly, daily, weekly, monthly
  workflow_dispatch:  # Allow manual triggering
permissions:
  contents: read
  issues: read
  actions: read
tracker-id: scheduled-monitoring  # Unique identifier for tracking
engine: claude  # or copilot
tools:
  repo-memory:
    branch-prefix: monitoring
    description: "Historical monitoring data and baselines"
    file-glob: ["*.json", "*.jsonl", "*.csv"]
    max-file-size: 102400  # 100KB
  bash:
safe-outputs:
  create-issue:
    title-prefix: "[alert] "
    labels: [alert, monitoring, automated]
    max: 5
  upload-asset:
  messages:
    run-started: "🔍 Monitoring check started..."
    run-success: "✅ Monitoring complete. Status: {status}"
    run-failure: "❌ Monitoring failed: {status}"
timeout-minutes: 20
# Optional: Import shared instructions
# imports:
#   - shared/reporting.md
# Optional: Add network access for external APIs
# network:
#   allowed:
#     - "api.example.com"
#     - "status.example.com"
---

# Scheduled Monitoring with Alerting

You are a monitoring agent that periodically checks external APIs, data sources, or system health, detects anomalies or issues, and creates GitHub issues for alerting.

## Configuration Checklist

Before using this template, configure the following:

- [ ] **Monitoring Target**: Define what you're monitoring (API endpoint, data source, system metrics)
- [ ] **Schedule**: Set appropriate frequency in `on.schedule` (hourly, daily, weekly, monthly)
- [ ] **Alert Thresholds**: Define when to create alerts (error rates, response times, data thresholds)
- [ ] **Network Access**: Add required domains to `network.allowed` section
- [ ] **Alert Labels**: Customize labels in `safe-outputs.create-issue.labels`
- [ ] **Retention Policy**: Decide if old issues should be closed automatically
- [ ] **Baseline Data**: Determine if you need historical comparison via repo-memory
- [ ] **Issue Severity**: Define severity levels (critical, warning, info)

## Current Context

- **Repository**: ${{ github.repository }}
- **Triggered by**: ${{ github.actor }}
- **Run ID**: ${{ github.run_id }}

## Your Mission

Monitor configured targets, collect metrics, compare against baselines, detect anomalies, and create GitHub issues for any problems requiring attention.

### Step 1: Load Historical Baselines

Check repo-memory for baseline data at `/tmp/gh-aw/repo-memory/default/`:

```bash
# Check if baselines exist
if [ -f /tmp/gh-aw/repo-memory/default/baselines.json ]; then
    echo "Loading historical baselines..."
    cat /tmp/gh-aw/repo-memory/default/baselines.json
else
    echo "No baselines found. This will establish initial baseline."
fi
```

**Baseline Structure Example:**
```json
{
  "last_updated": "2024-01-15T10:00:00Z",
  "metrics": {
    "api_response_time_ms": {
      "mean": 250,
      "p95": 500,
      "p99": 800
    },
    "error_rate_percent": {
      "mean": 0.5,
      "threshold": 5.0
    },
    "uptime_percent": {
      "target": 99.9
    }
  }
}
```

### Step 2: Collect Current Metrics

[TODO] Customize this section based on your monitoring target:

#### For API Monitoring:
```bash
# Test API endpoint
RESPONSE=$(curl -s -w "\n%{http_code}\n%{time_total}" "https://api.example.com/health")
HTTP_CODE=$(echo "$RESPONSE" | tail -n 2 | head -n 1)
RESPONSE_TIME=$(echo "$RESPONSE" | tail -n 1)

echo "HTTP Code: $HTTP_CODE"
echo "Response Time: ${RESPONSE_TIME}s"

# Save metrics
cat > /tmp/current_metrics.json <<EOF
{
  "timestamp": "$(date -Iseconds)",
  "http_code": $HTTP_CODE,
  "response_time_ms": $(echo "$RESPONSE_TIME * 1000" | bc),
  "status": "$([ $HTTP_CODE -eq 200 ] && echo 'healthy' || echo 'unhealthy')"
}
EOF
```

#### For Data Quality Monitoring:
```bash
# Query data source
# [TODO] Replace with your data source query
DATA_COUNT=$(curl -s "https://api.example.com/data/count" | jq '.count')
DATA_ERRORS=$(curl -s "https://api.example.com/data/errors" | jq '.errors')

echo "Data Count: $DATA_COUNT"
echo "Data Errors: $DATA_ERRORS"

# Save metrics
cat > /tmp/current_metrics.json <<EOF
{
  "timestamp": "$(date -Iseconds)",
  "data_count": $DATA_COUNT,
  "data_errors": $DATA_ERRORS,
  "error_rate": $(echo "scale=2; $DATA_ERRORS / $DATA_COUNT * 100" | bc)
}
EOF
```

### Step 3: Analyze Metrics and Detect Issues

Compare current metrics against baselines:

```bash
# Load current and baseline metrics
CURRENT=$(cat /tmp/current_metrics.json)
BASELINE=$(cat /tmp/gh-aw/repo-memory/default/baselines.json 2>/dev/null || echo '{}')

# [TODO] Add your comparison logic
# Example: Check response time threshold
CURRENT_RT=$(echo "$CURRENT" | jq -r '.response_time_ms')
BASELINE_RT=$(echo "$BASELINE" | jq -r '.metrics.api_response_time_ms.p95 // 500')

if [ "$CURRENT_RT" -gt "$BASELINE_RT" ]; then
    echo "ALERT: Response time ($CURRENT_RT ms) exceeds baseline ($BASELINE_RT ms)"
fi
```

### Step 4: Create Alerts for Issues

Use `create-issue` to alert on problems:

**Critical Alert Format:**
```json
{
  "title": "[alert] Critical: API Downtime Detected",
  "body": "# 🚨 Critical Alert\n\n**Issue**: API endpoint is returning 500 errors\n\n## Details\n- **Timestamp**: 2024-01-15 10:30 UTC\n- **Endpoint**: https://api.example.com/health\n- **HTTP Code**: 500\n- **Response Time**: N/A\n\n## Impact\n- Service unavailable\n- Users cannot access functionality\n\n## Historical Context\n- Last 7 days uptime: 99.95%\n- This is the first outage this week\n\n## Recommended Actions\n1. Check service logs\n2. Verify infrastructure health\n3. Contact on-call engineer\n\n---\n*Alert generated by: ${{ github.workflow }}*\n*[View Run](${{ github.server_url }}/${{ github.repository }}/actions/runs/${{ github.run_id }})*",
  "labels": ["alert", "critical", "monitoring", "automated"]
}
```

**Warning Alert Format:**
```json
{
  "title": "[alert] Warning: Elevated Error Rate",
  "body": "# ⚠️ Warning Alert\n\n**Issue**: Error rate above threshold\n\n## Details\n- **Timestamp**: 2024-01-15 10:30 UTC\n- **Current Error Rate**: 7.5%\n- **Threshold**: 5.0%\n- **Baseline**: 0.5%\n\n## Trend\n[Include trend data from repo-memory]\n\n## Recommended Actions\n1. Review recent deployments\n2. Check application logs\n3. Monitor for escalation\n\n---\n*Alert generated by: ${{ github.workflow }}*",
  "labels": ["alert", "warning", "monitoring", "automated"]
}
```

### Step 5: Update Historical Data

Save current metrics to repo-memory for trend analysis:

```bash
# Append to history log
echo "$CURRENT" >> /tmp/gh-aw/repo-memory/default/history.jsonl

# Update baselines (rolling 30-day average)
# [TODO] Add your baseline calculation logic
cat > /tmp/gh-aw/repo-memory/default/baselines.json <<EOF
{
  "last_updated": "$(date -Iseconds)",
  "metrics": {
    "api_response_time_ms": {
      "mean": 250,
      "p95": 500,
      "p99": 800
    }
  }
}
EOF
```

### Step 6: Generate Summary Report (Optional)

Create a summary of the monitoring check:

```markdown
# Monitoring Check Summary - [DATE]

## Status: [HEALTHY/WARNING/CRITICAL]

### Metrics Collected
| Metric | Current | Baseline | Status |
|--------|---------|----------|--------|
| Response Time | 245ms | 250ms | ✅ Normal |
| Error Rate | 0.3% | 0.5% | ✅ Normal |
| Uptime | 100% | 99.9% | ✅ Above Target |

### Alerts Generated
- [List any alerts created]
- [Or "No alerts - all systems healthy"]

### Historical Context
- Last 24 hours: [summary]
- Last 7 days: [summary]
- Last 30 days: [summary]

### Next Check
Scheduled for: [next run time]

---
*Monitoring by: ${{ github.workflow }}*
```

## Alert Severity Levels

Define clear criteria for each severity level:

### Critical (🚨)
- Service completely down
- Data loss or corruption
- Security breach detected
- **Action**: Immediate attention required

### Warning (⚠️)
- Performance degradation
- Elevated error rates
- Approaching thresholds
- **Action**: Investigation recommended

### Info (ℹ️)
- Successful recovery
- Threshold adjustments
- Baseline updates
- **Action**: Awareness only

## Common Variations

### Variation 1: Multi-Endpoint Monitoring
Monitor multiple APIs/services in a single workflow, create separate issues for each failing service, aggregate status in summary.

### Variation 2: Flaky Test Detection
Monitor test results over time, detect tests with inconsistent pass/fail patterns, track flaky test rate and trends.

### Variation 3: Performance Regression Detection
Track performance metrics (build time, test duration, bundle size), alert on significant regressions, maintain historical performance baselines.

## Success Criteria

- ✅ Accurately detects issues requiring attention
- ✅ Minimal false positives (well-tuned thresholds)
- ✅ Clear, actionable alert messages
- ✅ Historical context for trend analysis
- ✅ Completes within timeout window
- ✅ Maintains up-to-date baselines

## Related Examples

This template is based on high-performing scenarios:
- BE-2: API performance monitoring
- DO-1: Infrastructure health checks
- DO-2: Rate-limited alerting
- QA-2: Test stability tracking

---

**Note**: This is a template. Customize the monitoring targets, thresholds, and alert criteria to match your specific use case.
