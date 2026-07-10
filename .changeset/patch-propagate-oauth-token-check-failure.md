---
"gh-aw": patch
---

Propagate OAuth token check failure to the conclusion job failure issue. When `COPILOT_GITHUB_TOKEN`, `GH_AW_GITHUB_TOKEN`, or `GH_AW_GITHUB_MCP_SERVER_TOKEN` is configured with an OAuth token (`gho_...`), the activation job now sets an `oauth_token_check_failed` output, causing the conclusion job to run and create an actionable failure issue with remediation guidance.
