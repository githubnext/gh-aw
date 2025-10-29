---
on:
  schedule:
    # Every day at 10am UTC
    - cron: "0 10 * * *"
  workflow_dispatch:

permissions:
  contents: read
  actions: read

safe-outputs:
  create-discussion:
    category: "audits"
    max: 1

tools:
  github:
    toolsets:
      - default
      - actions
  bash:
    - "*"
  edit:
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
2. Download the firewall logs artifact (named `squid-logs-{workflow-name}`) from each run
   - **Note:** `{workflow-name}` refers to the workflow file name (e.g., `dev.firewall`), not the display name. Any special characters or spaces in the file name are preserved as-is in the artifact name. For example, if the workflow file is `dev.firewall.md`, the artifact will be named `squid-logs-dev.firewall`.
   - **Artifact contents:** Each artifact contains Squid proxy logs:
     - `access.log` - HTTP requests with domains, status codes, and connection types
     - `cache.log` - Cache operations and errors (typically less relevant for domain analysis)
3. Store the run ID, workflow name, and timestamp for tracking

### Step 3: Parse and Analyze Firewall Logs

For each downloaded firewall log:
1. Parse the access log files to extract domain information
   - **Squid native log format:** Each line contains space-separated fields:
     ```
     timestamp source_ip:source_port target_domain:target_port target_ip:target_port http_version method status_code connection_type:hierarchy domain:port user_agent
     ```
   - **Example log line:**
     ```
     1761610340.601 172.30.0.20:46014 api.github.com:443 140.82.116.6:443 1.1 CONNECT 200 TCP_TUNNEL:HIER_DIRECT api.github.com:443 "-"
     ```
   - **Parsing rules:**
     - **Allowed requests:** Lines with `TCP_*` connection types (e.g., `TCP_TUNNEL`, `TCP_MISS`, `TCP_HIT`)
     - **Denied/Rejected requests:** Lines with `error:` in the connection type field or `NONE_NONE` status
2. Categorize domains as:
   - **Allowed**: Successfully accessed domains
   - **Denied/Rejected**: Blocked domains
3. Track the following metrics per workflow:
   - Total requests
   - Allowed requests count
   - Denied requests count
   - List of unique denied domains

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
