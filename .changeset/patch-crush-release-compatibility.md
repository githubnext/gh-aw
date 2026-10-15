---
"gh-aw": patch
---

Update the unsupported Crush sample to 0.98.1. Route GPT-5 and newer models through the native OpenAI Responses provider while retaining Chat Completions for other non-Anthropic models, and reject reflected endpoints belonging to a different provider. Preserve observed plaintext and saved unified-session records without fabricating tool calls, turn counts, or token usage. Add offline coverage for provider routing, private MCP configuration, command arguments, failure reporting, and session normalization.
