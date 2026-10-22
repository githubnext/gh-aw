# ADR-70131: Share Checkout Trust Predicates and Expression Parsing Between Compiler and AST Threat Linters

**Date**: 2026-10-19
**Status**: Draft
**Deciders**: pelikhan, Unknown (review required)

---

### Context

`gh aw lint` runs read-only AST rules mapped to compiler threat IDs (see ADR-70026). Before this PR, those rules did not check two parts of the Compiler Threat Detection Specification. The first is authored checkout trust on `pull_request_target` triggers (CTR-008). The second is forbidden prompt-expression forms such as multiline interpolations, malformed expressions, and blocked JavaScript property traversal (CTR-010). The compiler already defined which `pull_request_target` checkouts it trusts: the base repository with `base.sha`/`base.ref`, or exact literal repository/ref pairs from `allowed-checkouts`. Those regexes and predicates were private to `pkg/workflow/pull_request_target_validation.go`. The prompt checks also need to parse `${{ }}` interpolations correctly. That parsing was a private Actions-lexer visitor in `pkg/workflow/taint`. If the linter reimplemented either piece, it could drift from the compiler. Lint would then disagree with what compilation accepts or rejects.

### Decision

We will add a small shared package, `pkg/workflow/checkoutpolicy`, that holds the `pull_request_target` checkout trust predicates: `MatchesExpression`, `Base`, `LiteralPair`, `LiteralRepository`, and `LiteralRef`. Both compiler validation and the new `pull-request-target-checkout` lint rule (CTR-008) will call it. We will also export the taint package's Actions-lexer interpolation visitor as `taint.VisitInterpolations` and reuse it in the new `prompt-expression` lint rule (CTR-010). Both rules are registered by default. They check only authored AW evidence, defer missing checkout or allowlist policy when unresolved imports could supply it, and claim only a subset of the threat rule, not full runtime conformance. The main reason is to keep a single source of truth, so lint and compiler apply identical trust predicates and expression parsing. This follows the shared-detector pattern ADR-70026 set with `pkg/workflow/shellpattern`.

### Alternatives Considered

#### Alternative 1: Reimplement the predicates and expression parsing inside `pkg/workflow/lint`

The lint rules could carry their own copies of the checkout regexes and their own `${{ }}` scanner. That keeps the lint package self-contained and leaves compiler code untouched. It was rejected because these are security predicates. A copy that drifts, for example accepting a ref form the compiler rejects, would produce false clean lint results. Naive delimiter matching also mishandles quoted `}}` and doubled single quotes, which the existing Actions-lexer visitor already handles.

#### Alternative 2: Have the linter invoke compiler validation directly

The lint rules could call `validatePullRequestTargetTrigger` or other compiler validation entry points on a partially built `WorkflowData`. That would guarantee identical behaviour. It was rejected because lint rules are read-only AST checks that must not compile workflows or depend on compiler state. Compiler validation also needs resolved imports and generated configuration. Lint has neither, and it must preserve import uncertainty by deferring rather than failing.

#### Alternative 3: Leave CTR-008 checkout and CTR-010 prompt forms to the compiler only

Authors would still see these failures at compile time. This was rejected because the AST linter exists to surface threat-mapped problems early and in a standalone way, with traceable `CTR-*` IDs. The specification also lists coverage gaps for these rules.

### Consequences

#### Positive
- Compiler and linter use one implementation of the checkout trust predicates, so their trusted base and literal-pair decisions cannot drift.
- Prompt expression checks reuse the Actions lexer, so quoted closing delimiters are treated correctly and malformed interpolations are reported without exposing expression contents.
- `pull_request_target_validation.go` shrinks, with its duplicate regexes and helpers removed, and the predicates now have direct unit tests in `checkoutpolicy/target_test.go`.
- New threat coverage is traceable to CTR-008 and CTR-010 in diagnostics and in specification version 1.0.62.

#### Negative
- Adds another small shared package under `pkg/workflow/`. Changes to checkout trust semantics now affect both compiler and lint, so they need tests on both sides.
- `taint.VisitInterpolations` becomes exported API, which widens the taint package's public surface and constrains future refactors.
- Both rules cover only a subset. They do not resolve imports, reconstruct generated checkout provenance, or implement the compiler's full expression-context allowlist, so a clean lint result is not proof of conformance.
- Deferring findings when imports are pending can hide real issues until compile time.

#### Neutral
- `pull-request-target-checkout` follows authored strict/non-strict severity, and trigger-risk acknowledgment or role-warning suppression does not bypass it. `prompt-expression` findings are always errors.
- Compiler checkout behaviour is intended to stay unchanged. The refactor replaces inline regexes with calls to the shared package.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
