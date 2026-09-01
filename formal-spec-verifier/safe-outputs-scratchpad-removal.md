# Formal Notes: safe-outputs-scratchpad-removal.md

**Last formalized**: 2026-09-01-15-46-43
**Notation**: TLA+ / Z3-style guard conjunction / F*
**Issue**: created via safeoutputs create_issue (number assigned post-run)

## Predicates

| ID | Predicate | Description |
|---|---|---|
| P1 | `formalCanonicalPathWellDefined` | Canonical path is non-empty and distinct from deprecated path |
| P2 | `formalNoReferencesToDeprecatedAfterMigration` | After migration, no reference targets the deprecated scratchpad path |
| P3 | `formalDeletionAllowedAnytimeUpToDeadline` | Deletion permitted any date on/before 2026-09-21 |
| P4 | `formalDeletionRequiredByDeadline` | If today > deadline, deprecated file must not exist |
| P5 | `formalNoDanglingReferencesAfterDeletion` | Once deleted, no reference may still resolve to deprecated path |
| P6 | `formalMigrationPrecedesDeletion` | Deletion complete implies migration was already complete |
| P7 | `formalChecklistItemsMonotonic` | Verified "no dangling references" state cannot regress |
| P8 | `formalDeadlineWellFormed` | Deadline constant parses to 2026-09-21 |
| P9 | `formalIdempotentReferenceReplacement` | Replacing an already-canonical reference is a no-op |
| P10 | `formalReferenceCountMonotonicDecreasing` | Migration never increases the count of deprecated-path references (new 2026-09-01) |
| P11 | `formalDeadlineBoundaryInclusive` | On the exact deadline date, deletion is not yet force-required (new 2026-09-01) |

## Key Invariants

- The deprecated `scratchpad/safe-outputs-specification.md` must be removed on or before 2026-09-21.
- All repository references (doc-site nav, workflow files, internal links) must be migrated to the canonical path before deletion.
- Deletion of the deprecated file implies migration already completed (no orphaned links).
- After deletion, no reference may still resolve to the deleted scratchpad path.
- Reference replacement must be idempotent (already-canonical refs are untouched).

## Edge Cases Identified

- Exact deadline date (2026-09-21) does not yet force deletion; only `today > deadline` triggers the requirement.
- Overdue state (today > deadline, file still exists) must be detected as a violation.
- Empty reference set trivially satisfies "no references to deprecated path" (vacuous truth).
- Malformed/empty-target reference must not falsely match the deprecated path string.

## Notes for Future Runs

- No production implementation exists yet for scratchpad-removal checking in `pkg/workflow/` or `pkg/cli/`;
  all helpers in the generated test file are stubs marked `// stub — replace with real implementation`.
- Re-confirmed on 2026-09-01 via `grep -rl "scratchpad/safe-outputs-specification"` that the deprecated
  file is STILL referenced from `scratchpad/agents/hierarchical-agents-quickstart.md` and
  `scratchpad/github-mcp-access-control-specification.md` — migration work (checklist item 1) remains
  incomplete after two rotation cycles.
- Extended the model this run with P10 (reference-count monotonic decrease during migration) and P11
  (deadline-boundary inclusivity — being exactly on 2026-09-21 does not yet force deletion). A future run
  could formalize the actual link-scanning tool (if one gets built) rather than relying on the stub
  `reference`/`migrateReferences` model.
- This spec is a lightweight, single-checklist document (9 lines) — much smaller than other specs in
  rotation; formalization intentionally kept compact (now 11 predicates, up from 9).
