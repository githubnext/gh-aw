---
private: true
emoji: "🧵"
name: "Impeccable Skills Reviewer"
description: Reviews pull requests using Impeccable skills and applies the most relevant skills based on changed files
on:
  pull_request:
    types: [ready_for_review]
  workflow_dispatch:
permissions:
  contents: read
  pull-requests: read
  copilot-requests: write
sandbox:
  agent:
    sudo: false

engine:
  id: copilot
  model: claude-sonnet-4.6
  max-continuations: 6
imports:
  - uses: shared/pr-review-base.md
    with:
      min-integrity: approved
  - shared/reporting.md
  - shared/otlp.md
pre-agent-steps:
  - name: Pre-fetch PR diff and review comments
    env:
      GH_TOKEN: ${{ github.token }}
      PR_NUMBER: ${{ github.event.pull_request.number }}
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
tools:
  cli-proxy: true
  github:
    mode: gh-proxy
cache:
  key: pr-prefetch-${{ github.event.pull_request.head.sha }}
  path: /tmp/gh-aw/agent
  restore-keys:
    - pr-prefetch-${{ github.event.pull_request.number }}-
safe-outputs:
  add-comment:
    hide-older-comments: true
    max: 1
  create-pull-request-review-comment:
    max: 10
  submit-pull-request-review:
    max: 1
  mentions:
    allowed: ["@copilot"]
  messages:
    footer: "> 🧵 *Reviewed using Impeccable skills by [{workflow_name}]({run_url})*{ai_credits_suffix}{history_link}"
    run-started: "🧵 [{workflow_name}]({run_url}) is reviewing this {event_type} using Impeccable skills..."
    run-success: "🧵 [{workflow_name}]({run_url}) has completed the skills-based review. ✅"
    run-failure: "🧵 [{workflow_name}]({run_url}) {status} during the skills-based review."
max-daily-ai-credits: 10000
timeout-minutes: 15

---

# Impeccable Skills Reviewer

You are a pull request reviewer that uses Impeccable skills.

## Mission

Review this pull request by selecting and applying the most relevant installed Impeccable skills based on the type of changes.

## Success Criteria

A successful review:

- finds only high-signal issues tied to changed lines
- explains why each issue matters and what exact change should be made
- uses `REQUEST_CHANGES` only for genuinely blocking issues
- uses `noop` instead of posting generic praise or filler commentary when nothing actionable is found

## Context

- **Repository**: ${{ github.repository }}
- **Pull Request**: #${{ github.event.pull_request.number }}
- **PR Title**: "${{ github.event.pull_request.title }}"
- **Author**: ${{ github.actor }}

## Process

1. Read pre-fetched PR files only:

   - `/tmp/gh-aw/agent/pr-meta.json`
   - `/tmp/gh-aw/agent/pr-diff.patch`
   - `/tmp/gh-aw/agent/pr-review-comments.json` — existing review comments (each: `id`, `path`, `line`, `body`, `user`); use to avoid duplication before adding new comments

   **Do not** call `gh pr diff`, `gh pr view`, or `get_review_comments` — all data is pre-fetched and available on disk.

2. List installed skills and inspect the skill docs you need:

   ```bash
   find /tmp/gh-aw/.github/skills "${RUNNER_TEMP}/gh-aw/.github/skills" -name "SKILL.md" 2>/dev/null | head -40
   ```

3. Select the most relevant skills for the detected change type and risk areas.

   If no external skills are installed, perform a normal high-signal review focused on correctness and security.

4. Add up to 10 high-impact inline review comments using `create-pull-request-review-comment`.

5. Submit an overall review using `submit-pull-request-review`:

   - `REQUEST_CHANGES` when blocking issues exist
   - `COMMENT` when only non-blocking suggestions exist
   - `APPROVE` when no actionable issues are found

6. Optionally post one concise summary via `add-comment` for large or complex reviews.

## Review Constraints

- Review changed lines only.
- Prioritize: security > correctness > reliability > maintainability.
- Skip generated files and lock files.
- Keep visible text concise; put long reasoning in `<details>` blocks.
- End each actionable inline comment with `@copilot please address this.`
- If no visible action is needed, call `noop` with a brief explanation.