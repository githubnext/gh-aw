---
"gh-aw": patch
---

Add `hide-older-comments` boolean and `allowed-reasons` array fields to `add-comment` and `hide-comment` safe outputs; includes parsing, JavaScript hiding logic, tests, and documentation updates.

This change adds support for hiding older comments from the same workflow (identified by workflow-id) and allows restricting which hide reasons are permitted via the `allowed-reasons` field. Backwards compatible: if `allowed-reasons` is omitted, all reasons are allowed.

