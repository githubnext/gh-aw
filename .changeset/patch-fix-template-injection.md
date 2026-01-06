---
"gh-aw": patch
---

Fix template injection vulnerabilities in the workflow compiler by moving
user-controlled inputs into environment variables and securing MCP lockdown
handling. This change updates the way safe-inputs and MCP lockdown values are
passed to runtime steps (moved to `env:` blocks) and simplifies lockdown value
conversion. Affects several workflows and related MCP renderer/server code.

Fixes: githubnext/gh-aw#9124

