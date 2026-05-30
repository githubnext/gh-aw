---
"gh-aw": major
---

Decouple the status comment from the ordinary ai-reaction emoji so they must each be enabled explicitly (e.g., add `status-comment: true` if you still need the started/completed comment). This fixes github/gh-aw#15831.

**⚠️ Breaking Change**: The status comment (started/completed notification) is no longer enabled by default. Previously it was implicitly enabled alongside the ai-reaction emoji; now both must be enabled explicitly.

**Migration guide:**
- If your workflows rely on the automatic status comment, add `status-comment: true` explicitly to your workflow frontmatter
- Example:
  ```yaml
  # Before (status comment was implicit)
  # (no configuration needed)

  # After (must be explicit)
  status-comment: true
  ```
- Workflows that did not rely on status comments require no changes
