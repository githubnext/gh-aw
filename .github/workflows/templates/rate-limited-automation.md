---
description: [TODO] Detects items requiring action, prioritizes them, and creates limited PRs/issues to avoid overwhelming the team
on:
  schedule: daily  # Run daily to check for new items
  workflow_dispatch:
permissions:
  contents: read
  issues: write
  pull-requests: write  # If creating PRs
  actions: read
engine: claude  # or copilot
tools:
  github:
    toolsets: [repos, issues, pull_requests]
  repo-memory:
    branch-prefix: rate-limited
    description: "Track what's been processed to avoid duplicates"
    file-glob: ["*.json", "*.jsonl"]
    max-file-size: 51200  # 50KB
  bash:
safe-outputs:
  create-issue:
    title-prefix: "[auto] "
    labels: [automated, needs-review]
    max: 3  # Limit to 3 issues per run
  create-pull-request:
    max: 2  # Limit to 2 PRs per run (if applicable)
  messages:
    run-started: "🔍 Scanning for items requiring action..."
    run-success: "✅ Processed {count} items (rate-limited)"
    run-failure: "❌ Rate-limited automation failed: {status}"
timeout-minutes: 20
---

# Rate-Limited Automation

You are a rate-limited automation agent that detects items requiring action, prioritizes them intelligently, and creates a limited number of issues or PRs per run to avoid overwhelming the team.

## Configuration Checklist

Before using this template, configure the following:

- [ ] **Detection Criteria**: Define what items need automated action
- [ ] **Prioritization Logic**: Set rules for ranking items by importance
- [ ] **Rate Limits**: Configure max issues/PRs per run in safe-outputs
- [ ] **Deduplication**: Decide how to track already-processed items
- [ ] **Schedule Frequency**: Set appropriate run frequency (daily recommended)
- [ ] **Action Type**: Choose between creating issues, PRs, or both
- [ ] **Quality Thresholds**: Set minimum confidence/quality scores for automation
- [ ] **Human Review**: Determine which items need human approval

## Current Context

- **Repository**: ${{ github.repository }}
- **Triggered by**: ${{ github.actor }}
- **Run ID**: ${{ github.run_id }}
- **Timestamp**: $(date -Iseconds)

## Your Mission

Scan for items requiring automated action, prioritize them based on impact and urgency, create a limited number of issues/PRs for highest-priority items, and track processed items to avoid duplicates.

---

## Step 1: Define Detection Criteria 🔍

[TODO] Customize detection logic for your use case:

### Example: Security Vulnerability Detection
```bash
echo "=== Step 1: Detection ==="

# Scan dependencies for known vulnerabilities
npm audit --json > /tmp/audit.json 2>/dev/null || true

# Extract high/critical vulnerabilities
VULNERABILITIES=$(jq -r '.vulnerabilities | to_entries[] | 
  select(.value.severity == "high" or .value.severity == "critical") | 
  {
    name: .key,
    severity: .value.severity,
    via: .value.via[0].title,
    range: .value.range
  }' /tmp/audit.json)

VULN_COUNT=$(echo "$VULNERABILITIES" | jq -s 'length')
echo "Found $VULN_COUNT high/critical vulnerabilities"
```

### Example: Outdated Dependencies Detection
```bash
# Check for outdated dependencies
npm outdated --json > /tmp/outdated.json 2>/dev/null || true

# Filter for major version updates
OUTDATED=$(jq -r 'to_entries[] | 
  select(.value.wanted != .value.latest) |
  select(.value.latest | split(".")[0] | tonumber > (.value.current | split(".")[0] | tonumber)) |
  {
    package: .key,
    current: .value.current,
    latest: .value.latest,
    age_days: .value.time
  }' /tmp/outdated.json)

OUTDATED_COUNT=$(echo "$OUTDATED" | jq -s 'length')
echo "Found $OUTDATED_COUNT packages with major updates available"
```

### Example: Code Pattern Detection
```bash
# Scan for deprecated API usage
grep -r "deprecatedFunction" --include="*.js" --include="*.ts" . | \
  grep -v node_modules | \
  awk '{print $1}' | \
  sort -u > /tmp/deprecated_usage.txt

DEPRECATED_COUNT=$(wc -l < /tmp/deprecated_usage.txt)
echo "Found $DEPRECATED_COUNT files using deprecated APIs"
```

---

## Step 2: Load Processing History 📋

Check repo-memory to avoid creating duplicate issues:

```bash
echo "=== Step 2: Load History ==="

HISTORY_FILE="/tmp/gh-aw/repo-memory/default/processed.jsonl"
PROCESSED_ITEMS=()

if [ -f "$HISTORY_FILE" ]; then
    echo "Loading processing history..."
    
    # Load list of already-processed items
    while IFS= read -r line; do
        ITEM_ID=$(echo "$line" | jq -r '.item_id')
        PROCESSED_ITEMS+=("$ITEM_ID")
    done < "$HISTORY_FILE"
    
    echo "Loaded ${#PROCESSED_ITEMS[@]} previously processed items"
else
    echo "No processing history found"
fi
```

---

## Step 3: Prioritize Items 🎯

Score and rank items by importance:

```bash
echo "=== Step 3: Prioritization ==="

# [TODO] Customize scoring logic for your use case
python3 <<'EOF'
import json
import sys

# Load detected items
# Example: vulnerabilities
with open('/tmp/audit.json') as f:
    audit = json.load(f)

vulns = []
for name, vuln in audit.get('vulnerabilities', {}).items():
    # Calculate priority score (0-100)
    score = 0
    
    # Severity weight (40 points)
    if vuln['severity'] == 'critical':
        score += 40
    elif vuln['severity'] == 'high':
        score += 30
    elif vuln['severity'] == 'moderate':
        score += 15
    
    # Exploitability (30 points)
    if any('exploited' in str(v).lower() for v in vuln.get('via', [])):
        score += 30
    elif any('proof-of-concept' in str(v).lower() for v in vuln.get('via', [])):
        score += 20
    
    # Impact on direct dependencies (20 points)
    if vuln.get('isDirect', False):
        score += 20
    
    # Recent discovery (10 points)
    # Check if vulnerability is less than 30 days old
    score += 10  # Simplified - add date logic if needed
    
    vulns.append({
        'name': name,
        'severity': vuln['severity'],
        'score': score,
        'description': vuln.get('via', [{}])[0].get('title', 'Unknown'),
        'range': vuln.get('range', 'unknown')
    })

# Sort by priority score (highest first)
vulns.sort(key=lambda x: x['score'], reverse=True)

# Save prioritized list
with open('/tmp/prioritized.json', 'w') as f:
    json.dump(vulns, f, indent=2)

print(f"Prioritized {len(vulns)} items")
for i, v in enumerate(vulns[:5], 1):
    print(f"  {i}. {v['name']} (score: {v['score']}, severity: {v['severity']})")
EOF
```

---

## Step 4: Filter and Deduplicate 🔄

Remove already-processed items and apply filters:

```bash
echo "=== Step 4: Filtering ==="

# Load prioritized items
PRIORITIZED=$(cat /tmp/prioritized.json)

# Filter out already-processed items
FILTERED=$(echo "$PRIORITIZED" | jq --argjson processed "$(printf '%s\n' "${PROCESSED_ITEMS[@]}" | jq -R . | jq -s .)" '
  [.[] | select(.name as $name | $processed | index($name) | not)]
')

FILTERED_COUNT=$(echo "$FILTERED" | jq 'length')
echo "After deduplication: $FILTERED_COUNT items remaining"

# Apply quality threshold (score >= 50)
HIGH_PRIORITY=$(echo "$FILTERED" | jq '[.[] | select(.score >= 50)]')
HIGH_PRIORITY_COUNT=$(echo "$HIGH_PRIORITY" | jq 'length')

echo "High-priority items (score >= 50): $HIGH_PRIORITY_COUNT"

# Save filtered list
echo "$HIGH_PRIORITY" > /tmp/actionable.json
```

---

## Step 5: Create Limited Actions 📝

Create issues/PRs for top-priority items, respecting rate limits:

```bash
echo "=== Step 5: Creating Actions ==="

# Get rate limit from safe-outputs (default: 3)
RATE_LIMIT=3

# Load actionable items
ACTIONABLE=$(cat /tmp/actionable.json)
TOTAL_ACTIONABLE=$(echo "$ACTIONABLE" | jq 'length')

echo "Actionable items: $TOTAL_ACTIONABLE"
echo "Rate limit: $RATE_LIMIT items per run"

# Select top N items (up to rate limit)
TO_PROCESS=$(echo "$ACTIONABLE" | jq --arg limit "$RATE_LIMIT" '[.[:($limit | tonumber)]]')
PROCESS_COUNT=$(echo "$TO_PROCESS" | jq 'length')

echo "Processing $PROCESS_COUNT items this run"
```

### Create Issues for Each Item

```bash
# Iterate through items to process
echo "$TO_PROCESS" | jq -c '.[]' | while IFS= read -r item; do
    NAME=$(echo "$item" | jq -r '.name')
    SEVERITY=$(echo "$item" | jq -r '.severity')
    SCORE=$(echo "$item" | jq -r '.score')
    DESCRIPTION=$(echo "$item" | jq -r '.description')
    
    echo "Creating issue for: $NAME (score: $SCORE)"
    
    # Use create-issue safe-output
    # Format issue body
    ISSUE_BODY=$(cat <<EOF
# 🔒 Security: $SEVERITY Vulnerability Detected

**Package**: \`$NAME\`
**Severity**: $SEVERITY
**Priority Score**: $SCORE/100

## Description

$DESCRIPTION

## Details

- **Current Range**: $(echo "$item" | jq -r '.range')
- **Detection Date**: $(date +%Y-%m-%d)
- **Auto-generated**: Yes

## Recommended Action

1. Review the vulnerability details
2. Update the package to a patched version
3. Test for any breaking changes
4. Deploy the fix

## Priority Justification

This vulnerability scored $SCORE/100 based on:
- Severity level: $SEVERITY
- Exploitability: [assessed]
- Direct dependency impact: [assessed]
- Age: [recent discovery]

## Resources

- [CVE Details](#)  # Add actual links
- [Package Security Advisory](#)

---

*Auto-generated by: ${{ github.workflow }}*
*Run ID: [${{ github.run_id }}](${{ github.server_url }}/${{ github.repository }}/actions/runs/${{ github.run_id }})*
*This is item $(expr $PROCESS_COUNT) of $TOTAL_ACTIONABLE high-priority items*
EOF
)
    
    # Create issue via safe-output
    # (In actual workflow, use the safe-output tool)
    
    # Record processed item
    echo "{\"item_id\": \"$NAME\", \"processed_at\": \"$(date -Iseconds)\", \"score\": $SCORE}" >> /tmp/processed_this_run.jsonl
    
    # Add delay to avoid rate limiting
    sleep 2
done
```

---

## Step 6: Update Processing History 💾

Save processed items to avoid duplicates in future runs:

```bash
echo "=== Step 6: Updating History ==="

# Append newly processed items to history
if [ -f /tmp/processed_this_run.jsonl ]; then
    cat /tmp/processed_this_run.jsonl >> /tmp/gh-aw/repo-memory/default/processed.jsonl
    
    PROCESSED_COUNT=$(wc -l < /tmp/processed_this_run.jsonl)
    echo "Updated history with $PROCESSED_COUNT new items"
fi

# Cleanup old entries (keep last 90 days)
python3 <<'EOF'
import json
from datetime import datetime, timedelta

history_file = '/tmp/gh-aw/repo-memory/default/processed.jsonl'
cutoff = datetime.now() - timedelta(days=90)

recent = []
try:
    with open(history_file, 'r') as f:
        for line in f:
            record = json.loads(line)
            processed_at = datetime.fromisoformat(record['processed_at'].replace('Z', '+00:00'))
            if processed_at > cutoff:
                recent.append(line)
    
    with open(history_file, 'w') as f:
        f.writelines(recent)
    
    print(f"Retained {len(recent)} records (90-day window)")
except FileNotFoundError:
    print("No history file to clean up")
EOF
```

---

## Step 7: Generate Summary Report 📊

Create summary of what was processed:

```markdown
# Rate-Limited Automation Report

**Run Date**: $(date +%Y-%m-%d)

## Summary

- **Items Detected**: $TOTAL_ACTIONABLE
- **Items Processed**: $PROCESS_COUNT
- **Items Remaining**: $(($TOTAL_ACTIONABLE - $PROCESS_COUNT))
- **Rate Limit**: $RATE_LIMIT items per run

## Processed This Run

$(echo "$TO_PROCESS" | jq -r '.[] | "- **\(.name)** (severity: \(.severity), score: \(.score))"')

## Remaining High-Priority Items

$(echo "$ACTIONABLE" | jq --arg limit "$RATE_LIMIT" -r '.[($limit | tonumber):] | .[] | "- **\(.name)** (severity: \(.severity), score: \(.score))"')

## Next Run

Estimated items to process in next run: $RATE_LIMIT

Remaining items will be processed over the next $(echo "scale=0; ($TOTAL_ACTIONABLE - $PROCESS_COUNT + $RATE_LIMIT - 1) / $RATE_LIMIT" | bc) runs.

---

*Automation by: ${{ github.workflow }}*
```

---

## Rate Limiting Strategy

### Why Rate Limit?

- **Prevents overwhelm**: Teams can handle a few items per day
- **Maintains quality**: Each item gets proper attention
- **Reduces noise**: Avoid alert fatigue
- **Spreads work**: Distributes work over time

### Configuring Rate Limits

```yaml
safe-outputs:
  create-issue:
    max: 3  # Max 3 issues per run
    
  create-pull-request:
    max: 2  # Max 2 PRs per run
```

### Prioritization Best Practices

1. **Severity-based**: Critical > High > Medium > Low
2. **Impact-based**: Security > Functionality > Performance > Style
3. **Recency-based**: Recent issues get higher priority
4. **Exploitability**: Known exploits get highest priority
5. **Complexity**: Consider fix effort in prioritization

## Common Variations

### Variation 1: Dependency Update Automation
Detect outdated dependencies, prioritize by security and compatibility, create PRs with rate limiting, include automated tests.

### Variation 2: Code Modernization
Scan for deprecated patterns, prioritize by frequency and risk, create batched refactoring PRs, maintain compatibility.

### Variation 3: Issue Triage Automation
Detect stale issues, prioritize by activity and labels, create meta-issues for cleanup, respect team capacity.

## Success Criteria

- ✅ Accurately detects items requiring action
- ✅ Prioritization is logical and consistent
- ✅ Rate limits prevent overwhelm
- ✅ Deduplication prevents duplicate issues
- ✅ Processing history is maintained
- ✅ Human review is facilitated
- ✅ Completes within timeout

## Related Examples

This template is based on high-performing scenarios:
- DO-2: Rate-limited alerting
- Security scanner with prioritization (5.0 rating)
- Automated dependency updates

---

**Note**: This is a template. Customize the detection, prioritization, and rate limits to match your team's capacity and needs.
