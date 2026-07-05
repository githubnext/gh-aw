# ADR-43376: Switch Linter Tests to RunWithSuggestedFixes with Golden Files

**Date**: 2026-07-04
**Status**: Draft
**Deciders**: Unknown (bot-authored PR by copilot-swe-agent)

---

### Context

Several linters in this repository (`stringsindexcontains`, `sprintfint`, `stringreplaceminusone`) emit `SuggestedFix` text edits so that automated tools can apply fixes to user code. Their test entrypoints used `analysistest.Run`, which validates only diagnostic positions and messages — it never exercises the fix builders that produce the actual replacement text. As a result, the fix builders for these three linters were completely unexercised: broken autofix output could pass the entire test suite silently. Other linters in the same repository (`fprintlnsprintf`, `lenstringzero`, `tolowerequalfold`) already use the correct harness, creating an inconsistency in test coverage quality.

### Decision

We will switch the test entrypoints for `stringsindexcontains`, `sprintfint`, and `stringreplaceminusone` from `analysistest.Run` to `analysistest.RunWithSuggestedFixes`, and add a corresponding `.go.golden` file for each linter that captures the expected post-fix source. This matches the test pattern already established by other fix-emitting linters in this repository, closes the coverage gap, and requires no changes to production analyzer code.

### Alternatives Considered

#### Alternative 1: Keep `analysistest.Run` and Add Separate Manual Fix Tests

Rather than switching the entrypoint, custom test cases could apply each linter's suggested fix by hand and assert on the resulting source text using string or diff comparisons.

This would exercise the fix builders but requires writing and maintaining bespoke test scaffolding for each linter, duplicating logic that the standard library already provides through `RunWithSuggestedFixes`. The added maintenance cost is higher than the benefit when a first-class mechanism exists.

#### Alternative 2: Write a Shared Custom Test Helper

A shared helper function could be introduced that applies suggested fixes using the `go/analysis` rewrite machinery and compares output against an in-memory expected string, avoiding the golden-file convention entirely.

This provides flexibility in how expectations are expressed but introduces a non-standard testing pattern that future contributors would need to learn. The `RunWithSuggestedFixes` + golden-file convention is already established in the codebase; a custom helper would diverge from it without a clear benefit.

### Consequences

#### Positive
- Fix builders for `stringsindexcontains`, `sprintfint`, and `stringreplaceminusone` are now exercised in CI; any regression in fix output will fail the test suite immediately.
- Test coverage for autofix linters is consistent across the entire repository — all fix-emitting linters now use the same harness and file convention.

#### Negative
- Golden files must be updated whenever fix logic changes: a change to how a linter rewrites code now requires a corresponding update to the `.go.golden` file, adding a small but recurring maintenance step.
- The golden-file approach makes expected output less visible inline; reviewers must open a separate file to see what the fixed source looks like.

#### Neutral
- No production analyzer code changed; linting behavior at runtime is identical.
- The three golden files are added under each linter's existing `testdata/` directory, following the structure already used by other linters.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
