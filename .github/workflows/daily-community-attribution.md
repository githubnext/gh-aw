---
name: Daily Community Attribution Updater
description: Maintains a live community contributions section in README.md and an all-time Community Contributors wiki page by attributing all community-labeled issues using the five-tier attribution strategy
on:
  schedule:
    - cron: daily
  workflow_dispatch:

permissions:
  contents: read
  pull-requests: read
  issues: read

engine:
  id: copilot
  model: claude-haiku-4.5
timeout-minutes: 30

network:
  allowed:
    - defaults

tools:
  mount-as-clis: true
  github:
    mode: "local"
    toolsets: [issues]
  repo-memory:
    wiki: true
    description: "All-time Community Contributors list"
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
  create-issue:
    title-prefix: "[community-attribution] "
    labels: [community, automation]
    close-older-issues: true
    group-by-day: true
    expires: 7d

imports:
  - shared/community-attribution.md
  - shared/observability-otlp.md
  - shared/issue-dedup.md

steps:
  - name: Fetch PR data for attribution index
    env:
      GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
      GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}
    run: |
      mkdir -p /tmp/gh-aw/agent/community-data

      # Fetch merged PRs from the last 90 days (wide enough to catch any recently attributed issue)
      SINCE=$(date -d '90 days ago' '+%Y-%m-%dT%H:%M:%SZ' 2>/dev/null \
              || date -v-90d '+%Y-%m-%dT%H:%M:%SZ')

      echo "Fetching PRs merged since $SINCE..."
      gh pr list \
        --state merged \
        --limit 500 \
        --json number,title,author,mergedAt,url,body,closingIssuesReferences \
        --jq "[.[] | select(.mergedAt >= \"$SINCE\")]" \
        > /tmp/gh-aw/agent/community-data/pull_requests.json \
        || echo "[]" > /tmp/gh-aw/agent/community-data/pull_requests.json

      PR_COUNT=$(jq length /tmp/gh-aw/agent/community-data/pull_requests.json)
      echo "✓ Fetched $PR_COUNT merged PRs"

      # Build closing references index: {issue_number: [pr_numbers]}
      # Use a nested reduce so the outer body always returns the accumulator,
      # even when closingIssuesReferences is empty (avoids jq setting acc to null).
      jq '
        reduce .[] as $pr (
          {};
          reduce ($pr.closingIssuesReferences // [])[] as $issue (
            .;
            ($issue.number | tostring) as $key |
            .[$key] = (.[$key] // []) + [$pr.number]
          )
        )
      ' /tmp/gh-aw/agent/community-data/pull_requests.json \
        > /tmp/gh-aw/agent/community-data/closing_refs_by_issue.json 2>/dev/null \
        || echo "{}" > /tmp/gh-aw/agent/community-data/closing_refs_by_issue.json

      LINK_COUNT=$(jq 'keys | length' /tmp/gh-aw/agent/community-data/closing_refs_by_issue.json)
      echo "✓ Built closing refs index: $LINK_COUNT issues with native GitHub close links"

      # Find community issues closed within the PR lookback window (attribution candidates)
      jq --arg since "$SINCE" \
        '[.[] | select(.closedAt != null and .closedAt >= $since)]' \
        /tmp/gh-aw/agent/community-data/community_issues.json \
        > /tmp/gh-aw/agent/community-data/community_issues_closed_in_window.json 2>/dev/null \
        || echo "[]" > /tmp/gh-aw/agent/community-data/community_issues_closed_in_window.json

      CLOSED_COUNT=$(jq length /tmp/gh-aw/agent/community-data/community_issues_closed_in_window.json)
      echo "✓ Found $CLOSED_COUNT community issues closed in the lookback window"

      echo ""
      echo "Data available in /tmp/gh-aw/agent/community-data/:"
      echo "  community_issues.json                  — all community-labeled issues (includes stateReason)"
      echo "  pull_requests.json                     — merged PRs (last 90 days)"
      echo "  closing_refs_by_issue.json             — native GitHub close links"
      echo "  community_issues_closed_in_window.json — closed during lookback"

  - name: Compute deterministic attributions (Tier 0, 1, 2)
    env:
      GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
      GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}
    run: |
      # Tier 0: COMPLETED issues (direct contributions, no PR needed)
      jq '[.[] | select(.stateReason == "COMPLETED") |
          . + {tier: 0, attribution_type: "direct issue", closing_prs: []}]' \
        /tmp/gh-aw/agent/community-data/community_issues.json \
        > /tmp/gh-aw/agent/community-data/tier0_attributed.json

      T0=$(jq length /tmp/gh-aw/agent/community-data/tier0_attributed.json)
      echo "Tier 0 (direct issue — COMPLETED): $T0"

      # Tier 1: native GitHub close references (exclude Tier 0 issues)
      jq --slurpfile issues /tmp/gh-aw/agent/community-data/community_issues.json \
         --slurpfile t0 /tmp/gh-aw/agent/community-data/tier0_attributed.json '
        ($t0[0] | map(.number) | map(tostring)) as $t0_keys |
        ($issues[0] | map(.number | tostring)) as $issue_keys |
        to_entries |
        map(select(
          .key as $k |
          ($issue_keys | index($k) != null) and
          ($t0_keys | index($k) == null)
        )) |
        map(.key as $k | .value as $prs |
          ($issues[0] | map(select(.number | tostring == $k))[0]) +
          {tier: 1, attribution_type: "resolved by PR", closing_prs: $prs}
        )
      ' /tmp/gh-aw/agent/community-data/closing_refs_by_issue.json \
        > /tmp/gh-aw/agent/community-data/tier1_attributed.json 2>/dev/null \
        || echo "[]" > /tmp/gh-aw/agent/community-data/tier1_attributed.json

      T1=$(jq length /tmp/gh-aw/agent/community-data/tier1_attributed.json)
      echo "Tier 1 (native close refs): $T1"

      # Tier 2: PR body keyword matching (exclude Tier 0 and Tier 1 issues)
      KW_ISSUES=$(jq -r '.[].body // ""' /tmp/gh-aw/agent/community-data/pull_requests.json \
        | grep -oP '(?i)(?:close[sd]?|fix(?:e[sd])?|resolve[sd]?)\s*(?:github/gh-aw#|#)\K[0-9]+' 2>/dev/null \
        | sort -u | jq -R 'tonumber' | jq -s 'sort | unique' 2>/dev/null \
        || echo "[]")

      jq --argjson kw "$KW_ISSUES" \
         --slurpfile t0 /tmp/gh-aw/agent/community-data/tier0_attributed.json \
         --slurpfile t1 /tmp/gh-aw/agent/community-data/tier1_attributed.json '
        ($t0[0] | map(.number)) as $t0_nums |
        ($t1[0] | map(.number)) as $t1_nums |
        [.[] |
          select(
            .number as $n |
            ($kw | index($n) != null) and
            ($t0_nums | index($n) == null) and
            ($t1_nums | index($n) == null)
          ) |
          . + {tier: 2, attribution_type: "resolved by PR", closing_prs: []}
        ]
      ' /tmp/gh-aw/agent/community-data/community_issues.json \
        > /tmp/gh-aw/agent/community-data/tier2_attributed.json 2>/dev/null \
        || echo "[]" > /tmp/gh-aw/agent/community-data/tier2_attributed.json

      T2=$(jq length /tmp/gh-aw/agent/community-data/tier2_attributed.json)
      echo "Tier 2 (PR body keywords): $T2"

      # Combine Tier 0 + 1 + 2 into pre_attributed.json
      jq -n \
        --slurpfile t0 /tmp/gh-aw/agent/community-data/tier0_attributed.json \
        --slurpfile t1 /tmp/gh-aw/agent/community-data/tier1_attributed.json \
        --slurpfile t2 /tmp/gh-aw/agent/community-data/tier2_attributed.json \
        '$t0[0] + $t1[0] + $t2[0]' \
        > /tmp/gh-aw/agent/community-data/pre_attributed.json

      TOTAL=$(jq length /tmp/gh-aw/agent/community-data/pre_attributed.json)
      echo ""
      echo "Pre-attributed: $TOTAL issues (Tier 0: $T0, Tier 1: $T1, Tier 2: $T2)"

      # Compute Tier 3+ candidates (closed in window, not yet pre-attributed)
      jq --slurpfile pre /tmp/gh-aw/agent/community-data/pre_attributed.json '
        ($pre[0] | map(.number)) as $attributed |
        [.[] | select(.number as $n | $attributed | index($n) == null)]
      ' /tmp/gh-aw/agent/community-data/community_issues_closed_in_window.json \
        > /tmp/gh-aw/agent/community-data/tier3_candidates.json 2>/dev/null \
        || echo "[]" > /tmp/gh-aw/agent/community-data/tier3_candidates.json

      T3=$(jq length /tmp/gh-aw/agent/community-data/tier3_candidates.json)
      echo "Tier 3+ candidates (agent lookup needed): $T3"
      echo ""
      echo "Data available in /tmp/gh-aw/agent/community-data/:"
      echo "  pre_attributed.json    — Tier 0+1+2 confirmed attributions"
      echo "  tier3_candidates.json  — issues needing Tier 3 agent lookup"

features:
  mcp-cli: true
---

# Daily Community Attribution Updater

Maintain an up-to-date **🌍 Community Contributions** section in `README.md`
and an all-time **Community Contributors** wiki page by attributing all
resolved community-labeled issues using the five-tier attribution strategy.

## Mission

The `community` label is the **primary attribution signal**: every issue
tagged with it was explicitly identified by a maintainer as community-authored.
This workflow attributes those issues (including direct-issue contributions
with `stateReason == "COMPLETED"`), updates `README.md`, maintains the wiki,
and opens a PR for review.

## Pre-fetched Data

All data is in `/tmp/gh-aw/agent/community-data/`:

```bash
# Tier 0+1+2 are already computed — start here:
cat /tmp/gh-aw/agent/community-data/pre_attributed.json | \
  jq -r '.[] | "- #\(.number) [Tier \(.tier)] \(.attribution_type) — \(.title) by @\(.author.login)"'

# Issues still needing Tier 3 agent lookup:
cat /tmp/gh-aw/agent/community-data/tier3_candidates.json | \
  jq -r '.[] | "- #\(.number): \(.title) by @\(.author.login) (closed: \(.closedAt), stateReason: \(.stateReason // "null"))"'

# View closing reference index
cat /tmp/gh-aw/agent/community-data/closing_refs_by_issue.json | jq

# View current README
head -80 /tmp/gh-aw/agent/community-data/README_current.md

# View existing wiki page (if any)
cat /tmp/gh-aw/repo-memory-default/Community-Contributors.md 2>/dev/null || echo "(wiki page does not exist yet)"
```

## Workflow

### 1. Attribute All Resolved Community Issues

**Tier 0, 1, and 2 attributions are already pre-computed** in
`pre_attributed.json` — do not re-derive them. Read this file directly
and use its contents as the confirmed attribution list.

For each issue in `tier3_candidates.json`, apply **Tier 3** from the
imported Community Attribution Strategy (GitHub MCP `issue_read` to
look for indirect linkage via follow-up or split issues).

Any candidate still unresolved after Tier 3 becomes a **Tier 4**
"needs review" item.

### 2. Update the Community Contributors Wiki Page

Read the existing wiki page at
`/tmp/gh-aw/repo-memory-default/Community-Contributors.md` (empty/missing on
first run).  Merge all confirmed attributions — both newly found ones and all
previously recorded ones — without duplicating entries.

> **Wiki page size limit**: Keep `Community-Contributors.md` under **9 KB**
> (hard limit is 10 KB). Check the byte size with `wc -c` before calling
> `push_repo_memory`. If the page exceeds 9 KB, remove entries to reduce it:
> sort all authors by total contribution count (descending), then remove the
> oldest entry (lowest issue number) from the author with the most entries,
> and repeat until the page is under 9 KB.

The wiki page uses issue numbers as link text for quick scanning, while `README.md`
uses issue titles. Both use full GitHub issue URLs.

The wiki page format:

```markdown
# Community Contributors

### @author

- [#N](https://github.com/OWNER/REPO/issues/N) Issue title — YYYY-MM-DD — direct issue
- [#N](https://github.com/OWNER/REPO/issues/N) Issue title — YYYY-MM-DD — resolved by #PR

### @author2

- [#N](https://github.com/OWNER/REPO/issues/N) Issue title — YYYY-MM-DD — direct issue
```

- Group entries by author (alphabetical order)
- Within each author section, sort by issue number descending (newest first)
- **`direct issue`** — Tier 0: closed as `COMPLETED`, no PR linkage
- **`resolved by #PR`** — Tiers 1–3: attributed to a specific merged PR
- Do not add entries for unresolved or ambiguous candidates (Tier 4)

Write the updated content back to
`/tmp/gh-aw/repo-memory-default/Community-Contributors.md` using the edit tool.

### 3. Build the Community Contributions Section

Produce a compact section of attributed community contributors for
`README.md`, wrapped in a `<details>` element. Use **one list item per
author** with all their issues listed inline. Use GitHub issue references
(`#N`) so that GitHub automatically expands them with the issue title —
do **not** use full URLs or explicit issue titles as link text (GitHub
renders the title for you):

```markdown
## 🌍 Community Contributions

<details>
<summary>Thank you to the community members whose issue reports were resolved in this project! This list is updated automatically and reflects all attributed contributions.</summary>

- @author: #N _(direct issue)_, #N, #N _(via follow-up #M)_
- @author2: #N, #N

</details>

```

**Important**: always leave a blank line after `</details>` (as shown
above) so that the next markdown header renders correctly.

- One bullet per author, sorted alphabetically by username
- Within each author's entry, list issues in descending order (newest first), comma-separated
- **`_(direct issue)_`** (Tier 0): issue closed as `COMPLETED`, no PR linkage
- _(no suffix)_ (Tier 1/2): PR closes the issue via native close reference or keyword
- **`_(via follow-up #M)_`** (Tier 3): indirect chain through a follow-up issue
- Omit issues that cannot be attributed (see Attribution Candidates section below)

If there are unattributed candidates (Tier 4), append after the `</details>` blank line:

```markdown
### ⚠️ Attribution Candidates Need Review

The following community issues were closed but could not be automatically
linked to a specific merged PR. Please verify whether they should be credited:

- **@author** for [Issue title](#N) — closed DATE
```

### 4. Update README.md

Replace the existing `## 🌍 Community Contributions` section in `README.md`
with the newly generated content, or append it after the `## Contributing`
section if it does not yet exist.

Use the edit tool to make the change in-place.

### 5. Open a Pull Request

If `README.md` **or** the wiki page changed, call the `create_pull_request`
safe-output tool to open a PR with the changes.

**PR title**: `[community] Update community contributions in README`

**PR body template**:
```markdown
### Community Contributions Update

Automated update to the 🌍 Community Contributions section in `README.md`
and the Community Contributors wiki page.

#### Changes
- N community issues newly attributed
- N attribution candidates flagged for review (if any)
- Wiki page updated: Y/N

#### Attribution Summary
[brief summary of what changed and how each was attributed]
```

**Important**: If no action is needed after completing your analysis, you
**MUST** call the `noop` safe-output tool with a brief explanation.

```json
{"noop": {"message": "No action needed: [brief explanation]"}}
```

### 6. Report Failures

If you encounter a genuine error that prevents completion (e.g., data fetch failure, unexpected error), report it using the `create_issue` safe-output tool — **never use the GitHub MCP `create_issue` tool directly**. The safe-output tool has built-in deduplication (`group-by-day` and `close-older-issues`) that prevents duplicate failure issues from accumulating.

```json
{"create_issue": {"title": "Brief description of the failure", "body": "### What failed\n\nDescribe the specific step or data source that failed.\n\n### Error details\n\n(Include the error message or unexpected output here)\n\n### Steps to investigate\n\n1. Check the workflow run logs for the full error message\n2. Verify that community_issues.json and pull_requests.json were fetched successfully\n3. Re-run the workflow manually via workflow_dispatch to see if the failure is transient"}}
```
