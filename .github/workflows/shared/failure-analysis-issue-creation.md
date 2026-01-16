---
tools:
  cache-memory: true
  bash:
    - "jq *"
    - "grep *"
    - "awk *"
    - "sed *"
    - "md5sum *"
---

## Failure Analysis and Issue Creation Framework

This component provides standardized patterns for analyzing failures, clustering similar issues, and creating high-quality diagnostic GitHub issues.

### Issue Template Structure

All failure analysis issues should follow this structure:

```markdown
## Summary

[Brief 1-2 sentence description of the failure]

## Failure Details

- **Source**: [Workflow name, tool, or component]
- **Severity**: [Critical/High/Medium/Low]
- **Frequency**: [How often this occurs - X times per day/week]
- **First Seen**: [Date or "New in this run"]
- **Last Seen**: [Date of most recent occurrence]
- **Occurrences (Last 30 Days)**: [Count from historical data]

## Root Cause Analysis

[Detailed analysis of what went wrong]

**Error Pattern**: `[Regex or signature of the error]`

**Why It Fails**:
- [Reason 1]
- [Reason 2]

**Contributing Factors**:
- [Factor 1]
- [Factor 2]

## Affected Components

| Component | Impact | Status |
|-----------|--------|--------|
| [Component 1] | [Description of impact] | [Broken/Degraded/At Risk] |
| [Component 2] | [Description of impact] | [Status] |

## Reproduction Steps

1. [Step 1 to reproduce the issue]
2. [Step 2]
3. [Step 3]
4. Expected: [What should happen]
5. Actual: [What actually happens]

## Error Messages and Logs

\`\`\`
[Exact error messages from logs - sanitize sensitive data]
\`\`\`

**Log Excerpt**:
\`\`\`
[Relevant log lines showing context before and after error]
\`\`\`

## Recommended Actions

**Immediate (P0/P1)**:
- [ ] [Specific actionable step 1]
- [ ] [Specific actionable step 2]

**Short-term**:
- [ ] [Step to prevent recurrence]
- [ ] [Monitoring/alerting improvement]

**Long-term**:
- [ ] [Architectural or process improvement]

## Historical Context

[Compare with previous occurrences from cache-memory]

**Previous Occurrences**:
- [Date 1]: [Context/resolution]
- [Date 2]: [Context/resolution]

**Trend**: [Increasing/Stable/Decreasing]

**Pattern Notes**: [Any patterns noticed across occurrences]

## References

- [Link to relevant documentation]
- [Link to related issues: githubnext/gh-aw#123, githubnext/gh-aw#456]
- [Link to workflow run: §12345]
- [Link to relevant specs or guides]

---
*Analysis performed by [Workflow Name]*
*Run: [§workflow-run-id]*
```

### Pattern Detection and Clustering

Before creating issues, cluster similar failures to avoid duplicates:

```bash
#!/bin/bash

# Generate error pattern signature
generate_error_signature() {
  local error_msg="$1"
  
  # Normalize error message:
  # - Remove timestamps, IDs, file paths, line numbers
  # - Convert to lowercase
  # - Remove extra whitespace
  normalized=$(echo "$error_msg" | \
    sed 's/[0-9]\{4\}-[0-9]\{2\}-[0-9]\{2\}//g' | \
    sed 's/[0-9]\{2\}:[0-9]\{2\}:[0-9]\{2\}//g' | \
    sed 's|/[^ ]*[.]go:[0-9]*||g' | \
    sed 's/line [0-9]*/line X/g' | \
    tr '[:upper:]' '[:lower:]' | \
    tr -s ' ')
  
  # Generate signature hash
  echo "$normalized" | md5sum | cut -d' ' -f1
}

# Store error pattern for historical tracking
store_error_pattern() {
  local signature=$1
  local error_msg=$2
  local context=$3
  local workflow=$4
  
  PATTERNS_FILE="/tmp/gh-aw/cache-memory/error-patterns.jsonl"
  mkdir -p "$(dirname "$PATTERNS_FILE")"
  
  jq -n \
    --arg sig "$signature" \
    --arg error "$error_msg" \
    --arg context "$context" \
    --arg workflow "$workflow" \
    --arg date "$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
    '{
      signature: $sig,
      error: $error,
      context: $context,
      workflow: $workflow,
      timestamp: $date
    }' >> "$PATTERNS_FILE"
  
  echo "Stored error pattern: $signature"
}

# Check pattern history (last 30 days)
check_pattern_history() {
  local signature=$1
  local patterns_file="/tmp/gh-aw/cache-memory/error-patterns.jsonl"
  
  if [ ! -f "$patterns_file" ]; then
    echo "[]"
    return
  fi
  
  # Get last 30 days of this pattern
  local cutoff_date=$(date -d '30 days ago' '+%Y-%m-%d' 2>/dev/null || \
                      date -v-30d '+%Y-%m-%d')
  
  jq -s \
    --arg sig "$signature" \
    --arg cutoff "$cutoff_date" \
    '[.[] | select(.signature == $sig and .timestamp >= $cutoff)] | 
     sort_by(.timestamp) | 
     reverse' \
    "$patterns_file"
}

# Cluster errors by signature
cluster_errors() {
  local errors_json="$1"  # Array of error objects
  
  # Group by signature and count occurrences
  echo "$errors_json" | jq -s '
    group_by(.signature) | 
    map({
      signature: .[0].signature,
      count: length,
      first_error: .[0],
      all_errors: .
    }) |
    sort_by(.count) |
    reverse
  '
}
```

### Usage Pattern: Error Analysis Workflow

```bash
#!/bin/bash
# Example: Analyze logs and create clustered issues

# 1. Parse logs and extract errors
errors=$(parse_logs_for_errors)  # Your log parsing logic

# 2. Generate signatures for each error
declare -A error_clusters
while IFS= read -r error; do
  signature=$(generate_error_signature "$error")
  error_clusters["$signature"]+="$error"$'\n'
done <<< "$errors"

# 3. For each unique signature, check history
for signature in "${!error_clusters[@]}"; do
  # Get historical occurrences
  history=$(check_pattern_history "$signature")
  occurrences=$(echo "$history" | jq 'length')
  
  # Determine if issue needed
  if [ "$occurrences" -gt 0 ]; then
    echo "Known issue (occurred $occurrences times in last 30 days)"
    # Update existing issue or skip if recently reported
  else
    echo "New issue detected: $signature"
    # Create new issue
  fi
  
  # Store this occurrence
  first_error=$(echo "${error_clusters[$signature]}" | head -n1)
  store_error_pattern "$signature" "$first_error" "$context" "$workflow_name"
done

# 4. Create issues (max 5 per run based on safe-outputs config)
# Prioritize by: 1) Severity, 2) Frequency, 3) Recency
```

### Historical Context Integration

Query cache memory for rich historical context:

```bash
# Get pattern statistics
get_pattern_statistics() {
  local signature=$1
  local history_json="$2"  # Output from check_pattern_history
  
  local total_count=$(echo "$history_json" | jq 'length')
  local first_seen=$(echo "$history_json" | jq -r 'last.timestamp')
  local last_seen=$(echo "$history_json" | jq -r 'first.timestamp')
  
  # Calculate frequency trend
  local recent_count=$(echo "$history_json" | jq '
    [.[] | select(.timestamp >= (now - 604800 | strftime("%Y-%m-%dT%H:%M:%SZ")))] |
    length
  ')
  
  # Determine trend
  local trend="Stable"
  if [ "$recent_count" -gt "$((total_count / 2))" ]; then
    trend="Increasing"
  elif [ "$recent_count" -lt "$((total_count / 4))" ]; then
    trend="Decreasing"
  fi
  
  jq -n \
    --argjson total "$total_count" \
    --arg first "$first_seen" \
    --arg last "$last_seen" \
    --arg trend "$trend" \
    --argjson recent "$recent_count" \
    '{
      total_occurrences: $total,
      first_seen: $first,
      last_seen: $last,
      recent_count: $recent,
      trend: $trend
    }'
}
```

### Issue Priority Scoring

Prioritize issues to create based on severity, frequency, and impact:

```bash
# Calculate priority score (0-100)
calculate_priority_score() {
  local severity=$1      # Critical=40, High=30, Medium=20, Low=10
  local frequency=$2     # Occurrences in last 30 days
  local is_new=$3       # 1 if new, 0 if recurring
  
  # Base score from severity
  local score=0
  case "$severity" in
    "Critical") score=40 ;;
    "High") score=30 ;;
    "Medium") score=20 ;;
    "Low") score=10 ;;
  esac
  
  # Add frequency score (max 30 points)
  local freq_score=$((frequency > 30 ? 30 : frequency))
  score=$((score + freq_score))
  
  # Add newness bonus (20 points for new issues)
  if [ "$is_new" = "1" ]; then
    score=$((score + 20))
  fi
  
  # Add recurrence penalty (10 points if recurring without fix)
  if [ "$is_new" = "0" ] && [ "$frequency" -gt 5 ]; then
    score=$((score + 10))
  fi
  
  echo "$score"
}
```

### Best Practices

1. **Always cluster before creating**: Group similar errors to avoid duplicate issues
2. **Use historical context**: Check cache memory for previous occurrences
3. **Limit issue creation**: Respect safe-outputs max limit (typically 3-5)
4. **Prioritize by impact**: Create issues for highest-priority problems first
5. **Include actionable steps**: Every issue needs concrete next steps
6. **Sanitize sensitive data**: Remove credentials, tokens, personal info from logs
7. **Link related resources**: Include workflow runs, documentation, related issues
8. **Update existing issues**: If issue exists, comment instead of creating new

### Cache Memory Structure

```
/tmp/gh-aw/cache-memory/
├── error-patterns.jsonl         # All error occurrences with timestamps
├── issue-tracker.json           # Map of signatures to issue numbers
└── resolution-history.json      # Track fixed issues and their solutions
```

## Usage Example

To use this framework in your workflow:

```yaml
imports:
  - shared/failure-analysis-issue-creation.md

safe-outputs:
  create-issue:
    max: 5
    
tools:
  cache-memory: true
  bash:
    - "jq *"
    - "grep *"
    - "awk *"
    - "sed *"
    - "md5sum *"
```

Then in your workflow prompt, refer to the patterns and functions defined here.
