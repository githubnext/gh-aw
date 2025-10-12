---
"gh-aw": minor
---

Add workflow run logs download and extraction to audit/logs commands

The `gh aw logs` and `gh aw audit` commands now automatically download and extract GitHub Actions workflow run logs in addition to artifacts, providing complete audit trail information by including the actual console output from workflow executions. The implementation includes security protection against zip slip vulnerability and graceful error handling for missing or expired logs.
