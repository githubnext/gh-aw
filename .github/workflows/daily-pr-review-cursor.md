---
private: true
emoji: "🖱️"
description: Daily review of recently opened pull requests for code quality issues
on:
  schedule: daily
  workflow_dispatch:
permissions:
  contents: read
  issues: read
  pull-requests: read
sandbox:
  agent:
    id: awf
tracker-id: daily-pr-review-cursor
engine:
  id: copilot
  copilot-sdk: true
strict: true
network:
  allowed:
    - github
tools:
  github:
    mode: local
    toolsets: [repos, pull_requests]
  bash:
    - cat
    - grep
    - head
    - python3
    - sed
    - tail
    - wc
safe-outputs:
  create-issue:
    expires: 1d
    title-prefix: "[pr-review] "
    labels: [automation, code-review]
    max: 1
    close-older-issues: true
    close-older-key: daily-pr-review-cursor
  missing-tool:
timeout-minutes: 20
imports:
  - shared/network-defaults.md
  - shared/otlp.md
  - shared/reporting.md
features:
  gh-aw-detection: true
---

# Daily PR Code Quality Review

Review recently opened pull requests for common code quality issues and produce a concise daily
report.

## Step 1 — Find recent open PRs

Use GitHub MCP tools to list open pull requests from `${{ github.repository }}` with at most
`perPage: 5`, ordered by `created_at` descending. Record:
- PR number
- Title
- Author

## Step 2 — Review each PR

For each PR, fetch its diff using the GitHub MCP `pull_request_read` tool with `method: get_diff`.
If you inspect local files, use the `view` tool (not `read`).
Look for:
- Missing error handling in Go (`if err != nil` blocks that are absent where errors could occur)
- Exported functions without doc comments
- Test files without assertions (only `t.Log` or no checks)
- Oversized functions (> 80 lines in a single function body)

Keep each per-PR analysis to 3–5 bullet points.

## Step 3 — Produce report issue

Use the `reporting` skill to format the report body. Use `###` (or lower) headers only — never
`#` or `##`. Keep the overall quality signal and PR table visible at the top; wrap the full
per-PR findings in a `<details><summary><b>Full Findings</b></summary>` block.

Use the `create_issue` safe-output tool to post a daily report:

- **Title**: `[pr-review] Daily PR Code Quality Review — ${{ github.run_id }}`
- **Body**:
  - `### Summary` with a markdown table of PR number + link, top issue count, and overall
    quality signal: 🟢 (≤1 issue/PR) / 🟡 (2–3 issues/PR) / 🔴 (>3 issues/PR)
  - `<details><summary><b>Full Findings</b></summary>` wrapping the detailed per-PR issue lists
    (or "No issues" if clean)

If no open PRs were found, call the `noop` MCP tool with `"No open PRs to review today."`.