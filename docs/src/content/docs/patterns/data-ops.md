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

## Subagents with Smaller Models

After moving computation into `steps:`, the next optimization is to delegate narrow, repetitive reasoning tasks—categorization, per-item summarization, sentiment scoring—to **inline sub-agents** backed by a smaller, cheaper model. The main agent then only needs to read the pre-processed results and synthesise a final output, spending its reasoning budget where it matters most.

### How the layers fit together

```
steps:          → deterministic shell commands (fast, reproducible, zero AI cost)
sub-agents:     → small-model agents for per-item analysis  (cheap, parallelizable)
main agent:     → orchestrates sub-agents, synthesizes final report (high-reasoning)
```

### Enabling inline sub-agents

Add the `inline-agents` feature flag and the `cli-proxy` tool so sub-agents can make authenticated GitHub API calls:

```yaml
features:
  inline-agents: true
tools:
  cli-proxy: true
```

### Example: Issue Triage with Categorization and Summarization

After `steps:` have fetched and split the raw data into per-item files, the prompt
orchestrates two small-model sub-agents and then synthesizes the results:

```aw wrap
# Weekly Issue Triage

The raw issue data is in `/tmp/gh-aw/triage/` — one file per issue (`issue-<number>.json`).

## Step 1 — categorize each issue

For every file matching `/tmp/gh-aw/triage/issue-*.json`, use the `issue-categorizer`
agent to classify it. Write the result to `/tmp/gh-aw/triage/category-<number>.json`.

## Step 2 — summarize each issue

For every issue file, use the `issue-summarizer` agent to produce a one-sentence
summary. Write the result to `/tmp/gh-aw/triage/summary-<number>.json`.

## Step 3 — synthesize triage report

Read all category and summary files, then create a discussion that groups issues
by category, lists each with its one-sentence summary and a link to the issue,
and highlights the top 3 issues that need the most urgent attention.

## agent: `issue-categorizer`
---
description: Classifies a GitHub issue into exactly one category
model: claude-haiku-4.5
---
You receive a JSON file for a single GitHub issue.
Classify the issue into exactly one of: bug, feature-request, question, documentation, performance, security, or other.
Return a JSON object: `{"number": <issue number>, "category": "<category>"}`.
Write nothing else.

## agent: `issue-summarizer`
---
description: Produces a one-sentence summary of a GitHub issue
model: claude-haiku-4.5
---
You receive a JSON file for a single GitHub issue.
Write a single sentence (≤ 20 words) that describes what the issue is about.
Return a JSON object: `{"number": <issue number>, "summary": "<sentence>"}`.
Write nothing else.
```

### Why this is faster and cheaper

| Layer | Model | Work done | Cost driver |
|---|---|---|---|
| `steps:` | — | Fetch + prepare data | GitHub API only |
| `issue-categorizer` | Haiku / small | Classify one issue | ~200 tokens per issue |
| `issue-summarizer` | Haiku / small | Summarize one issue | ~150 tokens per issue |
| Main agent | Full model | Read all results, write report | One high-quality pass |

For 50 issues the sub-agents consume roughly 50 × (200 + 150) = **17,500 tokens at haiku pricing**, while the main agent only sees the compact category/summary JSON—far less than reading raw issue bodies.

### Best practices for DataOps + subagents

- **One file per item** – write per-item JSON files in `steps:` so sub-agents can be directed to individual files without needing to slice arrays themselves.
- **Return structured JSON** – instruct sub-agents to return a compact JSON object; parsing prose costs extra tokens downstream.
- **Choose the right model** – classification and summarization rarely need chain-of-thought reasoning; a haiku-size model is sufficient and 10–20× cheaper.
- **Keep sub-agent prompts short** – sub-agent system prompts are loaded per invocation; shorter prompts reduce overhead for high-volume loops.
- **Let the main agent synthesize** – reserve the full-size model for the step that requires judgment: ranking, prioritization, generating actionable recommendations.

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
- [Inline Sub-Agents](/gh-aw/reference/inline-sub-agents/) - Defining specialized agents inside a workflow
