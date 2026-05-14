---
emoji: "🌍"
description: Daily GEO (Generative Engine Optimization) audit of the README and documentation site using geo-optimizer-skill
on:
  schedule: daily
  workflow_dispatch:
permissions:
  contents: read
  issues: read
  pull-requests: read
  discussions: read
tracker-id: daily-geo-optimizer
engine: copilot
strict: true
timeout-minutes: 30
tools:
  cli-proxy: true
  github:
    mode: gh-proxy
    toolsets: [default]
  bash:
    - "cat *"
    - "ls *"
    - "echo *"
    - "date *"
    - "jq *"
    - "find *"
    - "grep *"
features:
  copilot-requests: true
if: needs.geo_audit.result == 'success'
jobs:
  geo_audit:
    runs-on: ubuntu-latest
    permissions:
      contents: read
    steps:
      - name: Checkout repository
        uses: actions/checkout@v4

      - name: Setup Python
        uses: actions/setup-python@v5
        with:
          python-version: "3.11"

      - name: Install geo-optimizer-skill
        run: pip install geo-optimizer-skill

      - name: Create results directory
        run: mkdir -p /tmp/gh-aw/geo-optimizer

      - name: Audit documentation site homepage
        run: |
          geo audit --url https://github.github.com/gh-aw/ --format json \
            > /tmp/gh-aw/geo-optimizer/docs-site-audit.json 2>&1 || true

      - name: Audit documentation sitemap
        run: |
          geo audit --sitemap https://github.github.com/gh-aw/sitemap.xml \
            --max-urls 20 --format json \
            > /tmp/gh-aw/geo-optimizer/docs-sitemap-audit.json 2>&1 || true

      - name: Audit README via GitHub repository page
        run: |
          geo audit --url https://github.com/${{ github.repository }} --format json \
            > /tmp/gh-aw/geo-optimizer/readme-audit.json 2>&1 || true

      - name: Write audit metadata
        run: |
          python3 - <<'EOF'
          import json, datetime

          metadata = {
            "run_id": "${{ github.run_id }}",
            "timestamp": datetime.datetime.now(datetime.timezone.utc).strftime("%Y-%m-%d %H-%M-%S"),
            "docs_url": "https://github.github.com/gh-aw/",
            "readme_url": "https://github.com/${{ github.repository }}",
            "repository": "${{ github.repository }}",
          }
          path = "/tmp/gh-aw/geo-optimizer/metadata.json"
          with open(path, "w") as f:
            json.dump(metadata, f, indent=2)
          print(f"Wrote metadata to {path}")
          EOF

      - name: Upload geo-optimizer results
        uses: actions/upload-artifact@v7.0.1
        with:
          name: geo-optimizer-results
          path: /tmp/gh-aw/geo-optimizer
          if-no-files-found: error
          retention-days: 3

steps:
  - name: Download geo-optimizer results
    uses: actions/download-artifact@v8.0.1
    with:
      name: geo-optimizer-results
      path: /tmp/gh-aw/geo-optimizer

imports:
  - uses: shared/daily-audit-base.md
    with:
      title-prefix: "[geo-optimizer] "
      expires: 3d

  - shared/observability-otlp.md
---

{{#runtime-import? .github/shared-instructions.md}}

# GEO Optimizer Daily Audit

You are the GEO (Generative Engine Optimization) audit agent. Your task is to analyze the audit results produced by `geo-optimizer-skill` and report on the AI visibility of the `${{ github.repository }}` README and documentation site.

## Context

- **Repository**: ${{ github.repository }}
- **Run ID**: ${{ github.run_id }}
- **Run URL**: ${{ github.server_url }}/${{ github.repository }}/actions/runs/${{ github.run_id }}

## Your Mission

Analyze the GEO audit results downloaded from the `geo-optimizer-results` artifact into `/tmp/gh-aw/geo-optimizer/` and create a GitHub Discussion summarizing the findings and actionable recommendations to improve AI-engine citation coverage for this project.

---

## Phase 1: Load Audit Results

Read all JSON files from the results directory:

```bash
ls /tmp/gh-aw/geo-optimizer/
```

- `docs-site-audit.json` — full GEO audit of `https://github.github.com/gh-aw/`
- `docs-sitemap-audit.json` — sitemap-wide audit of up to 20 documentation pages
- `readme-audit.json` — GEO audit of the GitHub repository homepage (README)
- `metadata.json` — run metadata (timestamp, URLs)

Use `cat` and `jq` to inspect the contents of each file. Focus on:
- Overall score (0–100) and score band (Critical / Foundation / Good / Excellent)
- Top issues and recommendations per category
- Citability score and methods
- Negative signals detected
- Scores broken down by area: Robots.txt, llms.txt, Schema JSON-LD, Meta Tags, Content, Brand & Entity, Signals, AI Discovery

## Phase 2: Analyze and Summarize

Based on the audit results, identify:

1. **Scores** — What is the current GEO score for the docs site and README?
2. **Top Strengths** — What's already optimized well?
3. **Critical Gaps** — What's missing or scoring poorly?
4. **High-Impact Fixes** — Which specific recommendations would most improve AI citation coverage?

## Phase 3: Create Discussion Report

### Title
`[geo-optimizer] GEO Audit Report — YYYY-MM-DD`

Use today's date derived from the metadata.json timestamp.

### Body

```markdown
## GEO Audit Report — ${{ github.repository }}

**Audit Date**: [date from metadata]
**Run**: [link to run]

---

### 📊 Scores

| Target | Score | Band |
|--------|-------|------|
| Docs site (`github.github.com/gh-aw/`) | X/100 | Good/Foundation/... |
| README (github.com/github/gh-aw) | X/100 | ... |

---

### ✅ Top Strengths

[3–5 items already optimized well]

---

### 🚨 Critical Gaps

[Top 3–5 issues preventing AI engine citations]

---

### 🔧 Recommended Fixes

[Prioritized, actionable list of specific improvements ordered by impact]

<details>
<summary>📋 Full Breakdown by Category</summary>

[Category-by-category scores and notes from the audit JSON]

</details>

<details>
<summary>📄 Sitemap Page Scores</summary>

[Top pages by score from the sitemap audit, if available]

</details>

---
*Automated audit powered by [geo-optimizer-skill](https://github.com/Auriti-Labs/geo-optimizer-skill) · [Run logs](${{ github.server_url }}/${{ github.repository }}/actions/runs/${{ github.run_id }})*
```

---

## Important Guidelines

- **Be specific**: Quote actual scores and finding text from the JSON, don't make them up.
- **If a file is missing or empty**: Note it clearly rather than fabricating data.
- **Efficient**: Read each file once; avoid redundant bash calls.

{{#runtime-import shared/noop-reminder.md}}
