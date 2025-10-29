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
2. For each workflow that has `features.firewall: true` in its frontmatter, note the workflow name
3. Create a list of all firewall-enabled workflows

### Step 2: Collect Recent Workflow Runs

For each firewall-enabled workflow:
1. Get up to 10 workflow runs that occurred within the past 7 days (if there are fewer than 10 runs in that window, include all available; if there are more, include only the most recent 10)
2. Download the firewall logs artifact (named `squid-logs-{workflow-name}`) from each run
   - **Note:** `{workflow-name}` refers to the workflow file name (e.g., `dev.firewall`), not the display name. Any special characters or spaces in the file name are preserved as-is in the artifact name. For example, if the workflow file is `dev.firewall.md`, the artifact will be named `squid-logs-dev.firewall`.
3. Store the run ID, workflow name, and timestamp for tracking

### Step 3: Parse and Analyze Firewall Logs

**CRITICAL**: You MUST download and parse the actual Squid access log files from the artifacts. Use `gh run download <run-id> -n squid-logs-{workflow-name}` to download artifacts, then parse the `access.log` files line by line.

**Squid Access Log Format**:
```
timestamp client_ip:port domain dest_ip:port proto method status decision url user_agent
```

**Example Log Entries**:
```
# Allowed request (status 200, TCP_TUNNEL)
1761332530.474 172.30.0.20:35288 api.github.com:443 140.82.112.22:443 1.1 CONNECT 200 TCP_TUNNEL:HIER_DIRECT api.github.com:443 "-"

# Denied request (status 403, NONE_NONE)
1761332531.123 172.30.0.20:35289 blocked.example.com:443 140.82.112.23:443 1.1 CONNECT 403 NONE_NONE:HIER_NONE blocked.example.com:443 "-"
```

**Field Positions**:
- Field 1 (index 0): Unix timestamp with decimal precision
- Field 3 (index 2): Destination domain with port (e.g., `api.github.com:443`)
- Field 7 (index 6): HTTP status code (`200` for allowed, `403` for denied)
- Field 8 (index 7): Cache result/decision (`TCP_TUNNEL` for allowed, `TCP_DENIED` or `NONE_NONE` for denied)

**Parsing Instructions**:
1. Download firewall log artifacts to `/tmp/gh-aw/agent/firewall-logs/` for processing
2. For each artifact, extract the `access.log` file
3. Parse each line using space-separated fields (handle quoted user-agent strings)
4. Extract domain from field 3, removing port number for domain-level tracking
5. Determine allowed vs denied:
   - **Allowed**: Status code 200/206/304 OR decision contains `TCP_TUNNEL`/`TCP_HIT`/`TCP_MISS`
   - **Denied**: Status code 403/407 OR decision contains `NONE_NONE`/`TCP_DENIED`
6. Convert Unix timestamps to human-readable format for reporting

For each downloaded firewall log:
1. Parse the access log files to extract domain information
2. Categorize domains as:
   - **Allowed**: Successfully accessed domains (status 200/206/304 or TCP_TUNNEL/TCP_HIT/TCP_MISS)
   - **Denied/Rejected**: Blocked domains (status 403/407 or NONE_NONE/TCP_DENIED)
3. Track the following metrics per workflow run:
   - Total requests
   - Allowed requests count  
   - Denied requests count
   - List of unique allowed domains with request counts
   - List of unique denied domains with request counts
   - First seen timestamp (earliest request for each domain)
   - Last seen timestamp (most recent request for each domain)
4. Store intermediate results in `/tmp/gh-aw/agent/` for aggregation

### Step 4: Aggregate Results

Combine data from all workflows:
1. Create a master list of ALL domains (both allowed and denied) across all workflows
2. For each domain, track:
   - Total request count (allowed + denied)
   - Number of times blocked
   - Number of times allowed
   - Which workflows accessed it (workflow names list)
   - First seen timestamp (earliest across all runs)
   - Last seen timestamp (most recent across all runs)
   - Example URLs
3. Calculate overall statistics:
   - Total workflows analyzed
   - Total runs analyzed
   - Total unique domains (allowed + denied)
   - Total denied domains (unique)
   - Total allowed domains (unique)
   - Total denied requests
   - Total allowed requests
   - Percentage of denied vs allowed traffic

### Step 5: Generate Report

Create a comprehensive markdown report with the following sections:

#### 1. Executive Summary
- Date of report (today's date in YYYY-MM-DD format)
- Total workflows analyzed
- Total runs analyzed
- Total unique domains accessed (allowed + denied)
- Total unique denied domains
- Total unique allowed domains
- Total denied requests
- Total allowed requests
- Percentage of denied vs allowed traffic

#### 2. Domain Access Summary Table

**A comprehensive table showing ALL domains (allowed + blocked) across all runs:**

Table format:
```
| Domain | Status | Count | Workflows | First Seen | Last Seen | Example URL |
|--------|--------|-------|-----------|------------|-----------|-------------|
```

- **Domain**: Domain name (without port, e.g., `api.github.com`)
- **Status**: ✅ (allowed) or 🚫 (blocked)
- **Count**: Total number of requests (allowed or denied)
- **Workflows**: Comma-separated list of workflow names that accessed this domain
- **First Seen**: Human-readable date/time of first request (convert Unix timestamp)
- **Last Seen**: Human-readable date/time of most recent request
- **Example URL**: One example URL from the logs

**Sorting**: Show blocked domains first (🚫), then allowed domains (✅), each group sorted by count (descending)

**Limit**: Show top 50 domains. If more exist, add a note: "Note: Showing top 50 of N total domains"

#### 3. Top Blocked Domains

A table showing the most frequently blocked domains:

Table format:
```
| Domain | Blocked Count | Workflows | First Seen | Last Seen | Example URL |
|--------|---------------|-----------|------------|-----------|-------------|
```

- **Domain**: Domain name (without port)
- **Blocked Count**: Number of times this domain was blocked
- **Workflows**: Comma-separated list of workflows that blocked it
- **First Seen**: Date/time of first blocked request
- **Last Seen**: Date/time of most recent blocked request
- **Example URL**: One example blocked URL from the logs

Sort by blocked count (most blocked first), show top 20.

#### 4. Blocked Domains by Workflow

For each workflow that had blocked OR allowed domains:

**Format for each workflow:**
```
### [Workflow Name]
- **Total Requests**: N (M allowed, P denied)
- **Unique Allowed Domains**: X
- **Unique Blocked Domains**: Y

**Top 10 Most Frequent Allowed Domains:**
- domain1.com (N requests)
- domain2.com (N requests)
...

**Blocked Domains** (alphabetically sorted):
- blocked1.com (N requests)
- blocked2.com (N requests)
...
```

Sort workflows alphabetically by name.

#### 5. Complete Domain Lists

##### 5.1 All Blocked Domains

Alphabetically sorted list of ALL unique blocked domains with details:

Table format:
```
| Domain | Total Blocks | Workflows | First Seen | Last Seen |
|--------|--------------|-----------|------------|-----------|
```

For the **Workflows** column:
- If fewer than 5 workflows: show comma-separated workflow names
- If 5 or more workflows: show "Multiple workflows (N)"

##### 5.2 All Allowed Domains

Alphabetically sorted list of allowed domains (limit to top 50 by request count):

Table format:
```
| Domain | Total Requests | Workflows | First Seen | Last Seen |
|--------|----------------|-----------|------------|-----------|
```

For the **Workflows** column:
- If fewer than 5 workflows: show comma-separated workflow names  
- If 5 or more workflows: show "Multiple workflows (N)"

**Note**: Add "Showing top 50 of N total allowed domains" if more than 50 exist.

#### 6. Recommendations

Based on the actual data from the logs, provide:

1. **Allowlist Improvements**:
   - Domains that appear to be legitimate services (e.g., package registries, CDNs, cloud providers) that are frequently blocked and should be allowlisted
   - Patterns in blocked domains that suggest missing network permissions (e.g., multiple `*.cdn.com` domains blocked)

2. **Security Concerns**:
   - Suspicious or unusual domains that were blocked (good firewall behavior)
   - Domains that might indicate compromised dependencies or supply chain attacks

3. **Network Permission Optimization**:
   - Workflows with high block rates that may need updated network permissions
   - Common allowed domains that could be added to default allowlists
   - Suggestions for using ecosystem identifiers (e.g., `python`, `node`, `containers`) instead of listing individual domains

4. **Pattern Analysis**:
   - CDN or cloud provider patterns (e.g., AWS, Azure, GCP domains)
   - Language ecosystem patterns (e.g., PyPI mirrors, NPM registries)
   - Temporal patterns (domains that appeared recently vs historically)

### Step 6: Create Discussion

Create a new GitHub discussion with:
- **Title**: "Daily Firewall Report - [Today's Date]"
- **Category**: audits
- **Body**: The complete markdown report generated in Step 5

## Notes

- **CRITICAL**: You MUST download and parse the actual Squid access log files. Do not just list artifact metadata.
- Use `gh run download <run-id> -n squid-logs-{workflow-name}` to download artifacts
- Store downloaded artifacts in `/tmp/gh-aw/agent/firewall-logs/` for processing
- Parse logs line by line, handling potential errors gracefully (skip malformed lines)
- Access log files may contain thousands of entries - process them efficiently
- If no firewall logs are found, create a simple report stating that no firewall-enabled workflows ran in the past 7 days
- Include timestamps and run URLs for traceability
- Use tables and formatting for better readability
- Add emojis to make the report more engaging (🔥 for firewall, 🚫 for blocked, ✅ for allowed)
- Convert Unix timestamps to human-readable format (e.g., "2025-01-15 14:30:25 UTC")
- For domain extraction, remove port numbers (e.g., `api.github.com:443` → `api.github.com`)

## Expected Output

A GitHub discussion in the "audits" category containing a comprehensive daily firewall analysis report.
