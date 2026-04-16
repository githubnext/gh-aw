---
"gh-aw": patch
---

Fix shared MCP bearer token bypass in safeoutputs write-sink. The MCP gateway API key in `mcp-servers.json` was shared across all servers (read and write), allowing bash subprocesses to read it and use it to call the safeoutputs write-sink via the gateway, bypassing declared read-only permissions. Now each engine converter (Claude, Copilot, Gemini, Codex) gives the safeoutputs server a direct connection with `${GH_AW_SAFE_OUTPUTS_API_KEY}` as a literal env var reference rather than the shared gateway key. The LD_PRELOAD one-shot token library prevents bash subprocesses from expanding this reference from the environment, eliminating the file-based credential bypass.
