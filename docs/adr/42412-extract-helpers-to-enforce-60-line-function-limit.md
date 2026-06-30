# ADR-42412: Extract Helpers to Enforce the 60-Line Function Length Limit

**Date**: 2026-06-30
**Status**: Draft
**Deciders**: pelikhan, copilot-swe-agent

---

### Context

The codebase enforces a custom `largefunc` lint rule capping functions at 60 lines. Over time, 13 functions in `pkg/workflow` and `pkg/cli` grew well beyond this limit — the largest, `buildMaintenanceWorkflowYAML`, reached 960 lines, and others such as `extractSafeOutputsConfig` (752 lines), `buildMainJob` (420 lines), and `renderConsole` (310 lines) followed. Long functions in these packages reduce readability, impede focused unit testing of individual logical steps, and indicate that unrelated concerns have been co-located. The lint rule was already in place; the violation count had accumulated and needed to be cleared.

### Decision

We will fix all 13 `largefunc` violations by extracting cohesive logic blocks from each offending function into named helper functions, leaving the original function as a thin, readable dispatcher that orchestrates the helpers in sequence. The approach is applied consistently across both packages with no behavioral changes: existing tests must pass unchanged, and no public API surfaces are altered.

### Alternatives Considered

#### Alternative 1: Raise or Disable the Lint Rule Threshold

The 60-line limit could be relaxed (e.g., to 150 or 300 lines) or the lint check could be ignored for specific files. This would require no code changes and avoids the risk of poorly-bounded helper extraction. It was rejected because it would normalize large functions going forward, erode the investment in the lint rule, and leave the underlying readability and testability problems in place.

#### Alternative 2: Restructure into Separate Packages or Files

Each large function could be decomposed by moving logical groups into separate Go packages or files with their own exported API. This provides stronger encapsulation and clearer ownership boundaries. It was rejected because the scope would far exceed a lint-compliance fix — it would require API changes, import graph updates, and coordinated testing across callers — making it inappropriate as a standalone refactor with no behavioral intent.

### Consequences

#### Positive
- All 13 `largefunc` lint violations are eliminated, unblocking lint-clean CI runs.
- Each extracted helper is small, clearly named, and individually testable without the full parent-function setup.
- The dispatcher functions read as high-level summaries of their logic, improving onboarding for new contributors.

#### Negative
- The total file count increases (new `*_helpers.go` and `*_jobs.go` files are added), adding navigation surface.
- Helpers extracted purely to satisfy the line limit may have weaker cohesion than helpers designed from scratch; future refactors may need to re-evaluate boundaries.

#### Neutral
- No behavioral changes are made; all existing tests are expected to pass without modification.
- The refactoring pattern (dispatcher + helpers) is consistent throughout, but it is not formalized as a project-wide convention in this PR.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
