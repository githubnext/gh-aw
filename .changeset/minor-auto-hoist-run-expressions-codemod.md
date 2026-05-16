---
"gh-aw": minor
---

Add `auto-hoist-run-expressions` codemod that rewrites **all** `${{ ... }}` expressions inside `run:` blocks to `$VARNAME` shell references and inserts `EXPR_*` step-level `env:` bindings. This closes the gap left by `steps-run-secrets-to-env` (which only handles `secrets.*`, `env.*`, and `github.token`) by covering arbitrary expressions such as `github.repository`, `github.event.issue.title`, `inputs.*`, and `steps.*.outputs.*`. PowerShell steps (`shell: pwsh` / `shell: powershell`) receive `$env:VARNAME` syntax instead of `$VARNAME`.
