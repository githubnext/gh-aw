---
"gh-aw": patch
---

Added the internal `features: samples: true` frontmatter flag, a per-workflow opt-in to the deterministic safe-outputs samples replay driver that until now was only reachable through the hidden `gh aw compile --use-samples` flag. Workflows designed around `safe-outputs.*.samples` now compile to their samples-mode lock file under a plain `gh aw compile`, so the committed lock file stays in sync with `make recompile`. The repository's own `smoke-ci` workflow uses this to replace its custom `engine.command` bash script with declarative samples.
