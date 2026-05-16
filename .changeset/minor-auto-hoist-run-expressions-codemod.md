---
"gh-aw": minor
---

Extend `steps-run-secrets-to-env` codemod to hoist **all** `${{ ... }}` expressions from `run:` blocks — not just secrets, `env.*`, and `github.token`. Arbitrary expressions such as `github.repository`, `github.event.issue.title`, `inputs.*`, and `steps.*.outputs.*` now receive `EXPR_*` step-level `env:` bindings. PowerShell steps (`shell: pwsh` / `shell: powershell`) receive `$env:VARNAME` syntax. This closes the gap that previously required the separate `auto-hoist-run-expressions` codemod.
