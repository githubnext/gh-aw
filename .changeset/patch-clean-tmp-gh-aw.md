---
"gh-aw": patch
---

Ensure `actions/setup` removes `/tmp/gh-aw` before recreating it so each setup run starts from a clean temporary directory.
