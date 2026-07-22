---
# Shared pre-agent-steps for pre-fetching PR diff, metadata, and review comments.
# Works for both pull_request events and slash_command (issue) events on PRs.
#
# Outputs written to:
#   /tmp/gh-aw/agent/pr-diff.patch          — unified diff (up to 3000 lines)
#   /tmp/gh-aw/agent/pr-meta.json           — PR metadata (number, title, body, etc.)
#   /tmp/gh-aw/agent/pr-review-comments.json — existing inline review comments
#
# Skip-check: if all three files already exist (cache hit), fetch is skipped.
#
# Usage:
#   cache:
#     key: pr-prefetch-${{ github.event.pull_request.head.sha || github.event.issue.number }}
#     path: /tmp/gh-aw/agent
#     restore-keys:
#       - pr-prefetch-${{ github.event.pull_request.number || github.event.issue.number }}-
#   imports:
#     - shared/pr-diff-data-fetch.md

pre-agent-steps:
  - name: Pre-fetch PR diff and review comments
    env:
      GH_TOKEN: ${{ github.token }}
      PR_NUMBER: ${{ github.event.issue.number || github.event.pull_request.number }}
      EXPR_GITHUB_REPOSITORY: ${{ github.repository }}
      PR_DIFF_MAX_LINES: "3000"
    run: |
      set -euo pipefail
      mkdir -p /tmp/gh-aw/agent
      # Skip fetch if cache already populated this data (actions/cache restore)
      if [ -f /tmp/gh-aw/agent/pr-diff.patch ] && [ -f /tmp/gh-aw/agent/pr-meta.json ] && [ -f /tmp/gh-aw/agent/pr-review-comments.json ]; then
        LINES=$(wc -l < /tmp/gh-aw/agent/pr-diff.patch)
        COMMENT_COUNT=$(jq 'length' /tmp/gh-aw/agent/pr-review-comments.json)
        echo "Cache hit: using pre-fetched PR data (${LINES} diff lines, ${COMMENT_COUNT} review comments)"
      else
        { gh pr diff "$PR_NUMBER" --repo $EXPR_GITHUB_REPOSITORY \
            --exclude '**/*.lock.yml' \
            --exclude '**/generated/**' \
            --exclude '**/dist/**' \
            --exclude '**/build/**' \
            || true; } | head -n "${PR_DIFF_MAX_LINES}" > /tmp/gh-aw/agent/pr-diff.patch
        LINES=$(wc -l < /tmp/gh-aw/agent/pr-diff.patch)
        gh pr view "$PR_NUMBER" \
          --repo $EXPR_GITHUB_REPOSITORY \
          --json number,title,body,headRefName,additions,deletions,changedFiles,files \
          > /tmp/gh-aw/agent/pr-meta.json
        gh api "repos/$EXPR_GITHUB_REPOSITORY/pulls/$PR_NUMBER/comments" \
          --paginate \
          --jq '.[] | {id, path, line: (.line // .original_line), body: .body[:200], user: .user.login}' \
          2>/dev/null | jq -s '.' > /tmp/gh-aw/agent/pr-review-comments.json \
          || echo '[]' > /tmp/gh-aw/agent/pr-review-comments.json
        COMMENT_COUNT=$(jq 'length' /tmp/gh-aw/agent/pr-review-comments.json)
        echo "Pre-fetched PR diff (${LINES} lines), metadata, and ${COMMENT_COUNT} existing review comments"
      fi
---

<!--
## PR Diff Data Fetch

Shared pre-agent-steps component used by PR reviewer workflows to pre-fetch PR
diff, metadata, and inline review comments before the agent starts.

### Why this shared component exists

Three reviewer workflows (pr-code-quality-reviewer, impeccable-skills-reviewer,
mattpocock-skills-reviewer) previously each duplicated identical pre-fetch shell
steps. Extracting them into this shared component eliminates the duplication and
ensures all three workflows use the same fetch logic and cache key, so that the
dedicated `pr-data-prefetch.yml` workflow can warm the cache once per commit
before the reviewer agents start.

### How the cache works

1. `pr-data-prefetch.yml` triggers simultaneously with the reviewer workflows on
   `pull_request: [ready_for_review]` events. Because it has no AI engine, it
   completes (and saves the `pr-prefetch-<sha>` Actions cache) in ~30–60 s.
2. Reviewer workflows' activation jobs take ~60–90 s; their agent jobs restore
   the cache before running `pre-agent-steps`.
3. When the cache is warm, this shared step detects the pre-fetched files and
   skips all GitHub API calls.

### Output files

| File | Content |
|---|---|
| `/tmp/gh-aw/agent/pr-diff.patch` | Unified diff (lock/generated/dist/build excluded, capped at 3000 lines) |
| `/tmp/gh-aw/agent/pr-meta.json` | `number, title, body, headRefName, additions, deletions, changedFiles, files` |
| `/tmp/gh-aw/agent/pr-review-comments.json` | Array of `{id, path, line, body, user}` (body capped at 200 chars) |
-->
