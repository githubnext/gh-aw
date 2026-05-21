---
"gh-aw": patch
---

Fixed the Codex engine to emit `codex exec${VAR:+ --model "$VAR"}` instead of `codex ${VAR:+--model "$VAR" }exec` so the model flag is correctly positioned after the `exec` subcommand. Previously the model parameter was placed before `exec`, causing Codex CLI to ignore it and silently fall back to the default model.
