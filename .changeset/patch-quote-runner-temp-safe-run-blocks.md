---
"gh-aw": patch
---

Fix compiler-generated shell `run:` commands to quote `${RUNNER_TEMP}` paths and switch shared workflow `run:` template interpolation from `${{ github.repository }}` to `$GITHUB_REPOSITORY` to avoid shell expansion and template-injection issues.
