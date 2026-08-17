# Formal Notes: safe-outputs-scratchpad-removal.md

**Last formalized**: 2026-08-17-15-37-40
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
- Confirmed via `grep -rl "scratchpad/safe-outputs-specification"` that the deprecated file is still
  referenced from `scratchpad/agents/hierarchical-agents-quickstart.md` and
  `scratchpad/github-mcp-access-control-specification.md` as of this run — migration work (checklist
  item 1) has not yet been completed in the repository.
- A follow-up run could formalize the actual link-scanning tool (if one gets built) rather than relying
  on the stub `reference`/`migrateReferences` model.
- This spec is a lightweight, single-checklist document (9 lines) — much smaller than other specs in
  rotation; formalization intentionally kept compact (9 predicates) to match its scope.
