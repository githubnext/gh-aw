---
"gh-aw": patch
---

Upgrade actions/upload-artifact to v6.0.0 across workflows and recompiled lock files.

This adds the `v6.0.0` pin to `.github/aw/actions-lock.json` and updates compiled
workflows to reference `actions/upload-artifact@v6.0.0` (replacing v5.0.0 references).

This is an internal tooling change (workflow lock files) and does not affect runtime code.

