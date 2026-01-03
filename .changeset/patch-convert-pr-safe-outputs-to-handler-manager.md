---
"gh-aw": patch
---

Convert PR-related safe outputs and `hide-comment` to the handler manager architecture.

This change is an internal refactor: five PR-related safe outputs plus the `hide-comment`
safe output were converted to use the handler factory/manager pattern, legacy script getter
infrastructure was removed, and TypeScript/Go formatting and linting were applied. There are
no user-facing API changes.

All changes are internal and qualify as a patch-level release.

