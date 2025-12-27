---
"gh-aw": patch
---

Fix SC2155: Separate export declaration from command substitution in workflows

Split variable assignment from `export PATH=...$(...)` into a separate
assignment and `export` so that the exit status of the command substitution
is not masked. This resolves 31 shellcheck SC2155 warnings related to PATH
setup in generated workflows and keeps `claude_engine.go` and
`codex_engine.go` consistent by using the `pathSetup` variable pattern.

Fixes: githubnext/gh-aw#7897

