---
"gh-aw": patch
---

Split the large `compiler_safe_outputs_consolidated.go` file into six
domain-focused modules (core, issues, prs, discussions, shared,
specialized) to improve maintainability and reduce file size. This
refactor reduces the largest file size and lowers merge conflict risk.

Fixes githubnext/gh-aw#7250

