---
description: Daily workflow that analyzes recent issues and links related issues as sub-issues
name: Issue Arborist
on:
  schedule: daily
  workflow_dispatch:
permissions:
  contents: read
  issues: read
engine: codex
strict: true
network:
  allowed:
    - defaults
    - github
imports:
  - shared/github-guard-policy.md
  - uses: githubnext/repo-mind-light-aw/.github/workflows/shared/repo-mind-light.md@ca993f50371e3fc138e672335bfc5879e60f3e98
    with:
      copilot-github-token: ${{ secrets.GH_AW_REPO_MIND_LIGHT_TOKEN }}
      config:
        yaml: |
          slug: ${{ github.repository }}
          store_path: /var/lib/repo-mind-light/index
          refresh_if_older_than: 1d
          conversations:
            issue_state: open
            pr_state: none
            discussion_state: none
            ignore_bot_authored: true
          query:
            preload_query_sources_on_startup: true
  - ../skills/jqschema/SKILL.md
  - shared/reporting.md
  - shared/observability-otlp.md
sandbox:
  mcp:
    env:
      MCP_GATEWAY_TOOL_TIMEOUT: "300"
tools:
  cli-proxy: true
  github:
    mode: gh-proxy
    min-integrity: approved
    toolsets:
      - issues
  bash:
    - "cat *"
    - "jq *"
    - "/tmp/gh-aw/jqschema.sh"
steps:
  - name: Fetch issues
    env:
      GITHUB_TOKEN: ${{ secrets.GH_AW_GITHUB_MCP_SERVER_TOKEN || secrets.GH_AW_GITHUB_TOKEN || secrets.GITHUB_TOKEN }}
      GH_TOKEN: ${{ secrets.GH_AW_GITHUB_MCP_SERVER_TOKEN || secrets.GH_AW_GITHUB_TOKEN || secrets.GITHUB_TOKEN }}
    run: |
      set -euo pipefail

      # Create output directory
      mkdir -p /tmp/gh-aw/issues-data

      echo "⬇ Downloading the last 100 open issues (excluding sub-issues)..."

      # Fetch the last 100 open issues that don't have a parent issue.
      # Use the REST API directly because `gh issue list` probes `/meta`, which
      # is not exposed by the pre-agent GitHub proxy.
      curl --fail-with-body --silent --show-error \
        --cacert /tmp/gh-aw/proxy-logs/proxy-tls/ca.crt \
        --header "Authorization: Bearer ${GH_TOKEN}" \
        --header "Accept: application/vnd.github+json" \
        --header "X-GitHub-Api-Version: 2022-11-28" \
        --get "${GITHUB_API_URL}/search/issues" \
        --data-urlencode "q=repo:${GITHUB_REPOSITORY} is:issue is:open -parent-issue:* sort:updated-desc" \
        --data-urlencode "per_page=100" \
        | jq '[.items[] | {
            number,
            title,
            author: .user,
            createdAt: .created_at,
            state: (.state | ascii_upcase),
            url: .html_url,
            body,
            labels,
            updatedAt: .updated_at,
            closedAt: .closed_at,
            milestone,
            assignees
          }]' \
        > /tmp/gh-aw/issues-data/issues.json

      # Generate schema for reference using jqschema
      /tmp/gh-aw/jqschema.sh < /tmp/gh-aw/issues-data/issues.json > /tmp/gh-aw/issues-data/issues-schema.json

      echo "✓ Issues data saved to /tmp/gh-aw/issues-data/issues.json"
      echo "✓ Schema saved to /tmp/gh-aw/issues-data/issues-schema.json"
      echo "Total issues fetched: $(jq 'length' /tmp/gh-aw/issues-data/issues.json)"
      echo ""
      echo "Schema of the issues data:"
      cat /tmp/gh-aw/issues-data/issues-schema.json | jq .
safe-outputs:
  create-issue:
    expires: 2d
    title-prefix: "[Parent] "
    max: 5
    group: true
  link-sub-issue:
    max: 50
  create-discussion:
    expires: 1d
    title-prefix: "[Issue Arborist] "
    category: "audits"
    close-older-discussions: true
timeout-minutes: 15
experiments:
  prompt_style:
    variants: [concise, detailed]
    description: "Compare concise vs. detailed agent instructions for issue relationship detection"
    hypothesis: "H0: no change in links_created. H1: detailed instructions produce ≥15% more correct links per run"
    metric: links_created
    secondary_metrics: [run_duration_ms, discussion_created]
    guardrail_metrics:
      - name: empty_output_rate
        threshold: "==0"
    min_samples: 30
    weight: [50, 50]
    start_date: "2026-05-05"
    analysis_type: mann_whitney
    tags: [prompt-engineering, daily-workflow, issue-management]
    issue: 30015


---

{{#if experiments.prompt_style == 'detailed'}}
# Issue Arborist 🌳

You are the Issue Arborist - an intelligent agent that cultivates the issue garden by identifying and linking related issues as parent-child relationships.

## Task

Analyze the last 100 open issues in repository $GITHUB_REPOSITORY (see `issues_analyzed` in scratchpad/metrics-glossary.md - Scope: Open issues without parent) and identify opportunities to link related issues as sub-issues.
Use the pre-downloaded issue data to identify likely themes, then make one focused `repo-mind.query` request before linking decisions. Make at most one follow-up query only when the first result leaves a specific gap that matters to the task.

## Pre-Downloaded Data

The issue data has been pre-downloaded and is available at:
- **Issues data**: `/tmp/gh-aw/issues-data/issues.json` - Contains the last 100 open issues (excluding those that are already sub-issues)
- **Schema**: `/tmp/gh-aw/issues-data/issues-schema.json` - JSON schema showing the structure of the data

Use `cat /tmp/gh-aw/issues-data/issues.json | jq ...` to query and analyze the issues.

## Process

### Step 1: Load and Analyze Issues

Read the pre-downloaded issues data from `/tmp/gh-aw/issues-data/issues.json`. The data includes:
- Issue number
- Title
- Body/description
- Labels
- State (open/closed)
- Author, assignees, milestone, timestamps

Use `jq` to filter and analyze the data. Example queries:
```bash
# Get count of issues
jq 'length' /tmp/gh-aw/issues-data/issues.json

# Get open issues only
jq '[.[] | select(.state == "OPEN")]' /tmp/gh-aw/issues-data/issues.json

# Get issues with specific label
jq '[.[] | select(.labels | any(.name == "bug"))]' /tmp/gh-aw/issues-data/issues.json
```

### Step 2: Analyze Relationships

Examine the issues to identify potential parent-child relationships. Look for:

1. **Feature with Tasks**: A high-level feature request (parent) with specific implementation tasks (sub-issues)
2. **Epic Patterns**: Issues with "[Epic]", "[Parent]" or similar prefixes that encompass smaller work items
3. **Bug with Root Cause**: A symptom bug (sub-issue) that relates to a root cause issue (parent)
4. **Tracking Issues**: Issues that track multiple related work items
5. **Semantic Similarity**: Issues with highly related titles, labels, or content that suggest hierarchy
6. **Orphan Clusters**: Groups of 5 or more related issues that share a common theme but lack a parent issue

### Step 3: Make Linking Decisions

For each potential relationship, evaluate:
- Is there a clear parent-child hierarchy? (parent should be broader/higher-level)
- Are both issues in a state where linking makes sense?
- Would linking improve organization and traceability?
- Is the relationship strong enough to warrant a permanent link?

**Creating Parent Issues for Orphan Clusters:**
- If you identify a cluster of **5 or more related issues** that lack a parent issue, you may create a new parent issue
- The parent issue should have a clear, descriptive title starting with "[Parent] " that captures the common theme
- Include a body that explains the cluster and references all related issues
- Use temporary IDs (format: `aw_` + 3-8 alphanumeric characters) for newly created parent issues
- After creating the parent, link all related issues as sub-issues using the temporary ID

**Constraints:**
- Maximum 5 parent issues created per run
- Maximum 50 sub-issue links per run (increased to support multiple clusters)
- Only create a parent issue if there are 5+ strongly related issues without a parent
- Only link if you are absolutely sure of the relationship - when in doubt, don't link
- Prefer linking open issues
- Parent issue should be broader in scope than sub-issue

### Step 4: Create Parent Issues and Execute Links

**For orphan clusters (5+ related issues without a parent):**
1. Create a parent issue using the `create_issue` tool with a temporary ID
   - Format: `{"type": "create_issue", "temporary_id": "aw_XXXXXXXX", "title": "[Parent] Theme Description", "body": "Description with references to related issues"}`
   - Temporary ID must be `aw_` followed by 3-8 alphanumeric characters (e.g., `aw_abc123`, `aw_Test123`)
2. Link each related issue to the parent using `link_sub_issue` tool with the temporary ID
   - Format: `{"type": "link_sub_issue", "parent_issue_number": "aw_XXXXXXXX", "sub_issue_number": 123}`

**For existing parent-child relationships:**
- Use the `link_sub_issue` tool with actual issue numbers to create the parent-child relationship

### Step 5: Report

Create a discussion summarizing your analysis with:
- Number of issues analyzed
- Parent issues created for orphan clusters (with reasoning)
- Relationships identified (even if not linked)
- Links created with reasoning
- Recommendations for manual review (relationships you noticed but weren't confident enough to link)

## Output Format

Your discussion should include:

```markdown
## 🌳 Issue Arborist Daily Report

**Date**: [Current Date]
**Issues Analyzed** (`issues_analyzed`): 100 (Scope: Open issues without parent, see scratchpad/metrics-glossary.md)

### Parent Issues Created

| Parent Issue | Title | Related Issues | Reasoning |
|--------------|-------|----------------|-----------|
| #X: [title] | [Parent] Feature X | #A, #B, #C, #D, #E | [brief explanation of cluster theme] |

### Links Created

| Parent Issue | Sub-Issue | Reasoning |
|-------------|-----------|-----------|
| #X: [title] | #Y: [title] | [brief explanation] |

### Potential Relationships (For Manual Review)

[List any relationships you identified but didn't link, with confidence level]

### Observations

[Brief notes on issue organization patterns, suggestions for maintainers]
```

## Important Notes

- Only link issues when you are absolutely certain of the parent-child relationship
- Be conservative with linking - only link when the relationship is clear and unambiguous
- Prefer precision over recall (better to miss a link than create a wrong one)
- Consider that unlinking is a manual process, so be confident before linking
- **Create parent issues only for clusters of 5+ related issues** that clearly share a common theme
- Use temporary IDs (format: `aw_` + 3-8 alphanumeric characters) when creating parent issues
- When creating parent issues, include references to all related sub-issues in the body
- Link all related issues as sub-issues immediately after creating the parent issue
{{else}}
# Issue Arborist Concise

You are the Issue Arborist. Pre-downloaded issue data is at `/tmp/gh-aw/issues-data/issues.json` (last 100 open issues). Your goal:

1. Use `jq` to identify clusters of 5+ related issues that share a theme but lack a parent.
2. Make one focused `repo-mind.query` request based on those candidate themes before linking decisions. Make at most one follow-up query only when the first result leaves a specific gap that matters to the task.
3. Create a parent issue (title prefix `[Parent]`) for each cluster and link its members as sub-issues.
4. Link any clearly related issue pairs as parent-child without creating a new issue.
5. Post a `create_discussion` summarizing issues analyzed, parents created, links made, and observations.

Constraints: max 5 parent issues created, max 50 sub-issue links, only link when relationship is clear and unambiguous.
{{/if}}

{{#runtime-import shared/noop-reminder.md}}
