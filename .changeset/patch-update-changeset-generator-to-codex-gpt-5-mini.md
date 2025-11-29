---
"gh-aw": patch
---

Update the changeset generator workflow to use the `codex` engine with the
`gpt-5-mini` model. Add `strict: false` and remove the `firewall: true` network
setting to accommodate the codex engine's network behavior.

This is an internal tooling change.

