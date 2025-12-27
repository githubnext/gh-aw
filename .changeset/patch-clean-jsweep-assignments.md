---
"gh-aw": patch
---

Cleaned and modernized three github-script JavaScript actions:
- `assign_to_user.cjs`
- `check_command_position.cjs`
- `check_membership.cjs`

Refactored logic to use modern ES6+ patterns, improved readability, and
added comprehensive tests for `assign_to_user.cjs`.

This is an internal code quality change and test addition; no public API
behaviour is changed.

