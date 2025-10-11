---
"gh-aw": patch
---

Update security-fix-pr workflow to use GitHub remote mode

This change updates the security-fix-pr workflow to use remote mode for GitHub tools instead of local Docker mode. Remote mode provides faster startup times by using the hosted GitHub MCP server, eliminating the Docker container overhead. This improves the workflow's performance and responsiveness when analyzing and fixing security vulnerabilities.
