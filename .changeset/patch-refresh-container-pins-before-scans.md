---
"gh-aw": patch
---

Add a `compile --refresh-container-pins` option and use it in the daily container image security scan so rebuilt container tags are re-resolved before Syft, Grype, and Grant run.
