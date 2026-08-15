# ADR-52975: Consolidate runs-on Normalization Helpers into runs_on_snippet.go

**Date**: 2026-08-15
**Status**: Draft
**Deciders**: pelikhan (via copilot-swe-agent, PR #52975)

---

### Context

The `pkg/workflow` package contained three separate locations that independently implemented overlapping `runs-on` parsing and rendering logic:

- `repo_config.go` defined `RunsOnValue`, `UnmarshalJSON`, `toRunsOnValue`, `isRunsOnArrayValue`, and `FormatRunsOn`
- `safe_jobs.go` contained inline branching logic for array-vs-scalar `runs-on` rendering
- `runs_on_snippet.go` already held `renderRunsOnSnippet`/`normalizeRunsOnSnippet` helpers

This duplication meant that a bug fix to runner-shape handling could be applied in one parser while silently missing the others. The `runs-on` field accepts both a single string and an array of strings (e.g., for self-hosted runners with labels), and the normalization and YAML rendering must behave consistently across all callers.

### Decision

We will consolidate all `runs-on` type definitions and helpers into `runs_on_snippet.go`, which already houses the core snippet rendering logic. Specifically:

- `RunsOnValue`, `UnmarshalJSON`, `toRunsOnValue`, `isRunsOnArrayValue`, and `FormatRunsOn` are moved from `repo_config.go` into `runs_on_snippet.go`.
- A new `formatSafeJobRunsOn` helper is added to `runs_on_snippet.go`, replacing the inline array-vs-scalar branching in `safe_jobs.go`'s job-building loop.
- `safe_jobs.go` is simplified to a single call to `formatSafeJobRunsOn`.

This is a pure code-organization refactor with no change to supported `runs-on` shapes or rendered output.

### Alternatives Considered

#### Alternative 1: Keep Duplication As-Is

Leave `RunsOnValue` and its helpers in `repo_config.go` and retain the inline branching in `safe_jobs.go`. Simple in the short term and zero risk of behavioral regression, but perpetuates the maintenance hazard: the next `runs-on` bug fix must be applied in multiple places, and there is no structural enforcement ensuring all parsers stay in sync.

#### Alternative 2: Create a Dedicated `runs_on.go` File

Move all `runs-on` types and helpers into a new `pkg/workflow/runs_on.go` file rather than expanding `runs_on_snippet.go`. This would produce a cleaner name-to-responsibility mapping. The trade-off is an additional file that must be discovered, and `runs_on_snippet.go`'s snippet rendering helpers would remain separated from the type that drives them. Given the functions are tightly coupled (they all operate on `RunsOnValue` and produce YAML fragments), co-location in one file is preferred over splitting across two.

### Consequences

#### Positive
- Single source of truth for all `runs-on` normalization and YAML rendering; future bug fixes or new runner-shape support apply uniformly across `aw.json` and `safe-outputs.jobs` parsing.
- `safe_jobs.go` call site is reduced from 13 lines of branching logic to 1 line, improving readability and reducing cognitive overhead for future maintainers.

#### Negative
- `runs_on_snippet.go` now covers a broader scope than its filename implies (it holds the `RunsOnValue` type, JSON unmarshaling, and YAML rendering helpers, not just snippet generation). Readers may be surprised to find the type definition there rather than in a file named `runs_on.go`.
- As a pure refactor, behavioral parity must be verified by existing tests. Any gap in test coverage of `runs-on` edge cases (empty arrays, single empty-string elements, multi-label arrays) could mask an unintended regression introduced during the move.

#### Neutral
- The `encoding/json` and `fmt` imports are added to `runs_on_snippet.go` (previously only in `repo_config.go`) as a direct consequence of moving the type and its JSON unmarshaler.
- No public API surface changes: `RunsOnValue`, `FormatRunsOn`, and related helpers remain exported at the same package level.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
