---
emoji: "🔒"
description: Daily Semgrep security scan for SQL injection and other vulnerabilities
name: Daily Semgrep Scan
imports:
  - shared/security-analysis-base.md
  - shared/otlp.md
on:
  schedule: daily
  workflow_dispatch:
permissions:
  contents: read
  issues: read
  pull-requests: read
  security-events: read
safe-outputs:
  create-code-scanning-alert:
    driver: "Semgrep Security Scanner"

network:
  allowed:
    - defaults
    - python
    - containers

tools:
  cli-proxy: true
steps:
  - name: Run Semgrep scan
    run: |
      set -euo pipefail
      mkdir -p /tmp/gh-aw/agent
      docker run --rm \
        -v "$GITHUB_WORKSPACE":/src \
        -w /src \
        semgrep/semgrep:latest \
        semgrep --config=auto --json . > /tmp/gh-aw/agent/semgrep-findings.json
      test -s /tmp/gh-aw/agent/semgrep-findings.json

experiments:
  semgrep_output_format:
    variants: [bullet_list, structured_sections, prose]
    description: "Tests whether the structure of Semgrep findings output (bullet list vs. grouped sections vs. prose) affects code scanning alert creation rate and output completeness."
    hypothesis: "H0: no change in alert creation rate across formats. H1: structured_sections produces ≥15% more alerts successfully created vs. baseline bullet_list."
    metric: alert_creation_rate
    secondary_metrics: [run_duration_ms, output_length_chars, findings_reported]
    guardrail_metrics:
      - name: run_success_rate
        threshold: ">=0.85"
    min_samples: 30
    weight: [34, 33, 33]
    start_date: "2026-05-17"
    analysis_type: proportion_test
    tags: [security, output-quality, semgrep]
    issue: 32795

---

The Semgrep scan has already been run in a deterministic setup step. Read `/tmp/gh-aw/agent/semgrep-findings.json` and use it as the source of truth for all findings.

Do not rerun Semgrep or call any Semgrep MCP tools. Use the precomputed JSON findings file to produce your report and emit `create-code-scanning-alert` calls only.

{{#if experiments.semgrep_output_format == 'bullet_list' }}
Report each finding as a flat bullet point in this format:
- **[SEVERITY]** `<file>:<line>` — Rule: `<rule_id>` — <message>

Create one code scanning alert per finding.
{{/if}}
{{#if experiments.semgrep_output_format == 'structured_sections' }}
Structure your findings report with:
1. A summary table: | Severity | Count |
2. Sections grouped by severity (Critical, High, Medium, Low), then by rule ID
3. For each finding: file path, line number, rule, and recommended fix

Create one code scanning alert per finding.
{{/if}}
{{#if experiments.semgrep_output_format == 'prose' }}
Write a narrative security assessment describing the vulnerability patterns found. Embed specific findings (file, line, rule) within the prose. Conclude with a prioritized remediation list.

Create one code scanning alert per finding.
{{/if}}

If the findings file reports no results, state that clearly and do not create any code scanning alerts.

{{#runtime-import shared/noop-reminder.md}}
