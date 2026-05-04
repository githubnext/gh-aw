---
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
  cache-memory: true
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
    - "python3 *"
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

      - name: Set up Python
        uses: actions/setup-python@v5
        with:
          python-version: "3.11"

      - name: Install geo-optimizer-skill
        run: pip install geo-optimizer-skill

      - name: Create cache directory
        run: mkdir -p /tmp/gh-aw/cache-memory/geo-optimizer

      - name: Audit documentation site homepage
        run: |
          geo audit --url https://github.github.com/gh-aw/ --format json \
            > /tmp/gh-aw/cache-memory/geo-optimizer/docs-site-audit.json 2>&1 || true

      - name: Audit documentation sitemap
        run: |
          geo audit --sitemap https://github.github.com/gh-aw/sitemap.xml \
            --max-urls 20 --format json \
            > /tmp/gh-aw/cache-memory/geo-optimizer/docs-sitemap-audit.json 2>&1 || true

      - name: Audit README via GitHub repository page
        run: |
          geo audit --url https://github.com/${{ github.repository }} --format json \
            > /tmp/gh-aw/cache-memory/geo-optimizer/readme-audit.json 2>&1 || true

      - name: Write audit metadata
        run: |
          python3 - <<'EOF'
          import json, subprocess, datetime, os

          metadata = {
            "run_id": "${{ github.run_id }}",
            "timestamp": datetime.datetime.now(datetime.timezone.utc).strftime("%Y-%m-%d %H-%M-%S"),
            "docs_url": "https://github.github.com/gh-aw/",
            "readme_url": "https://github.com/${{ github.repository }}",
            "repository": "${{ github.repository }}",
          }
          path = "/tmp/gh-aw/cache-memory/geo-optimizer/metadata.json"
          with open(path, "w") as f:
            json.dump(metadata, f, indent=2)
          print(f"Wrote metadata to {path}")
          EOF

      - name: Save geo-optimizer results to cache
        uses: actions/cache/save@v4
        with:
          # Key prefix 'dailygeooptimizer' is the sanitized workflow ID (hyphens stripped
          # from 'daily-geo-optimizer') — matches GH_AW_WORKFLOW_ID_SANITIZED in the agent job.
          key: memory-none-nopolicy-dailygeooptimizer-${{ github.run_id }}
          path: /tmp/gh-aw/cache-memory

imports:
  - uses: shared/daily-audit-base.md
    with:
      title-prefix: "[geo-optimizer] "
      expires: 3d
---

{{#runtime-import? .github/shared-instructions.md}}

# GEO Optimizer Daily Audit

You are the GEO (Generative Engine Optimization) audit agent. Your task is to analyze the audit results produced by `geo-optimizer-skill` and report on the AI visibility of the `${{ github.repository }}` README and documentation site.

## Context

- **Repository**: ${{ github.repository }}
- **Run ID**: ${{ github.run_id }}
- **Run URL**: ${{ github.server_url }}/${{ github.repository }}/actions/runs/${{ github.run_id }}

## Your Mission

Analyze the GEO audit results stored in `/tmp/gh-aw/cache-memory/geo-optimizer/` and create a GitHub Discussion summarizing the findings, trends, and actionable recommendations to improve AI-engine citation coverage for this project.

---

## Phase 1: Load Audit Results

Read all JSON files from the cache directory:

```bash
ls /tmp/gh-aw/cache-memory/geo-optimizer/
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

## Phase 2: Load Historical Trend Data

Check for a trend history file from previous runs:

```bash
ls /tmp/gh-aw/cache-memory/geo-optimizer/history.json 2>/dev/null || echo "No history yet"
```

If history exists, parse it with `jq` to identify score changes over time (improvements or regressions).

## Phase 3: Analyze and Summarize

Based on the audit results, identify:

1. **Scores** — What is the current GEO score for the docs site and README?
2. **Top Strengths** — What's already optimized well?
3. **Critical Gaps** — What's missing or scoring poorly?
4. **High-Impact Fixes** — Which specific recommendations would most improve AI citation coverage?
5. **Trends** — Has the score improved or regressed since the last run (if history exists)?

## Phase 4: Update History

Before creating the discussion, update the rolling history file with today's scores. Read the current metadata.json to get the timestamp, then append today's scores to `history.json`. Keep only the last 30 entries.

Example format:
```json
[
  {"timestamp": "2026-05-04 10-30-00", "docs_score": 72, "readme_score": 58}
]
```

Write the updated history back with:
```bash
# read current scores from the audit JSONs, then write updated history
```

Limit the history file to the most recent 30 entries (drop the oldest when the list exceeds 30).

Create a GitHub Discussion with the audit findings using the following structure:

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

[If history exists:]
**Trend**: Docs score [+X / -X] since last run · README score [+X / -X]

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
- **Safe filenames**: Use `YYYY-MM-DD HH-MM-SS` format (no colons) for any timestamps you write.
- **Efficient**: Read each file once; avoid redundant bash calls.
- **History integrity**: Append to history before writing; keep only the most recent 30 entries.

{{#runtime-import shared/noop-reminder.md}}
