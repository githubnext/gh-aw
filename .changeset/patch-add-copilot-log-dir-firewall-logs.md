---
"gh-aw": patch
---

Add --log-dir to Copilot sandbox args and use the sandbox folder
structure for firewall logs so logs are written to the expected
locations for parsing and analysis.

This fixes firewall mode where Copilot logs were not being written to
/tmp/gh-aw/sandbox/agent/logs/ and updates firewall log collection
paths and artifact naming.
