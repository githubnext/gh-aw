---
"gh-aw": patch
---

Add discussion interaction to smoke workflows; deprecate the `discussion` flag and
add a codemod to remove it. Smoke workflows now query discussions and post
comments to both discussions and PRs to validate discussion functionality.

The compiler no longer emits a `discussion` boolean flag in compiled handler
configs; the `add_comment` handler auto-detects target type or accepts a
`discussion_number` parameter. A codemod `add-comment-discussion-removal` is
available via `gh aw fix --write` to remove the deprecated field from workflows.

