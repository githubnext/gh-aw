---
private: true
emoji: "🧭"
description: Daily review of specification documents for coverage gaps and outdated content using Kiro
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
    runtime: docker-sbx
    sudo: true
tracker-id: daily-spec-coverage-kiro
engine:
  id: kiro
model: kiro/claude-sonnet-4-5
strict: true
network:
  allowed:
    - defaults
    - github
tools:
  github:
    mode: local
    toolsets: [repos, issues]
  bash:
    - cat
    - grep
    - find
    - wc
safe-outputs:
  create-issue:
    expires: 2d
    title-prefix: "[spec-coverage] "
    labels: [automation, documentation]
    max: 1
    close-older-issues: true
    close-older-key: daily-spec-coverage-kiro
  missing-tool:
timeout-minutes: 20
imports:
  - shared/kiro.md
  - shared/otlp.md
---

# Daily Spec Coverage Review — Kiro

Audit the specification and documentation files in this repository for coverage gaps, stale
references, and missing sections.

## Step 1 — List specification files

Find all markdown files under `.github/aw/` that describe specification or syntax concepts:

```bash
find .github/aw -name "*.md" | sort | head -30
```

## Step 2 — Check for stale cross-references

For each file in `.github/aw/`, scan for `[text](filename.md)` links and verify the referenced
file exists:

```bash
grep -rh "\[.*\]([a-z].*\.md)" .github/aw/ \
  | grep -oP '\(([^)]+\.md)\)' | tr -d '()' | sort -u \
  | while read f; do
      [ -f ".github/aw/$f" ] || echo "BROKEN: $f"
    done
```

## Step 3 — Find spec files without a description front-matter field

```bash
for f in .github/aw/*.md; do
  if ! grep -q "^description:" "$f"; then echo "NO DESC: $f"; fi
done | head -10
```

## Step 4 — Search for open issues mentioning spec gaps

Use the GitHub MCP `list_issues` tool to fetch the 5 most-recently-created open issues from
`${{ github.repository }}` that contain "spec" or "docs" in their title. Record issue numbers
and titles.

## Step 5 — Report

Use the `create_issue` safe-output tool to post the daily report:

- **Title**: `[spec-coverage] Daily Spec Coverage Report — ${{ github.run_id }}`
- **Body**: Summarize
  - Broken cross-references found
  - Files without description frontmatter
  - Open issues mentioning spec gaps
  - Recommended next actions (if any)

If all checks pass, call `noop` with `"Spec coverage audit passed — no gaps found."`.
