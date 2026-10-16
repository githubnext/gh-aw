---
"gh-aw": patch
---

Update the default OpenAI Codex CLI from 0.162.1 to 0.163.0, including upstream command-output draining and proxy hardening fixes. The exec CLI and JSONL event contracts are unchanged; existing configuration, MCP connections, authentication, and harness behavior are preserved.

Stop harness retries when the AWF proxy rejects a model with unknown AI credit pricing, preserving the failed result instead of repeating a deterministic configuration error.
