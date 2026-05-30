---
"gh-aw": patch
---

Make `setup-gh-aw` idempotent when `gh-aw` is already installed, so existing `gh aw` commands are reused instead of failing on extension conflicts.
