---
# Fetches all issues carrying the "community" label. This label is the
# primary signal that a report was authored by a community member; every
# issue bearing it should be considered for attribution.
tools:
  bash:
    - "gh issue list *"
    - "jq *"
    - "mkdir *"
    - "echo *"

steps:
  - name: Fetch community-labeled issues
    env:
      GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
      GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}
    run: |
      mkdir -p /tmp/gh-aw/community-data

      # The "community" label is the **primary attribution signal**: a maintainer
      # explicitly tagged the issue as community-authored, making it a strong and
      # intentional marker that does not rely on free-text heuristics.
      echo "Fetching issues with 'community' label (primary attribution signal)..."
      if ! gh issue list \
        --label "community" \
        --state all \
        --limit 500 \
        --json number,title,author,labels,closedAt,createdAt,url,stateReason \
        > /tmp/gh-aw/community-data/community_issues.json; then
        echo "[]" > /tmp/gh-aw/community-data/community_issues.json
      fi

      COMMUNITY_COUNT=$(jq length "/tmp/gh-aw/community-data/community_issues.json")
      echo "✓ Fetched $COMMUNITY_COUNT community-labeled issues"
      echo "  Data: /tmp/gh-aw/community-data/community_issues.json"
---

## Community Attribution Strategy

The **`community` label** is the primary attribution signal.  It is applied by
maintainers to explicitly mark issues as community-authored — a strong,
intentional indicator that does not rely on free-text heuristics.

Pre-fetched data is available at `/tmp/gh-aw/community-data/`:

```bash
# List all community-labeled issues
cat /tmp/gh-aw/community-data/community_issues.json \
  | jq -r '.[] | "- #\(.number): \(.title) by @\(.author.login) (closed: \(.closedAt // "open"))"'
```

Use the following **five-tier** approach to identify which community-labeled
issues were resolved in a given period.  Work through all tiers before
concluding, and never silently drop a community issue that was closed during
the period under review.

### Tier 0 — Direct issue contributions (confirmed, no PR required)

Any community-labelled issue that was closed with `stateReason == "COMPLETED"`
is a **confirmed contribution** by the issue author — no PR linkage is needed.
External contributors in this repo file issues rather than PRs; when a
maintainer tags an issue `community` and a coding agent resolves it, the issue
is closed as `COMPLETED`.  This is the strongest possible attribution signal.

```bash
# List all community issues closed as COMPLETED (direct contributions)
cat /tmp/gh-aw/community-data/community_issues.json \
  | jq -r '.[] | select(.stateReason == "COMPLETED") | "- #\(.number): \(.title) by @\(.author.login) (closed: \(.closedAt))"'
```

Record every matched issue as a **confirmed** attribution with type
`direct issue`.  These issues do **not** need to be checked against PR data
in Tiers 1–3.

### Tier 1 — GitHub-native closing references (primary)

For community issues **not already attributed in Tier 0**, `closing_refs_by_issue.json`
records the issues that GitHub itself marks as "closed by" each merged PR (the
native close-with-keyword feature).  This is the strongest PR-linkage signal
because it does not depend on free-text conventions.

```bash
COMMUNITY_NUMBERS=$(jq '[.[].number]' /tmp/gh-aw/community-data/community_issues.json)

jq --argjson community "$COMMUNITY_NUMBERS" \
  'to_entries
   | map(select((.key | tonumber) as $n | $community | any(. == $n)))
   | from_entries' \
  /tmp/gh-aw/community-data/closing_refs_by_issue.json
```

Record every matched issue as **confirmed** attribution.

### Tier 2 — PR body keyword parsing (secondary fallback)

For issues **not yet matched** in Tier 0 or Tier 1, scan PR bodies for the standard
closing keywords.  Both bare (`#123`) and fully-qualified (`org/repo#123`)
forms are supported.

```bash
jq -r '.[].body // ""' /tmp/gh-aw/community-data/pull_requests.json \
  | grep -oP '(?i)(?:close[sd]?|fix(?:e[sd])?|resolve[sd]?)\s*(?:github/gh-aw#|#)\K[0-9]+' \
  | sort -u
```

Add any newly matched community issues to the **confirmed** list.

### Tier 3 — GitHub MCP cross-reference lookup (follow-up / split issue chains)

For community issues **still unmatched** after Tiers 1 and 2, use the GitHub
MCP `issue_read` tool to look for indirect linkage through follow-up or split
issues:

1. Call `issue_read` with `method: "get_comments"` on the community issue.
2. Collect any referenced issue or PR numbers found in the body or comments.
3. Check whether any of those numbers appear as a closed target in
   `closing_refs_by_issue.json` or in the PR bodies already scanned.
4. If a transitive chain is found (community issue → follow-up issue → merged
   PR), record the community issue as **confirmed (via follow-up)** and note
   the chain in the output, for example: `_(via follow-up #N)_`.

### Tier 4 — Surface ambiguous candidates (fail soft, not silent)

After all active tiers, any community issue that was closed during the
review period but cannot be linked to a specific merged PR (and was **not**
already attributed in Tier 0) must **not** be silently dropped.  Add it to
the **"⚠️ Attribution Candidates Need Review"** section so a maintainer can
make the final call.

```bash
# Issues in the window that are NOT COMPLETED (Tier 0) and not matched by PR tiers
cat /tmp/gh-aw/community-data/community_issues_closed_in_window.json | \
  jq '[.[] | select(.stateReason != "COMPLETED")] | length'
```

### Output sections

**Confirmed attributions → Community Contributions**

```markdown
### 🌍 Community Contributions

A huge thank you to the community members who reported issues that were
resolved in this release:

- **@author** for Issue title ([#N](url)) _(direct issue)_
- **@author** for Issue title ([#N](url))
- **@author** for Issue title ([#N](url)) _(via follow-up #M)_
```

Attribution type suffixes:
- `_(direct issue)_` — Tier 0: issue closed as `COMPLETED`, no PR linkage needed
- _(no suffix)_ — Tier 1/2: PR closes the issue via native close reference or keyword
- `_(via follow-up #M)_` — Tier 3: indirect chain through a follow-up issue

**Unlinked candidates → Attribution Candidates Need Review**

```markdown
### ⚠️ Attribution Candidates Need Review

The following community issues were closed during this period but could not
be automatically linked to a specific merged PR.  Please verify whether they
should be credited:

- **@author** for Issue title ([#N](url)) — closed DATE, no confirmed PR linkage found
```

Omit the "Attribution Candidates Need Review" section entirely if every closed
community issue has confirmed attribution.
