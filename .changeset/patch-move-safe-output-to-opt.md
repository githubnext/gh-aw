---
"gh-aw": patch
---

Move the safe-output storage from `/tmp` to `/opt` and update the agent intake and redaction scripts to read from the new location.

This updates the default `GH_AW_SAFE_OUTPUTS` path and related JavaScript intake/redaction modules so the agent reads the safe-output `.jsonl` from `/opt/gh-aw/safeoutputs/outputs.jsonl` (read-only from the container) while the MCP server retains write access.
