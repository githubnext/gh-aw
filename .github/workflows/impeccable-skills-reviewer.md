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
  - name: Pre-fetch PR diff
    env:
      GH_TOKEN: ${{ github.token }}
      PR_NUMBER: ${{ github.event.pull_request.number }}
      EXPR_GITHUB_REPOSITORY: ${{ github.repository }}
      PR_DIFF_MAX_LINES: "3000"
    run: |
      set -euo pipefail
      mkdir -p /tmp/gh-aw/agent
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
      echo "Pre-fetched PR diff (${LINES} lines) and metadata"
tools:
  cli-proxy: true
  github:
    mode: gh-proxy
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

## Context

- **Repository**: ${{ github.repository }}
- **Pull Request**: #${{ github.event.pull_request.number }}
- **PR Title**: "${{ github.event.pull_request.title }}"
- **Author**: ${{ github.actor }}

## Process

1. Read pre-fetched PR files only:

   - `/tmp/gh-aw/agent/pr-meta.json`
   - `/tmp/gh-aw/agent/pr-diff.patch`

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

{{#runtime-import shared/noop-reminder.md}}
