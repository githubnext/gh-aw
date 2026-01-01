---
"gh-aw": patch
---

Refactor system prompts to be file-based under `actions/setup/md/` and
update runtime to read prompts from `/tmp/gh-aw/prompts/` instead of
embedding them in the Go binary. This is an internal refactor that
moves prompt content to runtime-managed markdown files and updates the
setup script and prompt generation logic accordingly.

