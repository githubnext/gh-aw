---
"gh-aw": patch
---

Update Codex remote GitHub MCP configuration to new streamable HTTP format

Updated the Codex engine's remote GitHub MCP server configuration to use the new streamable HTTP format with `bearer_token_env_var` instead of deprecated HTTP headers. This includes adding the `experimental_use_rmcp_client` flag, using the `/mcp-readonly/` endpoint for read-only mode, and standardizing on `GH_AW_GITHUB_TOKEN` across workflows. The configuration now aligns with OpenAI Codex documentation requirements.
