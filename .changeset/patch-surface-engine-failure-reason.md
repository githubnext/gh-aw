---
"gh-aw": patch
---

Improve failure reporting when the AI engine exits before producing `agent_output.json` by avoiding downstream ENOENT noise and surfacing terminal error details from `agent-stdio.log` in conclusion comments and issues.
