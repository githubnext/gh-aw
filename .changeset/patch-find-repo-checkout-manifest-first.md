---
"gh-aw": patch
---

Fix safe-output handlers failing to locate side-repo checkouts when a "Configure Git credentials" step rewrote `remote.origin.url` at the workspace root. `findRepoCheckout` and `buildRepoCheckoutMap` now consult the compiler-written checkout manifest first, falling back to the existing `git config` scan only when the manifest has no entry for the requested slug.
