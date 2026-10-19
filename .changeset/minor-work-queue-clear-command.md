---
"gh-aw": minor
---

Add `gh aw work-queue clear --reason CODE` with an authenticated read-only preview and atomic cancellation of unfinished snapshot Work. Preserve completed Work, unresolved delivery barriers, audit history, admission identities, fairness debt, and native reservations. Add `--hard` for explicitly resetting a paused, fully settled ledger while retaining prior Git history and warning about identity/debt loss. Both modes require maintain/admin permission, show a queue preview and require interactive confirmation before mutation, refuse CI, and never stop native runs or bypass unresolved effects.
