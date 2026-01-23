---
"gh-aw": patch
---

Add discussion interaction to smoke workflows and serialize the discussion
flag in safe-outputs handler config.

Smoke workflows now select a random discussion and post thematic comments to
validate discussion comment functionality. The compiler now emits the
`"discussion": true` flag in `GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG` when a
workflow requests discussion output, and lock files include `discussions: write`
permission where applicable.

