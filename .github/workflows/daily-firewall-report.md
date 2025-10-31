---
on:
  schedule:
    # Every day at 10am UTC
    - cron: "0 10 * * *"
  workflow_dispatch:

permissions:
  contents: read
  actions: read
  issues: read
  pull-requests: read

safe-outputs:
  create-discussion:
    category: "audits"
    max: 1

tools:
  agentic-workflows:
  github:
    toolsets:
      - default
      - actions
  bash:
    - "*"
  edit:
imports:
  - shared/reporting.md
---

# Daily Firewall Logs Collector and Reporter

Collect and analyze firewall logs from all agentic workflows that use the firewall feature.

## Objective

Generate a comprehensive daily report of all rejected domains across all agentic workflows that use the firewall feature. This helps identify:
- Which domains are being blocked
- Patterns in blocked traffic
- Potential issues with network permissions
- Security insights from blocked requests

## Instructions

### Step 1: Identify Workflows with Firewall Feature

1. List all workflows in the repository
2. For each workflow that has `network.firewall: true` in its frontmatter, note the workflow name
3. Create a list of all firewall-enabled workflows

**Example frontmatter structure:**
```yaml
network:
  firewall: true
```

**Note:** The firewall field is under `network`, not `features`.

### Step 2: Collect Recent Workflow Runs

For each firewall-enabled workflow:
1. Get up to 10 workflow runs that occurred within the past 7 days (if there are fewer than 10 runs in that window, include all available; if there are more, include only the most recent 10)
2. For each run ID, use the `audit` tool from the agentic-workflows MCP server with `--json` flag to get detailed firewall information
3. Store the run ID, workflow name, and timestamp for tracking

**Using the audit tool:**
```bash
# Get firewall analysis in JSON format
gh aw audit <run-id> --json

# Example jq filter to extract firewall data:
gh aw audit <run-id> --json | jq '{
  run_id: .overview.run_id,
  workflow: .overview.workflow_name,
  firewall: .firewall_analysis // {},
  denied_domains: .firewall_analysis.denied_domains // [],
  allowed_domains: .firewall_analysis.allowed_domains // [],
  total_requests: .firewall_analysis.total_requests // 0,
  denied_requests: .firewall_analysis.denied_requests // 0
}'
```

**Important:** Do NOT manually download and parse firewall log files. Always use the `audit` tool which provides structured firewall analysis data including:
- Total requests, allowed requests, denied requests
- Lists of allowed and denied domains
- Request statistics per domain

### Step 3: Parse and Analyze Firewall Logs

Use the JSON output from the `audit` tool to extract firewall information. The `firewall_analysis` field in the audit JSON contains:
- `total_requests` - Total number of network requests
- `allowed_requests` - Count of allowed requests
- `denied_requests` - Count of denied/blocked requests
- `allowed_domains` - Array of unique allowed domains
- `denied_domains` - Array of unique denied/blocked domains
- `requests_by_domain` - Object mapping domains to request statistics (allowed/denied counts)

**Example jq filter for aggregating denied domains:**
```bash
# Get only denied domains across multiple runs
gh aw audit <run-id> --json | jq -r '.firewall_analysis.denied_domains[]? // empty'

# Get denied domain statistics with counts
gh aw audit <run-id> --json | jq -r '
  .firewall_analysis.requests_by_domain // {} | 
  to_entries[] | 
  select(.value.denied > 0) | 
  "\(.key): \(.value.denied) denied, \(.value.allowed) allowed"
'
```

For each workflow run with firewall data:
1. Extract the firewall analysis from the audit JSON output
2. Track the following metrics per workflow:
   - Total requests (from `total_requests`)
   - Allowed requests count (from `allowed_requests`)
   - Denied requests count (from `denied_requests`)
   - List of unique denied domains (from `denied_domains`)
   - Domain-level statistics (from `requests_by_domain`)

### Step 4: Aggregate Results

Combine data from all workflows:
1. Create a master list of all denied domains across all workflows
2. Track how many times each domain was blocked
3. Track which workflows blocked which domains
4. Calculate overall statistics:
   - Total workflows analyzed
   - Total runs analyzed
   - Total denied domains (unique)
   - Total denied requests

### Step 5: Generate Report

Create a comprehensive markdown report with the following sections:

#### 1. Executive Summary
- Date of report (today's date)
- Total workflows analyzed
- Total runs analyzed  
- Total unique denied domains
- Total denied requests
- Percentage of denied vs allowed traffic

#### 2. Top Blocked Domains
A table showing the most frequently blocked domains:
- Domain name
- Number of times blocked
- Workflows that blocked it
- Example URLs (if available)

Sort by frequency (most blocked first), show top 20.

#### 3. Blocked Domains by Workflow
For each workflow that had blocked domains:
- Workflow name
- Number of unique blocked domains
- List of blocked domains
- Total denied requests for this workflow

#### 4. Complete Blocked Domains List
An alphabetically sorted list of all unique blocked domains with:
- Domain name
- Total occurrences across all workflows
- First seen date (from run timestamps)

#### 5. Recommendations
Based on the analysis, provide:
- Domains that appear to be legitimate services that should be allowlisted
- Potential security concerns (e.g., suspicious domains)
- Suggestions for network permission improvements
- Workflows that might need their network permissions updated

### Step 6: Create Discussion

Create a new GitHub discussion with:
- **Title**: "Daily Firewall Report - [Today's Date]"
- **Category**: audits
- **Body**: The complete markdown report generated in Step 5

## Notes

- If no firewall logs are found, create a simple report stating that no firewall-enabled workflows ran in the past 7 days
- Include timestamps and run URLs for traceability
- Use tables and formatting for better readability
- Add emojis to make the report more engaging (🔥 for firewall, 🚫 for blocked, ✅ for allowed)

## Expected Output

A GitHub discussion in the "audits" category containing a comprehensive daily firewall analysis report.
