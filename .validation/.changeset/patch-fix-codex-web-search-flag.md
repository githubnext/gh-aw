---
"gh-aw": patch
---

Fixed Codex command generation to use `-c web_search="disabled"` instead of the invalid `--no-search` flag, and stopped emitting a non-existent `--search` flag when web search is enabled.
