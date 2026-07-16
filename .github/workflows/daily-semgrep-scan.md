---
private: true
emoji: "🔒"
description: Daily Semgrep security scan for SQL injection and other vulnerabilities
name: Daily Semgrep Scan
imports:
  - shared/security-analysis-base.md
  - shared/mcp/semgrep.md
  - shared/otlp.md
on:
  schedule: daily
  workflow_dispatch:
timeout-minutes: 30
permissions:
  contents: read
  issues: read
  pull-requests: read
  security-events: read
  copilot-requests: write

sandbox:
  agent:
    sudo: false

safe-outputs:
  create-code-scanning-alert:
    driver: "Semgrep Security Scanner"

network:
  allowed:
    - defaults
    - pypi.org
    - files.pythonhosted.org
    - semgrep.dev

tools:
  cli-proxy: true

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
evals:
  - id: scan_completed
    question: Did the agent complete a Semgrep security scan and report on the findings?
  - id: alert_created_or_noop
    question: Was a code scanning alert created for real security findings, or does the agent output confirm no vulnerabilities were found?

---

Scan the repository for SQL injection vulnerabilities using Semgrep.

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


### Output Format

Use `###` (or lower) headers only.

Structure reports as: overview → key metrics/issues → collapsible detail → next actions.

Wrap long content with `<details><summary><b>View Details</b></summary>...</details>`.