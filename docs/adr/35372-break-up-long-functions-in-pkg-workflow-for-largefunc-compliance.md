# ADR-35372: Break Up Long Functions in `pkg/workflow/` for `largefunc` Compliance

**Date**: 2026-05-28
**Status**: Draft
**Deciders**: pelikhan

---

## Part 1 — Narrative (Human-Friendly)

### Context

The `pkg/workflow/` package had accumulated 2,417 function-length lint violations against the `largefunc.max-lines=60` rule, concentrated in production engines (`claude_engine.go`, `codex_engine.go`, `copilot_engine_*.go`), the compiler pipeline (`compiler_main_job.go`, `compiler_jobs.go`, `compiler_orchestrator_*.go`, `compiler_validators.go`, `compiler_pre_activation_job.go`), and config/parsing helpers (`safe_outputs_config.go`, `engine.go`, `frontmatter_extraction_yaml.go`, `mcp_config_custom.go`, `tools.go`). Most violations were in production code; a long tail (~1,469 remaining) lives in ~412 test files. Beyond the lint signal, the oversized functions had grown into hard-to-review monoliths that mixed orchestration with low-level construction, making engine, compiler, and config code expensive to evolve.

### Decision

We will fix the violations by **extracting private helper functions** from each oversized function, rather than rewriting call sites, changing public APIs, or relaxing the lint rule. New helpers live either alongside their parent function in the original file or in dedicated `*_helpers.go` files when the parent file is already large, following a `{verb}_{object}` naming convention (e.g., `compiler_orchestrator_engine_helpers.go`, `tools_defaults_helpers.go`). For test files, we will additionally convert large table-driven tests into grouped `t.Run()` subtests with per-category case-builder functions, preserving coverage while reducing per-function length. The refactor is structural-only and **MUST NOT** change observable behavior, with the sole exception of a separately documented bug fix in `applySafeOutputsMessageConfig` (restoring independent handling of the `activation-comments` field that an earlier `else if` chain had incorrectly nested under an error path).

### Alternatives Considered

#### Alternative 1: Relax or remove the `-largefunc.max-lines=60` lint rule

Bump the threshold to e.g. 200 lines or disable the rule for `pkg/workflow/`. This is the cheapest path: zero diff, zero risk of regression. It was rejected because the rule is doing its job — the violating functions are objectively hard to review and modify, and the engine/compiler are the hot path for every workflow compile. Raising the cap would normalize the monolith pattern across new code and erase the signal entirely; an exemption per-file would just push the same decision down the road.

#### Alternative 2: Decompose into new sub-packages instead of private helpers

Split each oversized area (engine, compiler orchestrator, safe-outputs config) into its own sub-package with exported types and a documented internal API. This would create clearer module boundaries and make dependencies explicit. It was not chosen because the goal of this PR is to eliminate lint violations without churning import graphs, public surfaces, or call sites — a private-helper extraction is reviewable as "no behavior change" while a package split would require auditing every caller and risks rippling into downstream consumers. Sub-package decomposition remains available as a follow-up once function length is no longer a confounding factor.

#### Alternative 3: Rewrite tests as standalone functions instead of grouped `t.Run` subtests

Convert large table-driven tests into many top-level `TestX_Case_Y` functions. This trivially satisfies `largefunc` but multiplies the number of test functions and loses the cohesive grouping that table-driven tests provide. The chosen approach — `t.Run` subtests with per-category case-builder helpers — keeps the table's grouping intact (one parent test per concept, subtests per case) while moving the case data into separate helpers that fall under the line limit individually.

### Consequences

#### Positive
- Eliminates ~948 of 2,417 lint violations in production code, plus a meaningful chunk of test-file violations, restoring the `largefunc` signal across `pkg/workflow/`.
- Each extracted helper has a single, nameable responsibility, which makes future edits scoped and reviewable.
- The grouped `t.Run` subtest pattern preserves table-driven coverage while making individual cases easier to run in isolation and easier to read in test output.
- Surfaced and fixed a real bug (`applySafeOutputsMessageConfig` `activation-comments` handling) that the prior monolithic form had masked.

#### Negative
- The diff is large and structural (22+ production files, ~20 test files, plus 6 new helper files), making it expensive to review carefully — and the review value is mostly "verify nothing moved that shouldn't have."
- New helper functions add indirection: stepping through one of the refactored functions now requires hopping through 2–4 helpers to see the full flow that previously lived inline.
- ~1,469 violations remain in test files, so `largefunc` will continue to be noisy until follow-up PRs land; the lint rule cannot yet be treated as clean.
- Bundling a behavior-changing bug fix (`applySafeOutputsMessageConfig`) into a "no behavior change" refactor PR weakens the no-regression guarantee for reviewers; the fix is correct, but readers must explicitly recognize it as an exception.

#### Neutral
- New `*_helpers.go` files appear next to their parent file; this is consistent with existing conventions elsewhere in the repo but increases file count in `pkg/workflow/`.
- Public APIs of `pkg/workflow/` are unchanged; downstream packages should see no diff in usage.
- Test execution time is expected to be unchanged: subtests run in the same process and the case-builder helpers are pure construction.

---

## Part 2 — Normative Specification (RFC 2119)

> The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this section are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### Function Length Compliance

1. New or modified functions in `pkg/workflow/` **MUST** comply with the `largefunc.max-lines=60` lint rule unless a per-function lint suppression with a documented justification is added in the same change.
2. Implementations **MUST NOT** raise, relax, or disable the `largefunc.max-lines` threshold globally or for `pkg/workflow/` to silence violations.
3. When an existing function exceeds the threshold, contributors **SHOULD** decompose it by extracting private helpers rather than splitting it into new exported APIs or new sub-packages, unless a separate ADR authorizes a public reorganization.

### Helper Extraction Convention

1. Private helper functions extracted from a parent function **MUST** use a `{verb}{Object}` naming pattern (e.g., `buildEngineEnv`, `applyOrchestratorTools`) that describes what the helper does, not how it is called.
2. When a parent file is already large or organizationally dense, extracted helpers **SHOULD** live in a sibling `{parent_basename}_helpers.go` file (e.g., `compiler_orchestrator_engine_helpers.go` for helpers extracted from `compiler_orchestrator_engine.go`).
3. Helpers extracted purely to satisfy the line limit **MUST NOT** change the observable behavior of their parent function; any behavioral change accompanying such an extraction **MUST** be called out explicitly in the PR body and **SHOULD** be split into a separate commit.
4. Helpers **SHOULD** be unexported (lowercase first letter) unless they are needed by another file in the same package.

### Test Refactoring Pattern

1. Oversized table-driven tests in `pkg/workflow/` **SHOULD** be converted to a parent `func TestX` that groups cases via `t.Run(name, func(t *testing.T) { ... })` rather than expanded into many top-level test functions.
2. Per-category case data **SHOULD** be returned by a dedicated case-builder helper function (e.g., `buildSafeOutputsValidationCases`) so that each helper is independently below the line threshold.
3. Refactored tests **MUST** preserve the set of test cases and assertions of the pre-refactor form; case additions or removals **MUST NOT** be bundled into a refactor commit.

### Conformance

An implementation is considered conformant with this ADR if it satisfies all **MUST** and **MUST NOT** requirements above. Failure to meet any **MUST** or **MUST NOT** requirement constitutes non-conformance.

---

*This is a DRAFT ADR generated by the [Design Decision Gate](https://github.com/github/gh-aw/actions/runs/26555448431) workflow. The PR author must review, complete, and finalize this document before the PR can merge.*
