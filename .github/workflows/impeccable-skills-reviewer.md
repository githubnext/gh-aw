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
  max-continuations: 3
imports:
  - uses: shared/pr-review-base.md
    with:
      min-integrity: approved
  - shared/reporting.md
  - shared/otlp.md
pre-agent-steps:
  - name: Log retry attempt
    env:
      EXPR_RUN_ATTEMPT: ${{ github.run_attempt }}
    run: |
      if [ "${EXPR_RUN_ATTEMPT}" -gt 1 ]; then
        echo "⚠️ This is run attempt ${EXPR_RUN_ATTEMPT} — a previous attempt failed or was cancelled."
        echo "Likely cause: timeout (limit: 20 min), safe-output failure, or infrastructure issue."
      else
        echo "First run attempt."
      fi
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
  - name: Build skills manifest
    run: |
      find /tmp/gh-aw/.github/skills "${RUNNER_TEMP}/gh-aw/.github/skills" -name "SKILL.md" 2>/dev/null \
        | sort -u | head -40 | xargs -I{} cat {} > /tmp/gh-aw/agent/skills-manifest.txt 2>/dev/null || true
      echo "Skills manifest prepared ($(wc -c < /tmp/gh-aw/agent/skills-manifest.txt) bytes)"
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
timeout-minutes: 20

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

2. Invoke the `select-skills` agent to identify the most relevant skills for this PR.

   The agent reads the pre-fetched PR data and the skills manifest at `/tmp/gh-aw/agent/skills-manifest.txt` and returns a JSON array of the 1–3 most relevant `SKILL.md` paths.

3. Read the skill files returned by `select-skills` and apply them to your review.

   If `select-skills` returns an empty array or the manifest is empty, perform a normal high-signal review focused on correctness and security.

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
- Keep each inline comment under 120 words. Put extended reasoning in a `<details>` block.
- If the PR diff exceeds 1000 lines, limit inline comments to 5 maximum.
- End each actionable inline comment with `@copilot please address this.`
- If no visible action is needed, call `noop` with a brief explanation.

{{#runtime-import shared/noop-reminder.md}}

## agent: `select-skills`
---
model: claude-haiku-4.5
description: Classifies the PR change type and selects the 1–3 most relevant Impeccable SKILL.md paths from the pre-fetched manifest.
---
You are a deterministic skill-selection assistant for the Impeccable Skills Reviewer workflow.

Inputs are already pre-fetched on disk:
- `/tmp/gh-aw/agent/pr-meta.json` — PR metadata (title, body, changed files, additions, deletions)
- `/tmp/gh-aw/agent/pr-diff.patch` — first 3000 lines of the unified diff
- `/tmp/gh-aw/agent/skills-manifest.txt` — concatenated content of all installed SKILL.md files

Tasks:
1. Read `/tmp/gh-aw/agent/pr-meta.json` and up to 200 lines of `/tmp/gh-aw/agent/pr-diff.patch` (the full diff is already truncated to 3000 lines; reading 200 lines is sufficient for change-type classification).
2. Read `/tmp/gh-aw/agent/skills-manifest.txt` to discover available skills. If the file is missing or empty, return an empty array immediately.
3. Select 1–3 SKILL.md paths most relevant to the PR's change type and risk areas.

Return a JSON array of absolute file paths only — no prose, no explanation:
```json
["/tmp/gh-aw/.github/skills/example/SKILL.md"]
```

If no skills are installed or none are relevant, return an empty array: `[]`
