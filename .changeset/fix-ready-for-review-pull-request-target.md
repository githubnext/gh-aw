---
"gh-aw": patch
---

Fixed `ready_for_review` activity type not being accepted in `on.pull_request_target.types`. The schema now allows `ready_for_review` for `pull_request_target`, matching `pull_request` support for that activity type.
