---
"gh-aw": patch
---

Remove redundant syncing of JavaScript and shell scripts from the Go binary.

- Removed embedding and sync targets for `actions/setup/{js,sh}` scripts.
- Converted inline JavaScript to use `require()` loading at runtime.
- Simplified setup/copy logic and updated tests to use external scripts.

This is an internal/tooling change and does not change the CLI public API.

