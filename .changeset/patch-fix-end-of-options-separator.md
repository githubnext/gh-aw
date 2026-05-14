---
"gh-aw": patch
---

Fix prompt handling in `resolveClaudePromptFileArgs` by inserting `--` before the prompt content so Claude Code does not misinterpret the prompt as an additional `--mcp-config` path and fail with `ENAMETOOLONG`.
