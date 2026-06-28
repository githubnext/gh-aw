---
private: true
emoji: "🧵"
name: "Impeccable Skills Reviewer"
description: Reviews pull requests using Impeccable skills from needex/skills and applies the most relevant skills based on changed files
on:
  pull_request:
    types: [ready_for_review]
  workflow_dispatch:
permissions:
  contents: read
  pull-requests: read
  copilot-requests: write
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
  - name: Upgrade gh CLI
    run: |
      bash "${RUNNER_TEMP}/gh-aw/actions/install_gh_cli.sh"
      GH_VERSION=$(gh --version | head -1 | grep -oP '\d+\.\d+\.\d+')
      echo "gh version: ${GH_VERSION}"
      REQUIRED="2.90.0"
      if ! printf '%s\n%s\n' "$REQUIRED" "$GH_VERSION" | sort -V -C; then
        echo "::error::gh ${GH_VERSION} is older than required ${REQUIRED} (gh skill support requires v2.90+)"
        exit 1
      fi
  - name: Install Impeccable skills
    env:
      GH_TOKEN: ${{ github.token }}
    run: |
      set -euo pipefail
      SKILLS_SRC="needex/skills"
      SKILLS_DST="${RUNNER_TEMP}/gh-aw/impeccable-skills"
      SKILLS_LIST="${RUNNER_TEMP}/gh-aw/impeccable-skills-list.txt"
      mkdir -p "${SKILLS_DST}"

      # Discover available skills and install each one individually.
      # External skill repositories may be unavailable; continue with a best-effort review.
      if gh api "repos/${SKILLS_SRC}/contents/skills" --jq '[.[] | select(.type == "dir") | .name] | .[]' > "${SKILLS_LIST}"; then
        while IFS= read -r skill; do
          if ! gh skill install "${SKILLS_SRC}" "$skill" --dir "${SKILLS_DST}" --force; then
            echo "::warning::Failed to install skill '${skill}' from ${SKILLS_SRC}; continuing."
          fi
        done < "${SKILLS_LIST}"
      else
        echo "::warning::Failed to discover skills from ${SKILLS_SRC}; continuing without external skills."
      fi

      SKILL_COUNT=$(find "${SKILLS_DST}" -name "SKILL.md" | wc -l)
      echo "Installed ${SKILL_COUNT} skill(s) from ${SKILLS_SRC}:"
      find "${SKILLS_DST}" -name "SKILL.md" | head -20
      if [ "${SKILL_COUNT}" -eq 0 ]; then
        echo "::warning::No SKILL.md files found after installing ${SKILLS_SRC}; review will continue without external skills."
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

You are a pull request reviewer that uses Impeccable skills from `needex/skills`.

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
   find "${RUNNER_TEMP}/gh-aw/impeccable-skills" -name "SKILL.md" | head -40
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
