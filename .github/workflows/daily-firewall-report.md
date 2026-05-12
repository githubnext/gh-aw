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
  discussions: read
  security-events: read

tracker-id: daily-firewall-report
timeout-minutes: 45

safe-outputs:
  upload-asset:
    max: 3
    allowed-exts: [.png, .jpg, .jpeg, .svg]
tools:
  cli-proxy: true
  agentic-workflows:
  github:
    toolsets:
      - all
  bash:
    - "*"
  edit:
imports:
  - uses: shared/daily-audit-charts.md
    with:
      title-prefix: "[daily-firewall-report] "


  - shared/observability-otlp.md
firewall:
  effective-token-steering: true
---

{{#runtime-import? .github/shared-instructions.md}}

# Daily Firewall Logs Collector and Reporter

Collect and analyze firewall logs from all agentic workflows that use the firewall feature.

## 📊 Trend Charts

Use the `firewall-chart-generator` agent to collect 30-day firewall data, generate the two trend charts, and return their upload URLs. Record the returned `CHART1_URL` and `CHART2_URL` values for embedding in Step 5 of the report using markdown image links:
- `![Firewall Request Trends](<CHART1_URL returned by the sub-agent>)`
- `![Blocked Domains Frequency](<CHART2_URL returned by the sub-agent>)`
If the agent returns an `error` field, omit both image embeds and include a brief note in the final report that chart generation failed with the reported reason.

---

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

### Step 0: Fresh Analysis - No Caching

**ALWAYS PERFORM FRESH ANALYSIS**: This report must always use fresh data from the audit tool. 

**DO NOT**:
- Skip analysis based on cached results
- Reuse aggregated statistics from previous runs
- Check for or use any cached run IDs, counts, or domain lists

**ALWAYS**:
- Collect all workflow runs fresh using the `logs` tool
- Fetch complete firewall data from the `audit` tool for each run
- Compute all statistics fresh (blocked counts, allowed counts, domain lists)

This ensures accurate, up-to-date reporting for every run of this workflow.

### Step 1: Collect Recent Firewall-Enabled Workflow Runs

Use the `logs` tool from the agentic-workflows MCP server to efficiently collect workflow runs that have firewall enabled (see `workflow_runs_analyzed` in scratchpad/metrics-glossary.md - Scope: Last 7 days):

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

### Step 1.5: Early Exit if No Data

**IMPORTANT**: If Step 1 returns zero workflow runs (no firewall-enabled workflows ran in the past 7 days):

1. **Do NOT create a discussion or report**
2. **Exit early** with a brief log message: "No firewall-enabled workflow runs found in the past 7 days. Exiting without creating a report."
3. **Stop processing** - do not proceed to Step 2 or any subsequent steps

This prevents creating empty or meaningless reports when there's no data to analyze.

### Step 2–4: Audit and Aggregate Firewall Data

Pass the list of run IDs from Step 1 to the `firewall-data-aggregator` agent as a JSON array of integers (for example: `[123,456,789]`).
Example invocation payload:
```json
{
  "run_ids": [123,456,789]
}
```
Use the returned JSON object (keys: `totals`, `blocked_domains`, `policy_rules`, `denied_requests`) as the data source for Step 5 (Generate Report).

### Step 5: Generate Report

Create a comprehensive markdown report following the formatting guidelines above. Structure your report as follows:

#### Section 1: Executive Summary (Always Visible)
A brief 1-2 paragraph overview including:
- Date of report (today's date)
- Total workflows analyzed (`workflow_runs_analyzed`)
- Total runs analyzed
- Overall firewall activity snapshot (key highlights, trends, concerns)

#### Section 2: Key Metrics (Always Visible)
Present the core statistics:
- Total network requests monitored (`firewall_requests_total`)
  - ✅ **Allowed** (`firewall_requests_allowed`): Count of successful requests
  - 🚫 **Blocked** (`firewall_requests_blocked`): Count of blocked requests
- **Block rate**: Percentage of blocked requests (blocked / total * 100)
- Total unique blocked domains (`firewall_domains_blocked`)

> **Terminology Note**: 
> - **Allowed requests** = Requests that successfully reached their destination
> - **Blocked requests** = Requests that were prevented by the firewall
> - A 0% block rate with listed blocked domains indicates domains that would 
>   be blocked if accessed, but weren't actually accessed during this period

#### Section 3: Top Blocked Domains (Always Visible)
A table showing the most frequently blocked domains:
- Domain name
- Number of times blocked
- Workflows that blocked it
- Domain category (Development Services, Social Media, Analytics/Tracking, CDN, Other)

Sort by frequency (most blocked first), show top 20.

#### Section 4: Policy Rule Attribution (Always Visible — when data available)

**Include this section when `policy_analysis` data was available for at least one run.**

This section provides rule-level insights that go beyond simple domain counts, showing *which policy rules* are handling traffic and *why* specific requests were denied.

**4a. Policy Configuration**

Show the policy summary from the most recent run:
- Number of rules, SSL Bump status, DLP status
- Example: "📋 Policy: 12 rules, SSL Bump disabled, DLP disabled"

**4b. Policy Rule Hit Table**

Show aggregated rule hit counts across all analyzed runs:

```markdown
| Rule | Action | Description | Total Hits |
|------|--------|-------------|------------|
| allow-github | 🟢 allow | Allow GitHub domains | 523 |
| allow-npm | 🟢 allow | Allow npm registry | 187 |
| deny-blocked-plain | 🔴 deny | Deny all other HTTP/HTTPS | 12 |
| deny-default | 🔴 deny | Default deny | 3 |
```

- Sort by hits (descending)
- Include all rules that had at least 1 hit
- Use 🟢 for allow rules and 🔴 for deny rules in the Action column

**4c. Denied Requests with Rule Attribution**

Show denied requests grouped by rule, with domain details:

```markdown
| Domain | Deny Rule | Reason | Occurrences |
|--------|-----------|--------|-------------|
| evil.com:443 | deny-blocked-plain | Domain not in allowlist | 5 |
| tracker.io:443 | deny-blocked-plain | Domain not in allowlist | 3 |
| unknown.host:80 | deny-default | Default deny | 1 |
```

- Group by domain + rule combination
- Sort by occurrences (descending)
- Show top 30 entries; wrap the full list in `<details>` if more than 30

**4d. Rule Effectiveness Summary**

Provide a brief analysis:
- Which deny rules are doing the most work (catching the most unauthorized traffic)
- Which allow rules handle the most traffic (busiest legitimate pathways)
- Any rules with zero hits that could be removed or indicate unused policy entries
- Any `(implicit-deny)` attributions that indicate gaps in the policy (traffic denied without matching any explicit rule)

#### Section 5: Detailed Request Patterns (In `<details>` Tags)
**IMPORTANT**: Wrap this entire section in a collapsible `<details>` block:

```markdown
<details>
<summary>View Detailed Request Patterns by Workflow</summary>

For each workflow that had blocked domains, provide a detailed breakdown:

#### Workflow: [workflow-name] (X runs analyzed)

| Domain | Blocked Count | Allowed Count | Block Rate | Category |
|--------|---------------|---------------|------------|----------|
| example.com | 15 | 5 | 75% | Social Media |
| api.example.org | 10 | 0 | 100% | Development |

- Total blocked requests: [count]
- Total unique blocked domains: [count]
- Most frequently blocked domain: [domain]

[Repeat for all workflows with blocked domains]

</details>
```

#### Section 6: Complete Blocked Domains List (In `<details>` Tags)
**IMPORTANT**: Wrap this entire section in a collapsible `<details>` block:

```markdown
<details>
<summary>View Complete Blocked Domains List</summary>

An alphabetically sorted list of all unique blocked domains:

| Domain | Total Blocks | First Seen | Workflows |
|--------|--------------|------------|-----------|
| [domain] | [count] | [date] | [workflow-list] |
| ... | ... | ... | ... |

</details>
```

#### Section 7: Security Recommendations (Always Visible)
Based on the analysis, provide actionable insights:
- Domains that appear to be legitimate services that should be allowlisted
- Potential security concerns (e.g., suspicious domains)
- Suggestions for network permission improvements
- Workflows that might need their network permissions updated
- Policy rule suggestions (e.g., rules with zero hits that could be removed, domains that should be added to allow rules)

### Step 6: Create Discussion

Create a new GitHub discussion with:
- **Title**: "Daily Firewall Report - [Today's Date]"
- **Category**: audits
- **Body**: The complete markdown report following the formatting guidelines and structure defined in Step 5

Ensure the discussion body:
- Uses h3 (###) for main section headers
- Uses h4 (####) for subsection headers
- Wraps detailed data (per-workflow breakdowns, complete domain list) in `<details>` tags
- Keeps critical information visible (summary, key metrics, top domains, recommendations)

## Notes

- **Early exit**: If no firewall-enabled workflow runs are found in the past 7 days, exit early without creating a report (see Step 1.5)
- Include timestamps and run URLs for traceability
- Use tables and formatting for better readability
- Add emojis to make the report more engaging (🔥 for firewall, 🚫 for blocked, ✅ for allowed)

## Expected Output

A GitHub discussion in the "audits" category containing a comprehensive daily firewall analysis report.

{{#runtime-import shared/noop-reminder.md}}

## agent: `firewall-chart-generator`
---
model: small
description: Collects 30-day firewall data, generates two trend charts, uploads them, and returns chart URLs
---
You are a chart-generation sub-agent for daily firewall reporting.

Task:
1. Collect firewall request and blocked-domain trend data for the past 30 days (or all available days).
2. Create chart inputs under `/tmp/gh-aw/python/data/`.
3. Generate exactly 2 charts under `/tmp/gh-aw/python/charts/`:
   - `firewall_requests_trends.png` (allowed, blocked, total request trends over time)
   - `blocked_domains_frequency.png` (top blocked domains by frequency)
4. Upload both charts with the `upload_asset` safe-output tool using absolute paths.
5. Return a JSON object with these exact field mappings:
   - `CHART1_URL` = uploaded URL for `firewall_requests_trends.png`
   - `CHART2_URL` = uploaded URL for `blocked_domains_frequency.png`

Requirements:
- Use pandas + matplotlib + seaborn.
- Use readable labels, legends, and professional styling.
- Handle sparse data gracefully and still produce both charts.

Return ONLY a JSON object:
```json
{
  "CHART1_URL": "<url>",
  "CHART2_URL": "<url>"
}
```

If chart generation or upload ultimately fails after reasonable retries, return:
```json
{
  "CHART1_URL": "",
  "CHART2_URL": "",
  "error": "<brief reason>"
}
```

## agent: `firewall-data-aggregator`
---
model: small
description: Audits firewall-enabled run IDs and returns aggregated firewall, policy-rule, and denied-request statistics
---
You are a firewall data aggregation sub-agent.

Input:
- A JSON array of workflow run IDs as integers (for example: `[123,456,789]`).
- Iterate through the array and call `audit` for each run ID.

Task:
1. For each run ID, call the `audit` tool.
2. Extract `firewall_analysis` data and aggregate:
   - total requests, allowed requests, blocked requests
   - blocked domain frequencies
3. If `policy_analysis` is present, aggregate:
   - rule hit totals by rule ID/action/description
   - denied request frequencies grouped by domain + rule + reason

Return ONLY a JSON object with this shape:
```json
{
  "totals": {
    "workflow_runs_analyzed": 0,
    "firewall_requests_total": 0,
    "firewall_requests_allowed": 0,
    "firewall_requests_blocked": 0,
    "firewall_domains_blocked": 0
  },
  "blocked_domains": [
    {
      "domain": "example.com",
      "blocked_count": 0,
      "workflows": []
    }
  ],
  "policy_rules": [
    {
      "rule_id": "allow-github",
      "action": "allow",
      "description": "Allow GitHub domains",
      "hits": 0
    }
  ],
  "denied_requests": [
    {
      "domain": "evil.com:443",
      "rule_id": "deny-default",
      "reason": "Default deny",
      "occurrences": 0
    }
  ]
}
```
