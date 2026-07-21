# ADR-47022: Extract Named Helper Functions from Large Analyzer `run` Bodies

**Date**: 2026-07-21
**Status**: Draft
**Deciders**: pelikhan, copilot-swe-agent

---

### Context

`make golint-custom` reported 33 large-function violations (>60 body lines) across `pkg/linters` analyzer files. The violations were concentrated in `run` functions and a small number of related helpers that accumulated traversal, type-checking, and fix-generation logic inline — both as `insp.Preorder` anonymous callback bodies and as `for cur := range insp.Root().Preorder(...)` loop bodies. Because the large-function linter is a required quality gate that runs in CI, all 33 violations had to be resolved before the branch could merge. No change to analyzer semantics (diagnostics, positions, or suggested-fix `TextEdit`s) was permissible.

### Decision

We will extract the body of each oversized inline function literal or cursor loop body into a named package-level function (e.g., `analyzeWriteCall`, `checkHTTPCall`, `checkHardcodedFilePath`). Each `run` function will be reduced to setup, index construction, and a single-line delegation call. Constants that were previously scoped to `run` and needed by the extracted helper will be promoted to package-level. No behavioral changes are introduced: diagnostic messages, positions, and `SuggestedFix` values are identical before and after extraction.

### Alternatives Considered

#### Alternative 1: Suppress violations with `//nolint:largefunc` directives

Add per-function `//nolint` directives to silence the 33 findings without restructuring code. This would pass the gate but would accumulate permanent suppression debt, mask future growth of these functions, and set a precedent that `pkg/linters` functions are exempt from the quality rule.

#### Alternative 2: Raise or remove the large-function threshold for `pkg/linters`

Adjust `.golangci.yml` or the custom linter configuration to either increase the line-length limit (e.g., to 120 lines) or exclude the `pkg/linters` directory entirely. This resolves the gate failures without code changes but defeats the purpose of the linter for exactly the code that defines the linter rules — an especially bad precedent — and does not improve readability.

#### Alternative 3: Restructure analyzers as method sets on analyzer structs

Replace the current `func run(pass *analysis.Pass)` pattern with a per-analyzer struct that implements sub-passes as methods, eliminating the need to thread `pass`, `generatedFiles`, and `noLintIndex` as parameters to every extracted function. This is a deeper architectural change that would affect all 25 files uniformly and could be a worthwhile future refactor, but it is out of scope for a targeted lint-compliance fix: it introduces higher risk of behavior regression and a much larger diff.

### Consequences

#### Positive
- All 33 `largefunc` violations in `pkg/linters` are cleared; `make golint-custom` passes.
- Each extracted function has a single, named responsibility, making stack traces and profiling output more informative.
- Reviewers can read the `run` function as a high-level outline and drill into named helpers independently.
- The extraction pattern is uniform and mechanical, minimizing review surface.

#### Negative
- Extracted functions require all context to be passed explicitly (`pass`, `generatedFiles`, `noLintIndex`, and in some cases `seenImportFiles` or `inspector.Cursor`), increasing parameter counts relative to closures that captured variables implicitly.
- Two files (`execcommandwithoutcontext`, `nilctxpassed`) required a new import of `"golang.org/x/tools/go/ast/inspector"` to expose the `inspector.Cursor` type in the extracted function signatures, adding a minor import-set change.

#### Neutral
- The `formatArgOffset` constant in `errorfwrapv` was promoted from a function-local `const` to package-level to eliminate a redundant parameter; this has no behavioral effect but changes the constant's visibility scope.
- All diagnostic messages, positions, and suggested-fix `TextEdit`s are unchanged — this is a structural refactor only.
- The 25 changed files share a consistent before/after pattern, which means the diff is large in line count but low in cognitive complexity per file.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
