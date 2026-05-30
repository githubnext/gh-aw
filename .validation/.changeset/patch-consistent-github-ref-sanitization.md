---
"gh-aw": patch
---

make sanitize_content_core.cjs use the shared repo helper so wildcard and "repo" patterns work when neutralizing GitHub references, and exercise the new behavior in sanitize_content.test.cjs
