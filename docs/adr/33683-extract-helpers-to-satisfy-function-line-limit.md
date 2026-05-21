# ADR-33683: Extract Same-Package Helpers to Satisfy Function-Line Limit

**Date**: 2026-05-21
**Status**: Draft
**Deciders**: Unknown

---

## Part 1 — Narrative (Human-Friendly)

### Context

The repository's custom linter suite (`make golint-custom`) enforces a maximum function body length of 60 lines via the `largefunc` analyzer. Several functions in `pkg/parser` were the worst offenders in the repo — `processImportsFromFrontmatterWithManifestAndSource` at 492 lines, `ScatterSchedule` at 441 lines, `ParseMCPConfig` at 253 lines, and others between 100 and 250 lines. These functions were difficult to read, hard to test in isolation, and blocked the linter from being promoted to a blocking CI gate. The 60-line policy itself is established elsewhere; this PR is the first systematic effort to bring `pkg/parser` into compliance and therefore sets the precedent other packages will follow.

### Decision

We will refactor each oversized function by extracting focused, descriptively named helper functions into the **same file** in the **same package**, while preserving the original function's **exported signature** and external behavior. The original function becomes an orchestrator that delegates to the extracted helpers; no public API changes, no new packages, no interface redesign, and no lint suppressions. This minimizes the blast radius of mechanical cleanup work and lets reviewers verify "no logic changes" via signature parity.

### Alternatives Considered

#### Alternative 1: Suppress the linter on oversized functions

Add `//nolint:largefunc` or per-file ignores to the worst offenders and defer real cleanup. Rejected because it permanently exempts the most complex code in the repository from the rule that exists precisely to keep that code readable, and it prevents the linter from being promoted to a blocking gate.

#### Alternative 2: Split functions into new sub-packages or interfaces

Decompose each oversized function by introducing new internal packages (e.g., `pkg/parser/mcp/extractor`) or by promoting helpers behind interfaces to enable independent testing. Rejected for this pass because it conflates two distinct decisions — "shrink the function" and "redesign the module" — and would make the diff impossible to review as a pure refactor. A future ADR may revisit module boundaries once compliance is mechanical.

#### Alternative 3: Raise the 60-line threshold

Argue that 60 lines is too aggressive for a parser package and raise the limit (or scope it per-package). Rejected because the limit is already established repo-wide and several other packages comply; raising it would weaken a working convention rather than fix the offending code.

### Consequences

#### Positive

- Each refactored function fits in one screen and reads as a sequence of named steps (`buildInitialImportQueue`, `processNextImport`, etc.), improving review and onboarding.
- Same-signature refactor lets the reviewer verify behavioral parity by checking call sites are untouched; risk of regressing exported behavior is minimal.
- Establishes the canonical pattern for the remaining 25 `pkg/parser` and 277 `pkg/workflow` violations called out in the PR body, so subsequent cleanup PRs can follow this template without further design discussion.

#### Negative

- The same-file constraint means files grow in helper count even as average function size shrinks; some files now contain a dozen unexported helpers that are only meaningful in the context of one orchestrator.
- No new tests are added — the refactor is behavior-preserving by inspection, which depends on the existing test coverage of the orchestrator catching any extraction mistake.
- Cosmetic helpers may obscure the original flow for readers who preferred the linear procedural form; readability of the orchestrator improves at the cost of jumping through helper definitions.

#### Neutral

- Net line count decreased (1,361 added vs. 1,898 removed) because helper boundaries eliminate repeated local variable scaffolding and `if/else` ladders.
- Helpers are unexported, so future refactors that wish to relocate them into a new package or expose them behind interfaces remain unconstrained by this ADR.
- The same approach can be applied to `pkg/workflow` violations without further design review.

---

## Part 2 — Normative Specification (RFC 2119)

> The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this section are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### Refactor Scope

1. A function refactored to satisfy the `largefunc` linter **MUST** preserve its original name, parameter list, return types, and exported/unexported visibility.
2. Extracted helpers **MUST** be defined in the same Go file as the orchestrator function they support.
3. Extracted helpers **MUST** be unexported (lower-case initial letter) unless the orchestrator's existing siblings already export equivalent functionality.
4. The refactor **MUST NOT** introduce new packages, new interfaces, or new exported types solely for the purpose of shrinking a function.
5. The refactor **MUST NOT** add `//nolint:largefunc` or any equivalent suppression directive to the orchestrator or its helpers.

### Behavioral Preservation

1. The refactor **MUST NOT** change observable behavior of the orchestrator function as exercised by its existing call sites or tests.
2. Call sites outside the refactored file **MUST NOT** be modified by the refactor commit, except for purely mechanical changes such as gofmt or import reordering.
3. The refactor **SHOULD** keep the orchestrator function body under 60 lines so that the `largefunc` linter passes; helper functions **SHOULD** also satisfy the 60-line limit, decomposing further if needed.

### Naming and Organization

1. Extracted helper names **SHOULD** describe the step they perform (e.g., `buildInitialImportQueue`, `processNextImport`) rather than the position in the original function (e.g., `step1`, `helperA`).
2. Helpers **MAY** be placed adjacent to their orchestrator or grouped at the bottom of the file, but the file **SHOULD** maintain a consistent ordering convention throughout.

### Conformance

An implementation is considered conformant with this ADR if it satisfies all **MUST** and **MUST NOT** requirements above. Failure to meet any **MUST** or **MUST NOT** requirement constitutes non-conformance.

---

*This is a DRAFT ADR generated by the [Design Decision Gate](https://github.com/github/gh-aw/actions/runs/26206535616) workflow. The PR author must review, complete, and finalize this document before the PR can merge.*
