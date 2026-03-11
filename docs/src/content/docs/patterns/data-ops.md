---
title: DataOps
description: Deterministic data extraction in steps, followed by agentic analysis and reporting
sidebar:
  badge: { text: 'Hybrid', variant: 'caution' }
---

DataOps combines deterministic data extraction with agentic analysis: shell commands in `steps:` reliably collect and prepare data (fast, cacheable, reproducible), then the AI agent reads the results and generates insights. Use this pattern for data aggregation, report generation, trend analysis, and auditing.

## The DataOps Pattern

### Basic Structure

```aw wrap
---
on:
  schedule: daily
  workflow_dispatch:

steps:
  - name: Collect data
    run: |
      # Deterministic data extraction
      gh api ... > /tmp/gh-aw/data.json

safe-outputs:
  create-discussion:
    category: "reports"
---

# Analysis Workflow

Analyze the data at `/tmp/gh-aw/data.json` and create a summary report.
```

## Example: PR Activity Summary

This workflow collects statistics from recent pull requests and generates a weekly summary:

````aw wrap
---
name: Weekly PR Summary
description: Summarizes pull request activity from the last week
on:
  schedule: weekly
  workflow_dispatch:

permissions:
  contents: read
  pull-requests: read

engine: copilot
strict: true

network:
  allowed:
    - defaults
    - github

safe-outputs:
  create-discussion:
    title-prefix: "[weekly-summary] "
    category: "announcements"
    max: 1
    close-older-discussions: true

tools:
  bash: ["*"]

steps:
  - name: Fetch recent pull requests
    env:
      GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}
    run: |
      mkdir -p /tmp/gh-aw/pr-data

      # Fetch last 100 PRs with key metadata
      gh pr list \
        --repo "${{ github.repository }}" \
        --state all \
        --limit 100 \
        --json number,title,state,author,createdAt,mergedAt,closedAt,additions,deletions,changedFiles,labels \
        > /tmp/gh-aw/pr-data/recent-prs.json

      echo "Fetched $(jq 'length' /tmp/gh-aw/pr-data/recent-prs.json) PRs"

  - name: Compute summary statistics
    run: |
      cd /tmp/gh-aw/pr-data

      # Generate statistics summary
      jq '{
        total: length,
        merged: [.[] | select(.state == "MERGED")] | length,
        open: [.[] | select(.state == "OPEN")] | length,
        closed: [.[] | select(.state == "CLOSED")] | length,
        total_additions: [.[].additions] | add,
        total_deletions: [.[].deletions] | add,
        total_files_changed: [.[].changedFiles] | add,
        authors: [.[].author.login] | unique | length,
        top_authors: ([.[].author.login] | group_by(.) | map({author: .[0], count: length}) | sort_by(-.count) | .[0:5])
      }' recent-prs.json > stats.json

      echo "Statistics computed:"
      cat stats.json

timeout-minutes: 10
---

# Weekly Pull Request Summary

Analyze the prepared data:
- `/tmp/gh-aw/pr-data/recent-prs.json` - Last 100 PRs with full metadata
- `/tmp/gh-aw/pr-data/stats.json` - Pre-computed statistics

Create a discussion summarizing: total PRs, merge rate, code changes (+/- lines), top contributors, and any notable trends. Keep it concise and factual.
````

## Data Caching

For workflows that run frequently or process large datasets, use caching to avoid redundant API calls:

```aw wrap
---
cache:
  - key: pr-data-${{ github.run_id }}
    path: /tmp/gh-aw/pr-data
    restore-keys: |
      pr-data-

steps:
  - name: Check cache and fetch only new data
    run: |
      if [ -f /tmp/gh-aw/pr-data/recent-prs.json ]; then
        echo "Using cached data"
      else
        gh pr list --limit 100 --json ... > /tmp/gh-aw/pr-data/recent-prs.json
      fi
---
```

## Advanced: Multi-Source Data

Combine data from multiple sources before analysis:

```aw wrap
---
steps:
  - name: Fetch PR data
    run: gh pr list --json ... > /tmp/gh-aw/prs.json

  - name: Fetch issue data
    run: gh issue list --json ... > /tmp/gh-aw/issues.json

  - name: Fetch workflow runs
    run: gh run list --json ... > /tmp/gh-aw/runs.json

  - name: Combine into unified dataset
    run: |
      jq -s '{prs: .[0], issues: .[1], runs: .[2]}' \
        /tmp/gh-aw/prs.json \
        /tmp/gh-aw/issues.json \
        /tmp/gh-aw/runs.json \
        > /tmp/gh-aw/combined.json
---

# Repository Health Report

Analyze the combined data at `/tmp/gh-aw/combined.json` covering:
- Pull request velocity and review times
- Issue response rates and resolution times
- CI/CD success rates and flaky tests
```

## Best Practices

- **Keep steps deterministic** - Same inputs should produce the same outputs; avoid randomness or time-dependent logic.
- **Pre-compute aggregations** - Use `jq`, `awk`, or Python to compute statistics upfront, reducing agent token usage.
- **Structure data clearly** - Output JSON with clear field names; include a summary file alongside raw data.
- **Document data locations** - Tell the agent where to find the data and what format to expect.
- **Use safe outputs** - Discussions are ideal for reports (support threading and reactions).

## Additional Resources

- [Steps Reference](/gh-aw/reference/frontmatter/#custom-steps-steps) - Shell step configuration
- [Safe Outputs Reference](/gh-aw/reference/safe-outputs/) - Validated GitHub operations
- [Cache Memory](/gh-aw/reference/cache-memory/) - Caching data between runs
- [DailyOps](/gh-aw/patterns/daily-ops/) - Scheduled improvement workflows
