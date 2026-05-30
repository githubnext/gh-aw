---
"gh-aw": patch
---

Fix `gh aw add` incorrectly resolving body-level `{{#import shared/X.md}}` from the repo root instead of the workflow file's directory (`.github/workflows/`). Also preserve body-level imports as local references when the target file already exists in the consuming repository, matching the behaviour already implemented for `gh aw update`.
