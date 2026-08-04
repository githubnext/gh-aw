---
"gh-aw": major
---

Remove the experimental `opencode` engine.

Workflows using `engine: opencode` or `engine.id: opencode` no longer compile
or validate.

**Migration:** Change the workflow to use `copilot`, `claude`, `codex`, `gemini`,
`antigravity`, or `pi`. The runner no longer restores `opencode.jsonc` or
`.opencode/` configuration.
