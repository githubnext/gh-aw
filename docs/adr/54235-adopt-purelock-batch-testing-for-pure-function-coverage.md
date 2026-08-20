# ADR-54235: Adopt PureLock Batch Testing for Pure-Function Coverage Lockdown

**Date**: 2026-08-20
**Status**: Draft
**Deciders**: pelikhan, PureLock automation (app/github-actions)

---

### Context

Three Go functions in `pkg/parser` and `pkg/workflow` had 0% function coverage with no systematic mechanism to detect or address such gaps: `parseImportSpecsFromObject` (`pkg/parser/import_bfs.go:88`), `relativizeIncludedFilePath` (`pkg/parser/include_expander.go:100`), and `resolveCacheStepName` (`pkg/workflow/cache_steps.go:75`). All three are pure functions — no I/O, no global mutation, no observable side effects — making them safe targets for automated test generation. The PureLock workflow identifies pure-function coverage gaps from a ranked candidate list and generates maximal-coverage table-driven test suites in batch PRs to close those gaps.

### Decision

We will use the PureLock automated batch workflow to systematically identify pure Go functions with 0% function coverage and lock them down with table-driven unit test suites targeting 100% function coverage per function. Each batch PR targets functions confirmed pure by static analysis and manual inspection; fuzz testing is used only when marked explicitly fuzz-friendly and table-driven cases do not achieve full coverage. Tests are committed directly to the package under test (e.g., `pkg/parser/import_bfs_test.go`) using the `!integration` build tag.

### Alternatives Considered

#### Alternative 1: Rely on ad-hoc test coverage during feature work

The default model where developers add tests for new or changed code as part of feature PRs. This is well-understood and keeps tests co-located with the motivating change, but it does not systematically surface coverage gaps in pre-existing pure functions that have never been tested. The three functions in this PR had been in the codebase with 0% coverage indefinitely.

#### Alternative 2: Use Go fuzzing (`go test -fuzz`) for pure-function coverage

Some PureLock candidates are marked fuzz-friendly. Fuzzing is powerful for detecting edge cases in deterministic functions, but it requires a persistent corpus, is slower than table-driven tests in CI, and is overkill when a small set of structural branches can be exhaustively enumerated. For these three functions, table-driven cases provided 100% function coverage without fuzzing overhead.

### Consequences

#### Positive
- `parseImportSpecsFromObject`, `relativizeIncludedFilePath`, and `resolveCacheStepName` move from 0% to 100% function coverage, protecting against regressions in pure business logic.
- Establishes a repeatable, auditable batch pattern for closing coverage gaps in side-effect-free functions across the codebase.
- Test suites are fully deterministic and run under `-race`, catching data-race regressions.

#### Negative
- Automated test-only batch PRs cross volume thresholds in business logic directories, triggering ADR enforcement gates even when no production architecture is changing.
- Reviewers must validate AI-generated test cases match the intended behavior of the production function, not just the observed behavior at generation time.
- Pre-existing unrelated test failures in the sandbox environment (network-restricted and `/dev/fd`-dependent tests) must be manually distinguished from failures introduced by this change.

#### Neutral
- Package-level statement coverage changes are marginal (e.g., `pkg/parser` 72.1% → 72.4%) because function coverage targets only the specific locked-down functions, not untested statement branches within other functions.
- The `!integration` build tag keeps these tests out of integration test runs, consistent with existing patterns in `pkg/parser` and `pkg/workflow`.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
