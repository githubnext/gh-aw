---
"gh-aw": patch
---

Fixed regression (introduced in v0.80.6) where the compiler inlined `${{ github.event.inputs.* }}` expressions directly into the shell command inside the generated "Configure Git credentials" step for multi-repo (`target-repo`) workflows. The expression is now passed through a dedicated env var (`GH_AW_SUBREPO_N`) so the `git remote set-url` command never contains a raw `${{ }}` expression, preventing the template-injection scanner from rejecting its own generated code.
