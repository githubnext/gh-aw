---
"gh-aw": patch
---

Update awf to v0.6.0, add the `--proxy-logs-dir` flag to direct firewall proxy
logs to `/tmp/gh-aw/sandbox/firewall/logs`, and remove the post-run `find`
step that searched for agent log directories.

This is an internal/tooling change (updated dependency and script behavior).

