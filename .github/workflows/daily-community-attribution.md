---
name: Daily Community Attribution Updater
description: Maintains a live community contributions section in README.md by attributing all community-labeled issues to merged PRs using the four-tier attribution strategy
on:
  schedule:
    - cron: daily
  workflow_dispatch:

permissions:
  contents: read
  pull-requests: read
  issues: read

engine: copilot
timeout-minutes: 30

network:
  allowed:
    - defaults

tools:
  github:
    mode: "local"
    toolsets: [issues, pull_requests]
  bash:
    - "gh pr list *"
    - "gh issue list *"
    - "jq *"
    - "mkdir *"
    - "echo *"
    - "cp *"
    - "cat *"
    - "date *"
  edit:

safe-outputs:
  create-pull-request:
    expires: 1d
    title-prefix: "[community] "
    labels: [community, automation]
    reviewers: []
    draft: true

imports:
  - shared/community-attribution.md

steps:
  - name: Fetch PR data for attribution index
    env:
      GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
      GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}
    run: |
      mkdir -p /tmp/gh-aw/community-data

      # Fetch merged PRs from the last 90 days (wide enough to catch any recently attributed issue)
      SINCE=$(date -d '90 days ago' '+%Y-%m-%dT%H:%M:%SZ' 2>/dev/null \
              || date -v-90d '+%Y-%m-%dT%H:%M:%SZ')

      echo "Fetching PRs merged since $SINCE..."
      gh pr list \
        --state merged \
        --limit 500 \
        --json number,title,author,mergedAt,url,body,closingIssuesReferences \
        --jq "[.[] | select(.mergedAt >= \"$SINCE\")]" \
        > /tmp/gh-aw/community-data/pull_requests.json \
        || echo "[]" > /tmp/gh-aw/community-data/pull_requests.json

      PR_COUNT=$(jq length /tmp/gh-aw/community-data/pull_requests.json)
      echo "✓ Fetched $PR_COUNT merged PRs"

      # Build closing references index: {issue_number: [pr_numbers]}
      jq '
        reduce .[] as $pr (
          {};
          $pr.closingIssuesReferences[]? as $issue |
          ($issue.number | tostring) as $key |
          .[$key] = (.[$key] // []) + [$pr.number]
        )
      ' /tmp/gh-aw/community-data/pull_requests.json \
        > /tmp/gh-aw/community-data/closing_refs_by_issue.json 2>/dev/null \
        || echo "{}" > /tmp/gh-aw/community-data/closing_refs_by_issue.json

      LINK_COUNT=$(jq 'keys | length' /tmp/gh-aw/community-data/closing_refs_by_issue.json)
      echo "✓ Built closing refs index: $LINK_COUNT issues with native GitHub close links"

      # Find community issues closed within the PR lookback window (attribution candidates)
      jq --arg since "$SINCE" \
        '[.[] | select(.closedAt != null and .closedAt >= $since)]' \
        /tmp/gh-aw/community-data/community_issues.json \
        > /tmp/gh-aw/community-data/community_issues_closed_in_window.json 2>/dev/null \
        || echo "[]" > /tmp/gh-aw/community-data/community_issues_closed_in_window.json

      CLOSED_COUNT=$(jq length /tmp/gh-aw/community-data/community_issues_closed_in_window.json)
      echo "✓ Found $CLOSED_COUNT community issues closed in the lookback window"

      echo ""
      echo "Data available in /tmp/gh-aw/community-data/:"
      echo "  community_issues.json                  — all community-labeled issues"
      echo "  pull_requests.json                     — merged PRs (last 90 days)"
      echo "  closing_refs_by_issue.json             — native GitHub close links"
      echo "  community_issues_closed_in_window.json — closed during lookback"
---

# Daily Community Attribution Updater

Maintain an up-to-date **🌍 Community Contributions** section in `README.md`
by attributing all resolved community-labeled issues to merged PRs.

## Mission

The `community` label is the **primary attribution signal**: every issue
tagged with it was explicitly identified by a maintainer as community-authored.
This workflow attributes those issues to PRs, updates `README.md`, and opens
a PR for review.

## Pre-fetched Data

All data is in `/tmp/gh-aw/community-data/`:

```bash
# View all community-labeled issues
cat /tmp/gh-aw/community-data/community_issues.json | \
  jq -r '.[] | "- #\(.number) [\(.state // "?")] \(.title) by @\(.author.login)"'

# View recently closed community issues (attribution candidates)
cat /tmp/gh-aw/community-data/community_issues_closed_in_window.json | \
  jq -r '.[] | "- #\(.number): \(.title) by @\(.author.login) (closed: \(.closedAt))"'

# View closing reference index
cat /tmp/gh-aw/community-data/closing_refs_by_issue.json | jq

# View current README
head -80 /tmp/gh-aw/community-data/README_current.md
```

## Workflow

### 1. Attribute All Resolved Community Issues

Apply the **Community Attribution Strategy** from the imported shared component
to every closed community-labeled issue.

Focus on issues in `community_issues_closed_in_window.json` first (recently
closed, most likely to be new). Then check older closed issues in
`community_issues.json` that are not already reflected in `README.md`.

For each community issue, work through all four attribution tiers (see shared
component) and mark it as **confirmed**, **confirmed (via follow-up)**, or
**needs review**.

### 2. Build the Community Contributions Table

Produce a concise, sorted table of attributed community contributors:

```markdown
## 🌍 Community Contributions

Thank you to the community members whose issue reports were resolved in this project!
This list is updated automatically and reflects all attributed contributions.

| Issue | Title | Author | Resolved By | Attribution |
|-------|-------|--------|-------------|-------------|
| [#N](url) | Issue title | @author | [#PR](url) | direct |
| [#N](url) | Issue title | @author | [#PR](url) via [#M](url) | follow-up |
```

- Sort by issue number descending (newest first)
- `Attribution` column: `direct` for Tier 1/2, `via follow-up #M` for Tier 3
- Omit issues that cannot be attributed (see Attribution Candidates section below)

If there are unattributed candidates (Tier 4), append:

```markdown
### ⚠️ Attribution Candidates Need Review

The following community issues were closed but could not be automatically
linked to a specific merged PR. Please verify whether they should be credited:

- **@author** for [Issue title](#N) — closed DATE
```

### 3. Update README.md

Replace the existing `## 🌍 Community Contributions` section in `README.md`
with the newly generated content, or append it after the `## Contributing`
section if it does not yet exist.

Use the edit tool to make the change in-place.

If no changes are needed (all attributions already present and current),
call the `noop` safe-output tool and stop.

### 4. Open a Pull Request

If `README.md` was updated, call the `create_pull_request` safe-output tool
to open a PR with the changes.

**PR title**: `[community] Update community contributions in README`

**PR body template**:
```markdown
### Community Contributions Update

Automated update to the 🌍 Community Contributions section in `README.md`.

#### Changes
- N community issues newly attributed
- N attribution candidates flagged for review (if any)

#### Attribution Summary
[brief summary of what changed and how each was attributed]
```

**Important**: If no action is needed after completing your analysis, you
**MUST** call the `noop` safe-output tool with a brief explanation.

```json
{"noop": {"message": "No action needed: [brief explanation]"}}
```
