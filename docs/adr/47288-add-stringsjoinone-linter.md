# ADR-47288: Add `stringsjoinone` Custom Go Analyzer

**Date**: 2026-07-22
**Status**: Draft
**Deciders**: pelikhan, copilot-swe-agent

---

### Context

The `pkg/linters` package maintains a suite of custom `go/analysis` static analyzers that enforce Go code quality patterns specific to this codebase. A recurring antipattern exists where `strings.Join([]string{s}, sep)` is called with a single-element slice literal: the separator argument is irrelevant (no join ever occurs) and the call is semantically identical to `s` alone. This pattern misleads readers into believing the separator matters, adds unnecessary allocation, and is not caught by existing linters in the standard toolchain or in the current registry. Issue #47122 identified this gap.

### Decision

We will add a new `stringsjoinone` analyzer to `pkg/linters/stringsjoinone/` using the established `go/analysis` pass framework. The analyzer detects calls of the form `strings.Join([]string{expr}, sep)` where the first argument is a `[]string` composite literal with exactly one element, reports a diagnostic with an explanatory message, and provides a `SuggestedFix` that replaces the entire call with the single element directly. The analyzer respects `//nolint:stringsjoinone` directives and skips generated files, consistent with all other linters in the registry. It is registered in `All()` in `pkg/linters/registry.go`.

### Alternatives Considered

#### Alternative 1: Rely on an existing third-party linter (e.g., `staticcheck`, `golangci-lint`)

These general-purpose tools could theoretically gain this rule over time, but no widely-used tool currently flags this pattern. Waiting for upstream adoption would leave the antipattern unchecked in the meantime and would couple enforcement to an external release schedule outside the team's control. The custom `go/analysis` framework already in use makes adding a new analyzer a low-cost, high-control option.

#### Alternative 2: Enforce via code review guideline only (no automated check)

A documented coding style note would communicate the expectation, but relies entirely on reviewer attention, scales poorly as the codebase and contributor count grow, and provides no auto-fix capability. Existing custom analyzers in this repo exist precisely because automated static analysis is more reliable than code review for mechanical patterns. This option trades enforcement reliability for zero tooling overhead.

### Consequences

#### Positive
- Automatically detects and suggests fixes for a misleading `strings.Join` pattern, reducing cognitive load for code readers.
- Consistent with the established `go/analysis` pass structure — minimal boilerplate, shares infrastructure (`astutil`, `filecheck`, `nolint`) with all other linters.
- Provides an auto-fix via `SuggestedFix`, enabling `gopls` and editor tooling to apply corrections with a single action.
- Adds the analyzer to the documented registry (`doc.go`, `README.md`) so it is discoverable alongside the 55 other custom rules.

#### Negative
- One additional static analysis pass adds a small incremental build and CI time cost per invocation.
- Custom analyzers carry ongoing maintenance responsibility: the team must update the rule if the Go `ast` or `strings` package API changes in a breaking way.
- Developers unfamiliar with the analyzer may be surprised by diagnostics on code they considered idiomatic; `//nolint:stringsjoinone` is the escape hatch, but requires knowledge of the linter name.

#### Neutral
- The analyzer is added to `All()` unconditionally, meaning all builds that invoke the full linter suite will run it; there is no opt-in flag (consistent with other analyzers in this package).
- The new `pkg/linters/stringsjoinone/` subpackage follows the existing directory-per-analyzer convention; no structural change to the registry interface is needed.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
