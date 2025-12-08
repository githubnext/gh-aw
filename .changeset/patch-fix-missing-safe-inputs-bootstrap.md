---
"gh-aw": patch
---

Add missing agent bootstrap `safe_inputs_bootstrap.cjs` support.

The pull request fixes a bug where the embedded safe inputs bootstrap script
was not exposed via a getter and therefore not written to the
`/tmp/gh-aw/safe-inputs/` directory. This change adds the getter and the
file-writing step so workflows depending on `safe_inputs_bootstrap.cjs` can
load it correctly.

