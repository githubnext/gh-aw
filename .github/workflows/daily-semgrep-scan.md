---
description: Daily Semgrep security scan for SQL injection and other vulnerabilities
name: Daily Semgrep Scan
imports:
  - shared/security-analysis-base.md
  - shared/mcp/semgrep.md
  - shared/observability-otlp.md
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

tools:
  cli-proxy: true

---

Scan the repository for SQL injection vulnerabilities using Semgrep.

{{#import shared/noop-reminder.md}}
