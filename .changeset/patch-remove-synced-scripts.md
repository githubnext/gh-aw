---
"gh-aw": patch
---

Remove redundant JS/shell script syncing from `actions/setup` to `pkg/workflow`.

Scripts previously copied into `pkg/workflow/js` and `pkg/workflow/sh` are no longer required because `actions/setup/index.js` bundles them. This changeset documents the build-system and packaging cleanup (removed sync targets, deleted generated files, and adjusted embed directives).
