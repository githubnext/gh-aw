---
"gh-aw": patch
---

Add `hide-older-comments` and `allowed-reasons` fields to `add-comment` and `hide-comment` safe outputs.

This change:
- Adds `hide-older-comments` boolean to `add-comment` to automatically minimize previous comments from the same workflow.
- Adds `allowed-reasons` array to `add-comment` and `hide-comment` with restricted lowercase values: `spam`, `abuse`, `off_topic`, `outdated`, `resolved`.
- Implements parsing, JavaScript hiding logic, tests, and documentation updates.

These are non-breaking, backwards-compatible additions.

