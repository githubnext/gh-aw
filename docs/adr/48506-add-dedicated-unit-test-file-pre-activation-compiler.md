# ADR-48506: Add Dedicated Unit Test File for Pre-Activation Job Compiler

**Date**: 2026-07-28
**Status**: Draft
**Deciders**: pelikhan

---

### Context

`compiler_pre_activation_job.go` is the largest compiler module in `pkg/workflow/` with 40 functions and 956 lines of code. Despite this scale, it had no corresponding `_test.go` file, making it the only large compiler file with zero dedicated unit test coverage. This gap meant that bugs in the core pre-activation job assembly logic — including guard composition, permission merging, and job structure building — could only be caught through broader integration tests or not at all. The module's complexity (particularly `buildPreActivationJob`, `applyPreActivationIfConditionGuards`, and `buildPreActivationPermissions`) made it a high-risk area for undetected regressions.

### Decision

We will add a dedicated unit test file `pkg/workflow/compiler_pre_activation_job_test.go` targeting the highest-value helper functions in isolation, using the `!integration` build tag so tests run in the standard unit test suite without requiring integration infrastructure. Coverage focuses on `buildPreActivationJob` (guard composition, needs deduplication, output wiring), `applyPreActivationIfConditionGuards` (label guard + comment-author guard logic, expression-based bot suppression, skip-author-associations clauses), and `buildPreActivationPermissions` (release mode vs. script mode permission merging).

### Alternatives Considered

#### Alternative 1: Extend Existing Compiler Test Files

Add pre-activation coverage to `compiler_test.go` or other existing compiler test files. This was not chosen because those files are already integration-heavy; mixing isolated helper-level assertions into them would blur the unit/integration boundary, make failures harder to attribute, and increase cognitive overhead for future contributors navigating large, multi-purpose test files.

#### Alternative 2: Rely Solely on Integration Tests

Skip dedicated unit coverage entirely and depend on the existing integration test suite to catch regressions in this module. This was not chosen because integration tests are slower to run, harder to iterate on locally, and do not isolate failures at the function level — making it significantly harder to diagnose pre-activation logic bugs when they surface.

### Consequences

#### Positive
- Isolated, fast unit tests for the three most complex pre-activation helper functions, runnable with `make test-unit`
- Future contributors can verify and modify pre-activation behavior without spinning up integration infrastructure
- Follows the existing test style and `!integration` build tag convention established across `pkg/workflow/`

#### Negative
- An additional test file to maintain as the pre-activation compiler evolves; helper fixtures must stay in sync with production code structure
- Unit tests at this level cannot catch cross-module integration failures — integration test coverage still required for end-to-end validation

#### Neutral
- Uses the existing `testify` assert/require framework already standard across `pkg/workflow/`
- The `!integration` build tag is consistent with how other unit tests in this package exclude integration-only scenarios

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
