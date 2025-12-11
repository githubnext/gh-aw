---
"gh-aw": patch
---

Fix firewall logs not printing due to incorrect directory path. The firewall
log parser was reading from a sanitized workflow-specific directory but the
logs are written to a fixed sandbox path. This change documents the bugfix
that updates the parser to read from `/tmp/gh-aw/sandbox/firewall/logs/` and
removes the unnecessary workflow name sanitization.

