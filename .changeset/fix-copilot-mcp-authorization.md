---
"gh-aw": patch
---

Fix Copilot MCP configuration: convert 0.0.0.0 to 127.0.0.1 in URLs. The GitHub Copilot CLI requires proper localhost addresses, so the converter script now replaces 0.0.0.0 with 127.0.0.1 in MCP server URLs for compatibility.
