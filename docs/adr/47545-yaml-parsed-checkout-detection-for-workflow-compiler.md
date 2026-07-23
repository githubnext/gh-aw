# ADR-47545: YAML-Parsed Checkout Detection for Workflow Compiler

**Date**: 2026-07-23
**Status**: Draft
**Deciders**: Unknown

---

### Context

The workflow compiler inserts runtime setup steps (e.g., Node.js, ARC/DinD) immediately after the first `actions/checkout` step in user-defined custom steps. This deferral is a two-phase operation: a detection phase determines whether custom steps contain a checkout (triggering deferral), and an insertion phase locates the checkout step to insert runtime setup after it. The original implementation used substring string matching in both phases. However, the insertion phase looked ahead from `i + 1` lines, so when a checkout appeared as `- uses: actions/checkout@v6` (the unnamed shorthand), the action reference was on the current line — never examined by the look-ahead — causing the deferred runtime setup to be silently dropped from the generated workflow. Named checkout forms (`- name: Checkout\n  uses: actions/checkout@v6`) worked because the `uses:` key appeared on a subsequent line. Any workow using the unnamed shorthand was silently broken.

### Decision

We will replace the dual substring-based checkout matching with a single YAML-parsed helper (`findFirstCheckoutStepIndex`) that unmarshals the custom-steps YAML, iterates the parsed step list, and returns the zero-based index of the first step whose `uses` field exactly matches `actions/checkout` or starts with `actions/checkout@`. Both the `ContainsCheckout` detection function and the `addCustomStepsWithRuntimeInsertion` insertion function will share this helper, eliminating the detection/insertion mismatch. The insertion phase uses the pre-computed step index rather than per-line look-ahead to identify the target step during line-by-line YAML rendering.

### Alternatives Considered

#### Alternative 1: Fix the Look-Ahead in the Insertion Phase Only

The minimal fix would be to also examine the current `- uses:` line (not just `i + 1` onwards) in the existing look-ahead loop inside `addCustomStepsWithRuntimeInsertion`. This would resolve the immediate bug without changing the detection architecture.

Why not chosen: This fix would leave two independent and semantically inconsistent checkout detectors in the codebase — one using substring matching (detection phase) and one using look-ahead string scanning (insertion phase). Future checkout-form edge cases (e.g., quoted action references, whitespace variants) could reintroduce a mismatch. The substring detector also produced false positives on comments, `run:` field text, and action names containing "checkout" (as documented by the test cases changed in `permissions_parser_test.go`).

#### Alternative 2: Parse YAML Once at a Higher Level and Pass a Shared Representation

A more architecturally ambitious approach would parse the full custom-steps YAML once at the call site of `emitCustomSteps` and pass structured step data into both detection and insertion functions, eliminating re-parsing.

Why not chosen: This approach requires refactoring multiple function signatures and callers that currently pass `customSteps` as a raw string throughout the compiler pipeline. The single shared `findFirstCheckoutStepIndex` helper achieves consistent detection at lower refactoring cost, since it is called once per compilation path and its result is threaded into both phases within the same call chain.

### Consequences

#### Positive
- Checkout detection is now exact: YAML parsing rejects false positives from comments, `run:` field text (`echo "actions/checkout@..."`), and action names that merely contain "checkout" (e.g., `my-actions/checkout@...`).
- Named and unnamed checkout forms (`- name: Checkout\n  uses:` vs `- uses:`) are handled equivalently, eliminating the behavioral asymmetry that caused the bug.
- The first checkout step index is computed once and shared between detection and insertion, removing the structural possibility of the two phases diverging.

#### Negative
- The `go-yaml` package is now imported directly in `compiler_workflow_helpers.go`; previously this helper file had no YAML dependency. If the YAML library ever changes parse behavior, checkout detection will change with it.
- Malformed custom-steps YAML (e.g., missing `steps:` key or a top-level syntax error) now causes `findFirstCheckoutStepIndex` to return `(0, false)`, meaning checkout is not detected and no deferral occurs. Previously the substring matcher would still detect checkout in malformed YAML. The practical impact is low: malformed YAML also fails later compilation stages.

#### Neutral
- The `isCheckoutActionReference` helper normalizes the `uses` value (trims whitespace, strips surrounding quotes, lowercases) before comparison, making the check robust to minor formatting variation without adding regex complexity.
- Step boundary detection in the insertion loop is generalized from `- name:` or `- uses:` prefix checks to any `- ` prefix at the same indentation level, matching actual YAML list semantics more closely.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
