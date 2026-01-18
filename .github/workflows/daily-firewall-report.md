---
description: Collects and reports on firewall log events to monitor network security and access patterns
on:
  schedule:
    # Every day at 10am UTC
    - cron: daily
  workflow_dispatch:

permissions:
  contents: read
  actions: read
  issues: read
  pull-requests: read

tracker-id: daily-firewall-report
timeout-minutes: 45

safe-outputs:
  upload-asset:
  create-discussion:
    expires: 3d
    category: "audits"
    max: 1
    close-older-discussions: true

tools:
  agentic-workflows:
  github:
    toolsets:
      - default
      - actions
  bash:
    - "*"
  edit:
  repo-memory:
    branch-name: memory/firewall-reports
    description: "Firewall analysis history and aggregated data"
imports:
  - shared/mcp/gh-aw.md
  - shared/reporting.md
  - shared/trending-charts-simple.md
---

{{#runtime-import? .github/shared-instructions.md}}

# Daily Firewall Logs Collector and Reporter

Collect and analyze firewall logs from all agentic workflows that use the firewall feature.

## 📊 Trend Charts Requirement

**IMPORTANT**: Generate exactly 2 trend charts that showcase firewall activity patterns over time.

### Chart Generation Process

**Phase 1: Data Collection**

Collect data for the past 30 days (or available data) from cache memory and firewall audit logs:

1. **Firewall Request Data**:
   - Count of allowed requests per day
   - Count of blocked requests per day
   - Total requests per day

2. **Top Blocked Domains Data**:
   - Frequency of top 10 blocked domains over the period
   - Trends in blocking patterns by domain category

**Phase 2: Data Preparation**

1. Create CSV files in `/tmp/gh-aw/python/data/` with the collected data:
   - `firewall_requests.csv` - Daily allowed/blocked request counts
   - `blocked_domains.csv` - Top blocked domains with frequencies

2. Each CSV should have a date column and metric columns with appropriate headers

**Phase 3: Chart Generation**

Generate exactly **2 high-quality trend charts**:

**Chart 1: Firewall Request Trends**
- Stacked area chart or multi-line chart showing:
  - Allowed requests (area/line, green)
  - Blocked requests (area/line, red)
  - Total requests trend line
- X-axis: Date (last 30 days)
- Y-axis: Request count
- Save as: `/tmp/gh-aw/python/charts/firewall_requests_trends.png`

**Chart 2: Top Blocked Domains Frequency**
- Horizontal bar chart showing:
  - Top 10-15 most frequently blocked domains
  - Total block count for each domain
  - Color-coded by domain category if applicable
- X-axis: Block count
- Y-axis: Domain names
- Save as: `/tmp/gh-aw/python/charts/blocked_domains_frequency.png`

**Chart Quality Requirements**:
- DPI: 300 minimum
- Figure size: 12x7 inches for better readability
- Use seaborn styling with a professional color palette
- Include grid lines for easier reading
- Clear, large labels and legend
- Title with context (e.g., "Firewall Activity - Last 30 Days")
- Annotations for significant spikes or patterns

**Phase 4: Upload Charts**

1. Upload both charts using the `upload asset` tool
2. Collect the returned URLs for embedding in the discussion

**Phase 5: Embed Charts in Discussion**

Include the charts in your firewall report with this structure:

```markdown
## 📈 Firewall Activity Trends

### Request Patterns
![Firewall Request Trends](URL_FROM_UPLOAD_ASSET_CHART_1)

[Brief 2-3 sentence analysis of firewall activity trends, noting increases in blocked traffic or changes in patterns]

### Top Blocked Domains
![Blocked Domains Frequency](URL_FROM_UPLOAD_ASSET_CHART_2)

[Brief 2-3 sentence analysis of frequently blocked domains, identifying potential security concerns or overly restrictive rules]
```

### Python Implementation Notes

- Use pandas for data manipulation and date handling
- Use matplotlib.pyplot and seaborn for visualization
- Set appropriate date formatters for x-axis labels
- Use `plt.xticks(rotation=45)` for readable date labels
- Apply `plt.tight_layout()` before saving
- Handle cases where data might be sparse or missing

### Error Handling

If insufficient data is available (less than 7 days):
- Generate the charts with available data
- Add a note in the analysis mentioning the limited data range
- Consider using a bar chart instead of line chart for very sparse data

---

## Objective

Generate a comprehensive daily report of all rejected domains across all agentic workflows that use the firewall feature. This helps identify:
- Which domains are being blocked
- Patterns in blocked traffic
- Potential issues with network permissions
- Security insights from blocked requests

## Instructions

### MCP Servers are Pre-loaded

**IMPORTANT**: The MCP servers configured in this workflow (including `gh-aw` with tools like `logs` and `audit`) are automatically loaded and available at agent startup. You do NOT need to:
- Use the inspector tool to discover MCP servers
- Run any external tools to check available MCP servers
- Verify or list MCP servers before using them

Simply call the MCP tools directly as described in the steps below. If you want to know what tools are available, you can list them using your built-in tool listing capability.

### Step 0: Check Repo Memory for Recent Analysis

**EFFICIENCY FIRST**: Before starting the full analysis:

1. Check `/tmp/gh-aw/repo-memory-default/memory/default/` for the most recent report
2. If a report exists from the last 24 hours:
   - Read the cached run IDs that were analyzed
   - Determine if any new workflow runs have occurred since then
   - If no new runs, skip to Step 5 (Generate Report) using the same cached run IDs, but **always re-fetch fresh data from the audit tool** for accurate counts
3. Store the following in repo memory for the next run:
   - Last analysis timestamp
   - List of run IDs analyzed

**IMPORTANT**: Never cache or reuse aggregated statistics (blocked counts, allowed counts, domain lists). Always compute these fresh from the audit tool to ensure accurate reporting. Only cache run IDs to avoid re-discovering the same workflow runs.

### Step 1: Collect Recent Firewall-Enabled Workflow Runs

Use the `logs` tool from the agentic-workflows MCP server to efficiently collect workflow runs that have firewall enabled:

**Using the logs tool:**
Call the `logs` tool with the following parameters:
- `firewall`: true (boolean - to filter only runs with firewall enabled)
- `start_date`: "-7d" (to get runs from the past 7 days)
- `count`: 100 (to get up to 100 matching runs)

The tool will:
1. Filter runs based on the `steps.firewall` field in `aw_info.json` (e.g., "squid" when enabled)
2. Return only runs where firewall was enabled
3. Limit to runs from the past 7 days
4. Return up to 100 matching runs

**Tool call example:**
```json
{
  "firewall": true,
  "start_date": "-7d",
  "count": 100
}
```

### Step 2: Analyze Firewall Logs from Collected Runs

For each run collected in Step 1:
1. Use the `audit` tool from the agentic-workflows MCP server to get detailed firewall information
2. Store the run ID, workflow name, and timestamp for tracking

**Using the audit tool:**
Call the `audit` tool with the run_id parameter for each run from Step 1.

**Tool call example:**
```json
{
  "run_id": 12345678
}
```

The audit tool returns structured firewall analysis data including:
- Total requests, allowed requests, blocked requests
- Lists of allowed and blocked domains
- Request statistics per domain

**Example of extracting firewall data from audit result:**
```javascript
// From the audit tool result, access:
result.firewall_analysis.blocked_domains  // Array of blocked domain names
result.firewall_analysis.allowed_domains  // Array of allowed domain names
result.firewall_analysis.total_requests   // Total number of network requests
result.firewall_analysis.blocked_requests  // Number of blocked requests
```

**Important:** Do NOT manually download and parse firewall log files. Always use the `audit` tool which provides structured firewall analysis data.

### Step 3: Parse and Analyze Firewall Logs

Use the JSON output from the `audit` tool to extract firewall information. The `firewall_analysis` field in the audit JSON contains:
- `total_requests` - Total number of network requests
- `allowed_requests` - Count of allowed requests
- `blocked_requests` - Count of blocked requests
- `allowed_domains` - Array of unique allowed domains
- `blocked_domains` - Array of unique blocked domains
- `requests_by_domain` - Object mapping domains to request statistics (allowed/blocked counts)

**Example jq filter for aggregating blocked domains:**
```bash
# Get only blocked domains across multiple runs
gh aw audit <run-id> --json | jq -r '.firewall_analysis.blocked_domains[]? // empty'

# Get blocked domain statistics with counts
gh aw audit <run-id> --json | jq -r '
  .firewall_analysis.requests_by_domain // {} | 
  to_entries[] | 
  select(.value.blocked > 0) | 
  "\(.key): \(.value.blocked) blocked, \(.value.allowed) allowed"
'
```

For each workflow run with firewall data:
1. Extract the firewall analysis from the audit JSON output
2. Track the following metrics per workflow:
   - Total requests (from `total_requests`)
   - Allowed requests count (from `allowed_requests`)
   - Blocked requests count (from `blocked_requests`)
   - List of unique blocked domains (from `blocked_domains`)
   - Domain-level statistics (from `requests_by_domain`)

### Step 4: Aggregate Results

Combine data from all workflows:
1. Create a master list of all blocked domains across all workflows
2. Track how many times each domain was blocked
3. Track which workflows blocked which domains
4. Calculate overall statistics:
   - Total workflows analyzed
   - Total runs analyzed
   - Total blocked domains (unique)
   - Total blocked requests

### Step 5: Generate Report

Create a comprehensive markdown report with the following sections:

#### 1. Executive Summary
- Date of report (today's date)
- Total workflows analyzed
- Total runs analyzed  
- Total network requests monitored
  - ✅ **Allowed**: Count of successful requests
  - 🚫 **Blocked**: Count of blocked requests
- **Block rate**: Percentage of blocked requests (blocked / total * 100)
- Total unique blocked domains

> **Terminology Note**: 
> - **Allowed requests** = Requests that successfully reached their destination
> - **Blocked requests** = Requests that were prevented by the firewall
> - A 0% block rate with listed blocked domains indicates domains that would 
>   be blocked if accessed, but weren't actually accessed during this period

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
- Total blocked requests for this workflow

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