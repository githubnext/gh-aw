---
"gh-aw": patch
---

`gh aw upgrade` now updates `uses:` references in workflow source `.md` files alongside `actions-lock.json`, so compiled `.lock.yml` files always use the new action version after an upgrade.
