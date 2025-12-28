---
"gh-aw": patch
---

Removed redundant syncing of JavaScript and shell scripts from
`actions/setup/` into `pkg/workflow/{js,sh}` and converted inline
JavaScript to a `require()`-based runtime-loading pattern. This reduces
binary size, eliminates duplicated generated files, consolidates setup
script copying into `actions/setup/setup.sh`, and updates workflow
script loading and tests to the new runtime behavior.

See PR #7654 for details.

