---
"gh-aw": patch
---

Add support for `imports.plugins` in workflow frontmatter, with merge/dedup across shared imports and GitHub tree URL normalization (auto-converted to `OWNER/REPO:PATH` spec). Supports all Copilot CLI plugin spec formats: `plugin@marketplace`, `OWNER/REPO`, `OWNER/REPO:PATH/TO/PLUGIN`, and Git URLs.
