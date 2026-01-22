---
"gh-aw": patch
---

Add discussion interaction to smoke workflows; deprecate `discussion` flag and add a codemod to remove it.

Smoke workflows now query the latest discussion and post comments to both the discussion and PR to validate discussion functionality. The compiler now serializes the legacy `discussion` boolean into safe-outputs handler configs for compatibility. A codemod was added to automatically remove the deprecated `safe-outputs.add-comment.discussion: true` field from workflows.

---

