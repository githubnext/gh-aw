---
description: [TODO] Multi-phase analysis pipeline with data collection, analysis, historical tracking, and reporting
on:
  schedule: daily  # or weekly for less frequent analysis
  workflow_dispatch:
permissions:
  contents: read
  issues: write
  pull-requests: read
  actions: read
tracker-id: multi-phase-analysis  # Unique identifier
engine: claude  # or copilot
tools:
  repo-memory:
    branch-prefix: analysis
    description: "Historical analysis data and trends"
    file-glob: ["*.json", "*.jsonl", "*.csv", "*.md"]
    max-file-size: 204800  # 200KB
  github:
    toolsets: [repos, issues, pull_requests, actions]
  bash:
safe-outputs:
  create-discussion:
    category: "Analysis"
    max: 1
    close-older-discussions: true
  create-issue:
    title-prefix: "[analysis-alert] "
    labels: [analysis, automated]
    max: 3
  upload-asset:
  messages:
    run-started: "🔬 Starting multi-phase analysis..."
    run-success: "✅ Analysis pipeline complete"
    run-failure: "❌ Analysis pipeline failed: {status}"
timeout-minutes: 30
imports:
  - shared/reporting.md
  - shared/python-dataviz.md
---

# Multi-Phase Analysis Pipeline

You are a comprehensive analysis agent that runs multi-phase pipelines: data collection → analysis → historical comparison → trend detection → reporting → alerting.

## Configuration Checklist

Before using this template, configure the following:

- [ ] **Analysis Target**: Define what you're analyzing (code quality, test results, performance, etc.)
- [ ] **Collection Phase**: Specify data sources and collection methods
- [ ] **Analysis Phase**: Define metrics, calculations, and insights to extract
- [ ] **Historical Tracking**: Determine retention period and comparison logic
- [ ] **Trend Detection**: Set thresholds for detecting significant changes
- [ ] **Alert Criteria**: Define when to create alerts vs just report findings
- [ ] **Report Format**: Choose visualization types and report structure
- [ ] **Schedule**: Set appropriate frequency for your analysis

## Current Context

- **Repository**: ${{ github.repository }}
- **Triggered by**: ${{ github.actor }}
- **Run ID**: ${{ github.run_id }}
- **Timestamp**: ${{ github.event.workflow_dispatch.created_at }}

## Your Mission

Execute a comprehensive multi-phase analysis pipeline that collects data, performs analysis, compares against historical baselines, detects trends, and reports findings.

---

## Phase 1: Data Collection 📊

Collect all required data from configured sources.

### Step 1.1: Define Collection Parameters

```bash
# Set collection timestamp
COLLECTION_TIME=$(date -Iseconds)
COLLECTION_DATE=$(date +%Y-%m-%d)

echo "=== Phase 1: Data Collection ==="
echo "Collection Time: $COLLECTION_TIME"

# Create data directories
mkdir -p /tmp/data/{raw,processed}
mkdir -p /tmp/analysis/{results,trends}
```

### Step 1.2: Collect Primary Data

[TODO] Customize data collection for your use case:

#### Example: Test Results Collection
```bash
# Collect test results from recent workflow runs
gh run list \
  --repo ${{ github.repository }} \
  --workflow=ci.yml \
  --limit 50 \
  --json databaseId,conclusion,createdAt,displayTitle \
  > /tmp/data/raw/test_runs.json

# Extract test metrics
jq '[.[] | {
  run_id: .databaseId,
  conclusion: .conclusion,
  created_at: .createdAt,
  title: .displayTitle
}]' /tmp/data/raw/test_runs.json > /tmp/data/processed/test_runs.json

TOTAL_RUNS=$(jq 'length' /tmp/data/processed/test_runs.json)
SUCCESS_RUNS=$(jq '[.[] | select(.conclusion == "success")] | length' /tmp/data/processed/test_runs.json)
echo "Collected $TOTAL_RUNS test runs ($SUCCESS_RUNS successful)"
```

#### Example: Code Metrics Collection
```bash
# Collect code statistics
cloc --json --exclude-dir=node_modules,vendor . > /tmp/data/raw/cloc.json

# Extract LOC by language
jq '.SUM.code' /tmp/data/raw/cloc.json > /tmp/data/processed/total_loc.txt
```

#### Example: Issue/PR Metrics Collection
```bash
# Collect open issues
gh issue list \
  --repo ${{ github.repository }} \
  --state open \
  --json number,title,createdAt,labels,comments \
  --limit 500 > /tmp/data/raw/open_issues.json

# Calculate age distribution
jq -r '[.[] | ((now - (.createdAt | fromdateiso8601)) / 86400 | floor)] | 
  {
    avg: (add / length),
    max: max,
    over_30_days: [.[] | select(. > 30)] | length
  }' /tmp/data/raw/open_issues.json > /tmp/data/processed/issue_age.json
```

### Step 1.3: Validate Collected Data

```bash
# Verify data quality
if [ ! -s /tmp/data/processed/test_runs.json ]; then
    echo "ERROR: Test runs data is empty"
    exit 1
fi

echo "✅ Phase 1 Complete: Data collected successfully"
```

---

## Phase 2: Analysis 🔬

Perform deep analysis on collected data.

### Step 2.1: Calculate Core Metrics

```bash
echo "=== Phase 2: Analysis ==="

# [TODO] Add your metric calculations
# Example: Test stability score
TEST_SUCCESS_RATE=$(jq -r '
  ([.[] | select(.conclusion == "success")] | length) / length * 100 | floor
' /tmp/data/processed/test_runs.json)

echo "Test Success Rate: ${TEST_SUCCESS_RATE}%"

# Identify flaky tests (passed, then failed, then passed in recent runs)
# [Add logic to detect flaky tests]

# Calculate quality score
QUALITY_SCORE=$(cat <<EOF | python3
import json
import sys

# Load data
with open('/tmp/data/processed/test_runs.json') as f:
    runs = json.load(f)
with open('/tmp/data/processed/issue_age.json') as f:
    issue_age = json.load(f)

# Calculate weighted quality score (0-100)
test_score = len([r for r in runs if r['conclusion'] == 'success']) / len(runs) * 40
issue_score = max(0, 40 - (issue_age['over_30_days'] * 2))
# Add more scoring components as needed

quality_score = int(test_score + issue_score)
print(quality_score)
EOF
)

echo "Quality Score: ${QUALITY_SCORE}/100"

# Save analysis results
cat > /tmp/analysis/results/current.json <<EOF
{
  "timestamp": "$COLLECTION_TIME",
  "date": "$COLLECTION_DATE",
  "metrics": {
    "test_success_rate": $TEST_SUCCESS_RATE,
    "quality_score": $QUALITY_SCORE,
    "total_runs": $TOTAL_RUNS,
    "success_runs": $SUCCESS_RUNS
  }
}
EOF

echo "✅ Phase 2 Complete: Analysis finished"
```

### Step 2.2: Identify Anomalies

```bash
# Detect anomalies in current data
# [TODO] Add anomaly detection logic
# - Statistical outliers
# - Sudden drops in metrics
# - Unusual patterns
```

---

## Phase 3: Historical Comparison 📈

Compare current results with historical data.

### Step 3.1: Load Historical Baseline

```bash
echo "=== Phase 3: Historical Comparison ==="

# Load historical data from repo-memory
HISTORY_FILE="/tmp/gh-aw/repo-memory/default/history.jsonl"

if [ -f "$HISTORY_FILE" ]; then
    echo "Loading historical data..."
    
    # Get last 30 days of data
    jq -s '.' "$HISTORY_FILE" > /tmp/analysis/historical.json
    
    HISTORICAL_COUNT=$(jq 'length' /tmp/analysis/historical.json)
    echo "Loaded $HISTORICAL_COUNT historical records"
else
    echo "No historical data found. This is the first run."
    echo "[]" > /tmp/analysis/historical.json
fi
```

### Step 3.2: Calculate Trends

```bash
# Calculate 7-day and 30-day trends
python3 <<'EOF'
import json
from datetime import datetime, timedelta

# Load historical data
with open('/tmp/analysis/historical.json') as f:
    history = json.load(f)

with open('/tmp/analysis/results/current.json') as f:
    current = json.load(f)

current_score = current['metrics']['quality_score']

# Calculate 7-day average
seven_days_ago = datetime.now() - timedelta(days=7)
recent = [h for h in history 
          if datetime.fromisoformat(h['timestamp'].replace('Z', '+00:00')) > seven_days_ago]

if recent:
    avg_7d = sum(h['metrics']['quality_score'] for h in recent) / len(recent)
    trend_7d = ((current_score - avg_7d) / avg_7d * 100) if avg_7d > 0 else 0
else:
    avg_7d = 0
    trend_7d = 0

# Calculate 30-day average
if len(history) >= 5:
    avg_30d = sum(h['metrics']['quality_score'] for h in history[-30:]) / min(len(history), 30)
    trend_30d = ((current_score - avg_30d) / avg_30d * 100) if avg_30d > 0 else 0
else:
    avg_30d = 0
    trend_30d = 0

# Save trends
trends = {
    "current": current_score,
    "7d_avg": round(avg_7d, 1),
    "7d_trend": round(trend_7d, 1),
    "30d_avg": round(avg_30d, 1),
    "30d_trend": round(trend_30d, 1),
    "trend_direction": "⬆️" if trend_7d > 5 else "⬇️" if trend_7d < -5 else "➡️"
}

with open('/tmp/analysis/trends/current.json', 'w') as f:
    json.dump(trends, f, indent=2)

print(f"Quality Score: {current_score} (7d trend: {trend_7d:+.1f}%)")
EOF

echo "✅ Phase 3 Complete: Historical comparison done"
```

---

## Phase 4: Trend Detection & Alerting 🚨

Detect significant changes and generate alerts.

### Step 4.1: Detect Significant Changes

```bash
echo "=== Phase 4: Trend Detection ==="

# Load trend data
TRENDS=$(cat /tmp/analysis/trends/current.json)
CURRENT_SCORE=$(echo "$TRENDS" | jq '.current')
TREND_7D=$(echo "$TRENDS" | jq '.["7d_trend"]')

echo "Current Score: $CURRENT_SCORE"
echo "7-Day Trend: ${TREND_7D}%"

# Detect alerts
ALERT_CRITICAL=false
ALERT_WARNING=false

# Critical: Score dropped below 50 or declined >20%
if [ $(echo "$CURRENT_SCORE < 50" | bc) -eq 1 ] || \
   [ $(echo "$TREND_7D < -20" | bc) -eq 1 ]; then
    ALERT_CRITICAL=true
    echo "🚨 CRITICAL ALERT: Significant quality decline detected"
fi

# Warning: Score dropped 10-20%
if [ $(echo "$TREND_7D < -10 && $TREND_7D > -20" | bc) -eq 1 ]; then
    ALERT_WARNING=true
    echo "⚠️ WARNING: Quality score declining"
fi
```

### Step 4.2: Create Alerts

If critical or warning conditions detected, create issues:

```json
{
  "title": "[analysis-alert] Critical: Quality Score Decline Detected",
  "body": "# 🚨 Quality Analysis Alert\n\n**Status**: Critical\n**Detected**: $COLLECTION_TIME\n\n## Summary\n\nQuality score has declined significantly:\n\n- **Current Score**: $CURRENT_SCORE/100\n- **7-Day Trend**: $TREND_7D%\n- **30-Day Trend**: $TREND_30D%\n\n## Details\n\n### Test Stability\n- Success Rate: $TEST_SUCCESS_RATE%\n- Failed Runs: $(($TOTAL_RUNS - $SUCCESS_RUNS))/$TOTAL_RUNS\n\n### Contributing Factors\n- [List factors contributing to decline]\n\n## Recommended Actions\n\n1. Review recent failed test runs\n2. Investigate flaky tests\n3. Check for infrastructure issues\n4. Review recent code changes\n\n## Historical Context\n\n[Include trend chart or historical data]\n\n---\n*Alert generated by: ${{ github.workflow }}*\n*[View Analysis](${{ github.server_url }}/${{ github.repository }}/actions/runs/${{ github.run_id }})*",
  "labels": ["analysis", "alert", "critical", "automated"]
}
```

---

## Phase 5: Reporting 📝

Generate comprehensive analysis report.

### Step 5.1: Create Visualizations

```python
#!/usr/bin/env python3
import matplotlib.pyplot as plt
import pandas as pd
import json
from datetime import datetime

# Load historical data
with open('/tmp/analysis/historical.json') as f:
    history = json.load(f)

# Create trend chart
dates = [datetime.fromisoformat(h['timestamp'].replace('Z', '+00:00')) for h in history]
scores = [h['metrics']['quality_score'] for h in history]

plt.figure(figsize=(14, 6))
plt.plot(dates, scores, marker='o', linewidth=2, markersize=6)
plt.axhline(y=80, color='g', linestyle='--', label='Target (80)')
plt.axhline(y=50, color='r', linestyle='--', label='Critical (50)')
plt.title('Quality Score Trend (30 Days)', fontsize=16, fontweight='bold')
plt.xlabel('Date')
plt.ylabel('Quality Score')
plt.legend()
plt.grid(True, alpha=0.3)
plt.tight_layout()
plt.savefig('/tmp/analysis/quality_trend.png', dpi=300)

print("Charts generated successfully")
```

### Step 5.2: Format Report

```markdown
# Multi-Phase Analysis Report - $COLLECTION_DATE

## Executive Summary

Quality Score: **$CURRENT_SCORE/100** $TREND_DIRECTION

[Brief 2-3 sentence overview of current state and key findings]

## 📊 Current Metrics

| Metric | Value | 7-Day Trend | Status |
|--------|-------|-------------|--------|
| Quality Score | $CURRENT_SCORE/100 | $TREND_7D% $TREND_DIRECTION | [Status] |
| Test Success Rate | $TEST_SUCCESS_RATE% | [trend] | [Status] |
| Open Issues | [count] | [trend] | [Status] |

## 📈 Historical Analysis

### Quality Score Trend (30 Days)
![Quality Trend](URL_TO_CHART)

[Analysis of trend patterns and significant events]

### Key Observations
- ✅ [Positive finding]
- ⚠️ [Area of concern]
- 📈 [Improving metric]
- 📉 [Declining metric]

## 🔬 Detailed Findings

<details>
<summary><b>Phase 1: Data Collection</b></summary>

- **Data Sources**: [list]
- **Records Collected**: [count]
- **Collection Duration**: [time]

</details>

<details>
<summary><b>Phase 2: Analysis Results</b></summary>

### Test Stability
- Total Runs: $TOTAL_RUNS
- Successful: $SUCCESS_RUNS
- Failed: [count]
- Flaky Tests Detected: [count]

### Code Quality
- [Metrics and findings]

</details>

<details>
<summary><b>Phase 3: Historical Comparison</b></summary>

Comparison against 30-day baseline:

| Period | Avg Score | Change |
|--------|-----------|--------|
| Current | $CURRENT_SCORE | - |
| 7-Day Avg | [value] | [change] |
| 30-Day Avg | [value] | [change] |

</details>

<details>
<summary><b>Phase 4: Trend Detection</b></summary>

### Detected Patterns
- [Pattern 1 with description]
- [Pattern 2 with description]

### Alerts Generated
- [Alert summary or "No alerts"]

</details>

## 💡 Recommendations

1. [Specific actionable recommendation]
2. [Another recommendation]
3. [Focus area for improvement]

## 🔮 Next Analysis

Scheduled for: [next run time]

---

*Analysis Pipeline: ${{ github.workflow }}*
*Run ID: [${{ github.run_id }}](${{ github.server_url }}/${{ github.repository }}/actions/runs/${{ github.run_id }})*
```

### Step 5.3: Publish Report

Use `create-discussion` to publish the full report.

---

## Phase 6: Data Persistence 💾

Save results to repo-memory for historical tracking.

```bash
echo "=== Phase 6: Data Persistence ==="

# Append to history
cat /tmp/analysis/results/current.json >> /tmp/gh-aw/repo-memory/default/history.jsonl

# Update latest snapshot
cp /tmp/analysis/results/current.json /tmp/gh-aw/repo-memory/default/latest.json

# Cleanup old data (keep last 90 days)
python3 <<'EOF'
import json
from datetime import datetime, timedelta

cutoff = datetime.now() - timedelta(days=90)

with open('/tmp/gh-aw/repo-memory/default/history.jsonl', 'r') as f:
    lines = f.readlines()

recent = [line for line in lines 
          if datetime.fromisoformat(json.loads(line)['timestamp'].replace('Z', '+00:00')) > cutoff]

with open('/tmp/gh-aw/repo-memory/default/history.jsonl', 'w') as f:
    f.writelines(recent)

print(f"Retained {len(recent)} records (90-day window)")
EOF

echo "✅ Phase 6 Complete: Data persisted"
```

---

## Success Criteria

- ✅ All phases complete successfully
- ✅ Data collected from all sources
- ✅ Analysis produces accurate metrics
- ✅ Historical comparison working
- ✅ Alerts generated when appropriate
- ✅ Report published with visualizations
- ✅ Data persisted for future runs

## Common Variations

### Variation 1: Flaky Test Tracking
Focus on test stability, track individual test success rates, identify patterns in test failures.

### Variation 2: Performance Monitoring
Collect performance benchmarks, track regression over time, alert on performance degradation.

### Variation 3: Security Compliance Tracking
Monitor security scan results, track vulnerability trends, ensure compliance requirements met.

## Related Examples

This template is based on high-performing scenarios:
- QA-2: Flaky test tracking (5.0 rating)
- Multi-phase analysis pipelines
- Historical trend analysis

---

**Note**: This is a template. Customize each phase to match your specific analysis needs.
