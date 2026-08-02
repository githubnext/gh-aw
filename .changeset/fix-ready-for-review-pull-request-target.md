---
"gh-aw": patch
---

Fixed `ready_for_review` activity type not being accepted in `on.pull_request_target.types`. The schema now allows the same activity types for `pull_request_target` as for `pull_request`, including `ready_for_review`.
