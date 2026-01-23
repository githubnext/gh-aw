---
description: Daily Semgrep security scan for SQL injection and other vulnerabilities
name: Daily Semgrep Scan
imports:
  - shared/mcp/semgrep.md
on:
  schedule: daily
  workflow_dispatch:
permissions:
  contents: read
---

Scan the repository for SQL injection vulnerabilities using Semgrep.
