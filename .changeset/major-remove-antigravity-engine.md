---
"gh-aw": major
---

Remove the experimental `antigravity` engine.

Workflows using `engine: antigravity` or `engine.id: antigravity` no longer
compile or validate.

**Migration:** Change the workflow to use `copilot`, `claude`, `codex`,
`gemini`, or `pi`. The runner no longer restores `ANTIGRAVITY.md` or
`.antigravity/` configuration.
