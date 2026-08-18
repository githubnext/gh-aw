# ADR-53818: Split Oversized Test Files by Functional Boundary

**Date**: 2026-08-18
**Status**: Draft
**Deciders**: pelikhan, copilot-swe-agent

---

### Context

The `developer-code-organization` skill mandates files of 100–500 lines (800 lines only for core infrastructure). Two test files significantly exceeded this ceiling: `pkg/workflow/compiler_jobs_test.go` (4,511 lines / 84 test functions) and `compiler_safe_outputs_config_test.go` (3,837 lines / 48 functions). Both files group tests by broad subsystem rather than by narrower functional concern, making them merge-conflict hotspots and slow for humans and LSP tooling to navigate. The repository has a documented precedent for this kind of split: `frontmatter.go` was previously decomposed into five focused files and is cited in the skill itself.

### Decision

We will split both oversized test files along functional boundaries — `compiler_jobs_test.go` into 11 focused files (by job type: frontmatter extraction, built-in jobs, step ordering, safe outputs, dependencies, memory, edge cases, custom jobs, custom fields, state push, and engine-env handling) and `compiler_safe_outputs_config_test.go` into 8 focused files (by safe-output concern: handler config fields, target repo and patch limits, auto-enabled handlers, staged mode, protected files and checkout mapping, PR policy and fork-backed PRs, and failure reporting). No test logic is changed; this is a pure file-layout reorganization. One residual `compiler_safe_outputs_config_test.go` (965 lines) is kept intact because it contains a single table-driven function (`TestAddHandlerManagerConfigEnvVar`) that cannot be split without duplicating its runner body.

### Alternatives Considered

#### Alternative 1: Keep files as-is and accept the size violation

Leave `compiler_jobs_test.go` and `compiler_safe_outputs_config_test.go` unchanged. This avoids any migration cost and requires no reviewer attention to a pure-refactor PR. However, it perpetuates merge conflicts on two of the highest-traffic test files and worsens LSP responsiveness over time. It also sets a precedent that the documented file-size convention is advisory rather than enforced.

#### Alternative 2: Use test suites to group related tests within a single file

Introduce `testing.Suite` or similar sub-grouping so related tests share a logical namespace while the file count stays the same. This addresses navigability somewhat but does not reduce merge-conflict risk (the file is still one large target for concurrent edits) and does not comply with the line-count ceiling enforced by the skill.

### Consequences

#### Positive
- Smaller files (most under 700 lines) comply with the `developer-code-organization` ceiling and reduce merge-conflict risk on high-traffic paths.
- Editors and LSP tooling load focused files faster, improving daily developer experience.
- Each new file's name and scope make it straightforward to locate the right test without full-file search.
- Follows the established `frontmatter.go` precedent, keeping the codebase internally consistent.

#### Negative
- Import lists must be independently maintained in each of the 19 new files instead of a single shared block.
- Obtaining a holistic view of a component's test coverage now requires reading multiple files rather than scrolling one.
- One residual file (`compiler_safe_outputs_config_test.go`, 965 lines) still exceeds the 500-line guideline because its single large table-driven function cannot be split without duplicating its runner body.

#### Neutral
- All 84 compiler-jobs test functions and 48 safe-outputs config test functions are preserved verbatim; no test logic is altered.
- Doc comments that fell on a split boundary were carried to the file containing their associated function.
- Each new file receives only the imports it actually uses, derived from the original shared import block.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
