---
"gh-aw": patch
---

Move safe-output storage from `/tmp` to `/opt` and update the agent intake and secret-redaction
scripts to read from the new path `/opt/gh-aw/safeoutputs/outputs.jsonl`. This keeps the file writable
by the MCP server while making it read-only inside the agent container.
