---
"gh-aw": patch
---

Fix `--name` flag to only apply to the first workflow when adding multiple workflows with `gh aw add workflow1 workflow2 ...`. Previously the name was applied to all workflows, causing each to overwrite the previous one.
