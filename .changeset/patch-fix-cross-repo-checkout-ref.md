---
"gh-aw": patch
---

Fixed `create-pull-request` with `target-repo`: the `safe_outputs` checkout step now uses a cross-repo-safe ref expression (omitting `github.ref_name`) to avoid failures when the triggering branch does not exist in the target repository.
