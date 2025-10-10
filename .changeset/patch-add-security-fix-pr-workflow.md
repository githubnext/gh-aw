---
"gh-aw": patch
---

Add security-fix-pr workflow for automated security issue remediation

This adds a new agentic workflow that automatically generates pull requests to fix code security issues detected by GitHub Code Scanning. The workflow can be triggered manually via workflow_dispatch and will identify the first open security alert, analyze the vulnerability, generate a fix, and create a draft pull request for review.
