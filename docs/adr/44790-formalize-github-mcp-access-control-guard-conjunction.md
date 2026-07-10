# ADR-44790: Formalize GitHub MCP Access-Control Decision as a Guard Conjunction

**Date**: 2026-07-10
**Status**: Draft
**Deciders**: pelikhan, copilot-swe-agent

---

### Context

The GitHub MCP access-control compliance suite previously used a fixture-driven approach: each test fixture described an input `ToolConfig` + `AccessRequest` pair and an expected outcome (allow/deny). This left the decision function implicit — there was no single authoritative statement of which predicates must hold for access to be granted, in what order they are evaluated, or which error code is returned when multiple guards fail simultaneously.

As the number of normative requirements grew (repo scope, role filtering, private-repo gating, tool allowlists, blocked-user safety, and integrity ordering — §§4–10 of the spec), the absence of a formal model made it difficult to reason about completeness of test coverage or to verify that implementations honour the first-failing-guard semantics required by the spec. This PR was the direct consequence of that gap: a formal model was needed to tie predicates to tests in an auditable way.

### Decision

We will express the GitHub MCP access-control decision function as a deterministic conjunction of six ordered guard predicates (`P1_RepoMatch ∧ P2_RoleAllow ∧ P3_PrivateRepoAllow ∧ P4_ToolAllowed ∧ P5_NotBlocked ∧ P6_IntegrityMet`), document the conjunction in the compliance spec (`specs/github-mcp-access-control-compliance/README.md`), and bind each predicate to an executable conformance test in a self-contained in-test mini-evaluator (`pkg/workflow/github_mcp_access_control_formal_test.go`). The denial error code is always the code of the first failing guard in evaluation order.

### Alternatives Considered

#### Alternative 1: Maintain fixture-only compliance coverage

Keep the existing fixture-based approach and add more fixtures to cover the new predicates, without introducing a formal model or a standalone evaluator.

This was rejected because it keeps the decision function implicit. There is no single place where a reader can see all six guards, their evaluation order, and the first-failing-guard rule. New contributors and compliance auditors must reconstruct the model by reading all fixtures. It also makes it impossible to write invariant-style safety properties (e.g., "blocked users are _always_ denied regardless of all other conditions") without either duplication or framework support that the fixture runner does not provide.

#### Alternative 2: Use a dedicated formal-specification language (TLA+, Alloy, or Coq)

Express the access-control model in a formal specification language outside the Go test suite, and generate or manually write Go conformance tests that mirror the spec.

This was rejected because it introduces a toolchain and expertise requirement separate from the existing Go environment. It also creates a two-language synchronisation burden: every change to the spec must be reflected in the Go tests independently. The in-test mini-evaluator approach keeps specification and conformance tests in the same file, in the same language, reviewed and maintained together.

#### Alternative 3: Extract the evaluator into a shared production package and test against it directly

Instead of a self-contained in-test evaluator, promote the formal model to a production `pkg/mcp/accesscontrol` package and write the predicate-mapped tests against the real implementation.

This was considered and deferred rather than rejected. It would eliminate the risk of the test model drifting from the real implementation (see Negative consequences). However, the real access-control logic is currently distributed across multiple locations without a single canonical evaluator, so extracting it first would require a larger refactor outside the scope of this PR. The in-test evaluator was chosen as a first step: it documents the intended semantics precisely enough that the canonical implementation can be validated against it later.

### Consequences

#### Positive
- The guard conjunction, evaluation order, and first-failing-guard denial-code rule are now documented in the compliance spec (`README.md`) and enforced by executable tests — a single source of truth for the formal model.
- Each normative predicate (P1–P6) and each safety invariant (INV1, INV2, SAFETY_BlockedUser, SAFETY_NoSpuriousAllow) is mapped to a named `TestFormal_*` function in the behavioral coverage map, making coverage audits straightforward.
- Adding a new access-control dimension requires only: (a) adding a guard predicate to the formal model in README.md, (b) adding a branch to `formalEvaluateAccess`, and (c) writing a corresponding `TestFormal_` function — a clear, low-friction extension path.
- Safety properties that would require many fixtures to express (e.g., "blocked user is always denied regardless of other conditions") are now expressible as single invariant tests.

#### Negative
- The in-test mini-evaluator (`formalEvaluateAccess`) is a parallel implementation of the production access-control logic. If the real production logic changes without a corresponding update to the formal test model, the tests will still pass while the production behaviour diverges from the spec.
- The formal model uses a fixed evaluation order (BlockedUser checked first, IntegrityMet checked last), which the real implementation must also honour. This is not enforced by a compiler or type system — only by convention and code review.

#### Neutral
- The `formalEvaluateAccess` function and its supporting types (`formalToolConfig`, `formalAccessRequest`, `formalDecision`) are package-internal to `workflow` tests (build tag `!integration`). They are not exported and do not affect the public API.
- The predicate numbering in the spec (P1–P6) does not match the guard evaluation order in the code (blocked-user check runs before repo-match check). This is by design — the spec numbers reflect conceptual grouping while the code reflects safety-first ordering — but reviewers unfamiliar with the design may find this surprising.
- The `containsExact` helper added in this file shadows or duplicates any similar helper in the `workflow` package. [TODO: verify whether a `containsExact` already exists elsewhere in the package and consolidate if so.]

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
