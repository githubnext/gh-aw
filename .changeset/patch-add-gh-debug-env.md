---
"gh-aw": patch
---

Add `GH_DEBUG=1` to the shared `gh` safe-input tool configuration so
that `gh` commands executed via the `safeinputs-gh` tool run with
verbose debugging enabled.

This is an internal/tooling change that affects workflow execution
verbosity only.

