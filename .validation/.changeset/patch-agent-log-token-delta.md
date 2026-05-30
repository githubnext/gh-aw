---
"gh-aw": patch
---

Improved agent log rendering by displaying the effective-token delta (ΔET) for each MCP tool call. The delta is computed by correlating tool call timestamps from `rpc-messages.jsonl` with LLM API call data in `token-usage.jsonl`, showing how much each tool call result contributed to the growing context window.

- Go CLI: adds a "Tool Call Timeline (Effective Token Δ)" table to the MCP tool usage section when delta data is available
- GitHub Actions step summary: adds a ΔET column to the REQUEST table in the MCP Gateway Activity section
