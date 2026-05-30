---
"gh-aw": minor
---

Enable `reaction: eyes` and `status-comment: true` by default when `slash_command` or `label_command` triggers are used. Both can be disabled explicitly with `reaction: none` and `status-comment: false`.

Previously, `reaction: eyes` was only auto-enabled for `slash_command` workflows, and `status-comment` always required explicit opt-in. Now both defaults apply to `label_command` workflows as well, and `status-comment: true` is the default for both command trigger types.
