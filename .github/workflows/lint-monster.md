---
emoji: "🧌"
name: LintMonster
description: Daily workflow that runs custom linters, groups findings, and launches up to three Copilot agent sessions to fix lint issues
on:
  schedule: daily
  workflow_dispatch:
permissions:
  contents: read
  issues: read
  discussions: read
  pull-requests: read
tracker-id: lint-monster
engine:
  id: copilot
  model: claude-haiku-4.5
strict: true
timeout-minutes: 45
tools:
  cli-proxy: true
  github:
    mode: gh-proxy
    toolsets: [default, issues, discussions]
  bash:
    - "cat /tmp/gh-aw/agent/golint-custom.log"
    - "cat /tmp/gh-aw/agent/lint-diagnostics.txt"
    - "cat /tmp/gh-aw/agent/skill-index.txt"
    - "cat .github/skills/go-linters/SKILL.md"
steps:
  - name: Run custom lint pre-check
    id: lint_scan
    run: |
      set -euo pipefail
      mkdir -p /tmp/gh-aw/agent
      rm -f /tmp/gh-aw/agent/lint-clean.flag

      if make golint-custom > /tmp/gh-aw/agent/golint-custom.log 2>&1; then
        : > /tmp/gh-aw/agent/lint-diagnostics.txt
        : > /tmp/gh-aw/agent/skill-index.txt
        touch /tmp/gh-aw/agent/lint-clean.flag
        exit 0
      fi

      grep -E '^[^:]+:[0-9]+:[0-9]+:' /tmp/gh-aw/agent/golint-custom.log > /tmp/gh-aw/agent/lint-diagnostics.txt || true
      diag_count=$(wc -l < /tmp/gh-aw/agent/lint-diagnostics.txt | tr -d ' ')
      if [ "${diag_count}" -eq 0 ]; then
        grep -E '^[[:space:]]*[^[:space:]].*$' /tmp/gh-aw/agent/golint-custom.log | head -n 50 > /tmp/gh-aw/agent/lint-diagnostics.txt || true
        diag_count=$(wc -l < /tmp/gh-aw/agent/lint-diagnostics.txt | tr -d ' ')
      fi

      find .github/skills -maxdepth 6 -name 'SKILL.md' | sort > /tmp/gh-aw/agent/skill-index.txt
      echo "Lint diagnostics captured: ${diag_count}"

safe-outputs:
  create-issue:
    expires: 7d
    title-prefix: "[lint-monster] "
    labels: [automation, lint, cookie]
    max: 3
  assign-to-agent:
    max: 3
    target: "*"
    allowed: [copilot]
  create-discussion:
    expires: 2d
    category: reports
    title-prefix: "[lint-monster] "
    max: 1
    close-older-discussions: true
  noop:

imports:
  - shared/otlp.md
---

{{#runtime-import? .github/shared-instructions.md}}

# LintMonster

You are **LintMonster**, a daily custom-linter remediation orchestrator.

## Mission

Use the pre-check lint output from `make golint-custom`. If lint is clean, do nothing. If lint issues exist, group them and launch up to three Copilot agent sessions to resolve the groups.

## Runtime Inputs

Read:
- `/tmp/gh-aw/agent/golint-custom.log`
- `/tmp/gh-aw/agent/lint-diagnostics.txt`
- `/tmp/gh-aw/agent/skill-index.txt`
- `/tmp/gh-aw/agent/lint-clean.flag` (exists only when lint is already clean)

## Skill mining and fusion (required)

1. Read `/tmp/gh-aw/agent/skill-index.txt` and identify the minimum relevant skill material for fixing custom linter findings.
2. Use **skill fusion**: extract only precise fragments instead of loading full skills broadly.

<!-- gh-skill-fusion: .github/skills/go-linters/SKILL.md#build-and-test-linters -->
Use these fused constraints while creating remediation instructions:
- Validate fixes by running `make golint-custom`.
- Keep remediation scoped to findings in the assigned lint group.
- Prefer minimal, targeted code edits.
<!-- End fusion -->

Convert fused guidance into clear, actionable instructions that Copilot can execute for each lint group issue.

## Required flow

1. If `/tmp/gh-aw/agent/lint-clean.flag` exists, call `noop` and stop.
2. Group findings from `/tmp/gh-aw/agent/lint-diagnostics.txt` into **at most three** distinct sets (for example by subsystem/path prefix).
3. For each selected group:
   - Create one issue summarizing findings (paths, representative diagnostics, expected outcome).
   - Include a concise remediation checklist using fused skill guidance.
   - Assign the created issue to Copilot using `assign_to_agent`.
4. If at least one assignment succeeded, create one discussion report containing:
   - Daily lint scan summary
   - Group definitions and finding counts
   - Issues created and agent assignments
   - Any groups skipped and why
5. If no assignments were made, call `noop` with a short reason.

## Output rules

- Launch **no more than three** agent sessions total.
- Never assign the same group twice.
- Always use safe outputs for issue creation, assignment, and discussion creation.
- Final action must be `create_discussion` when agents were launched, otherwise `noop`.
