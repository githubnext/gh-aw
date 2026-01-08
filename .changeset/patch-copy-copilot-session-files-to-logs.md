---
"gh-aw": patch
---

Copy Copilot session state files (`~/.copilot/session-state/*.jsonl`) to
`/tmp/gh-aw/sandbox/agent/logs/` before secret redaction so they are included
in workflow artifacts and available for debugging.

