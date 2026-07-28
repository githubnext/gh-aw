# ADR-48455: Split token_usage.go Monolith into Focused Modules

**Date**: 2026-07-28
**Status**: Draft
**Deciders**: Unknown (automated refactor by copilot-swe-agent)

---

### Context

`pkg/cli/token_usage.go` had grown to 1,141 lines by mixing unrelated concerns in a single file: data type definitions, file discovery logic, JSONL parsing, token usage analysis, API proxy steering event counting, and sub-agent model attribution. This made the file hard to navigate, understand, and test in isolation. The Go convention of grouping code into focused, single-responsibility files was not being followed, and the growing complexity increased the risk of unintended coupling between these concerns.

### Decision

We will split `pkg/cli/token_usage.go` into six focused modules, each owning a single responsibility: `token_usage_types.go` (structs/constants), `token_usage_find.go` (file discovery), `token_usage_parse.go` (JSONL parsing), `token_usage_analyze.go` (aggregation logic), `token_usage_steering.go` (steering event counting), and `token_usage_subagent.go` (sub-agent attribution). No logic changes, no API surface changes — this is a pure structural reorganization within the same `cli` package.

### Alternatives Considered

#### Alternative 1: Keep the monolithic file

Leave `token_usage.go` as-is and continue adding to it. This avoids all churn and merge risk, but the file will continue to grow, making navigation and testing harder over time. Rejected because the 1,141-line size already exceeds a navigable threshold and no natural stopping point exists.

#### Alternative 2: Extract to a dedicated sub-package (`pkg/cli/tokenusage/`)

Move all token usage logic into its own sub-package. This would provide stronger encapsulation via Go's package visibility rules. Rejected for this PR because it would change import paths, affect the public API surface, and is a larger structural change that should be a separate, explicit decision with its own ADR. The within-package file split is a lower-risk first step.

#### Alternative 3: Extract only types to a shared package

Move `TokenUsageSummary` and related types to a shared `pkg/tokenusage` package to break potential import cycles elsewhere. Rejected because no import cycles currently exist, and the types are only consumed within the `cli` package; moving them would add premature abstraction.

### Consequences

#### Positive
- Each file is now under ~400 lines and has a clear, single responsibility, reducing cognitive load when navigating or modifying token usage logic.
- Focused files make it easier to write targeted unit tests for individual concerns (e.g., testing file discovery independently of parsing).
- The split establishes a clear convention for future growth: new concerns get their own file rather than appending to a monolith.

#### Negative
- The number of files in `pkg/cli/` increases by five, which may make directory listings noisier.
- Future refactors that need to move token usage logic to its own sub-package will still need to happen; this split does not resolve the underlying question of whether this logic belongs in `pkg/cli/` long-term.

#### Neutral
- All existing tests pass unchanged — the split is transparent to callers and tests.
- The `pkg/cli` package boundary is preserved; no import path changes are required.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
