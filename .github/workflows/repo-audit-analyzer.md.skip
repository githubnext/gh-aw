---
description: Comprehensive repository audit to identify productivity improvement opportunities using agentic workflows
on:
  workflow_dispatch:
    inputs:
      repository:
        description: 'Target repository to audit (e.g., FStarLang/FStar)'
        required: false
        type: string
        default: 'FStarLang/FStar'
permissions:
  contents: read
  actions: read
  issues: read
  pull-requests: read
tools:
  github:
    toolsets: [default]
  web-fetch:
  bash: ["*"]
  cache-memory:
    - id: repo-audits
      key: repo-audits-${{ github.workflow }}
safe-outputs:
  create-discussion:
    category: "audits"
    max: 1
    close-older-discussions: true
  missing-tool:
    create-issue: true
timeout-minutes: 45
strict: true
imports:
  - shared/reporting.md
---

<!-- Edit the file linked below to modify the agent without recompilation. Feel free to move the entire markdown body to that file. -->
{{#runtime-import agentics/repo-audit-analyzer.md}}
