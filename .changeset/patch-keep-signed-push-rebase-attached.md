---
"gh-aw": patch
---

Keep the current branch attached when rebasing stale commits before signed delivery. If GraphQL signing is unavailable, the permitted git push fallback now pushes the rebased branch rather than its stale pre-rebase tip, avoiding a spurious non-fast-forward rejection. Explicitly disabled fallback and signing policies are unchanged.
