# ADR-27198: Consolidate Expression Helpers and Unify CLI Version State

**Date**: 2026-04-19
**Status**: Draft
**Deciders**: pelikhan, Copilot

---

## Part 1 — Narrative (Human-Friendly)

### Context

The `pkg/workflow` package accumulated near-duplicate expression-detection helpers across multiple files (`dispatch_repository_validation.go`, `shell.go`, `safe_outputs_validation.go`, `templatables.go`). Each file defined its own local boolean helpers for checking GitHub Actions `${{ ... }}` expressions, with subtly different semantics (permissive marker checks vs. full containment checks vs. whole-string checks). Separately, `pkg/cli` maintained its own local `version` variable and an `init()` synchronization path that had to be manually kept in sync with the canonical version state in `pkg/workflow`. Both patterns violated the single-source-of-truth principle and created silent divergence risks.

### Decision

We will centralize the three expression-detection predicates (`hasExpressionMarker`, `containsExpression`, `isExpression`) into `pkg/workflow/expression_patterns.go` and remove all per-file duplicates. We will also remove the local version variable and `init()` in `pkg/cli/version.go`, making `cli.GetVersion()` and `cli.SetVersionInfo()` thin delegators to the canonical `workflow.GetVersion()` / `workflow.SetVersion()` functions. Both changes enforce a single source of truth within their respective domains, eliminating the risk of semantic drift between helper copies.

### Alternatives Considered

#### Alternative 1: Keep per-file helpers but add linting rules

Each file could retain its own inline helper, and a custom linter or `grep`-based CI check could be added to flag divergence. This approach avoids touching call sites and keeps the helpers co-located with their consumers. It was rejected because linting checks would detect drift only after it happens rather than preventing it structurally, and the helpers are small enough that centralization adds no meaningful indirection.

#### Alternative 2: Introduce a separate `expressionutil` sub-package

A new package (e.g., `pkg/expressionutil`) could expose the helpers, providing a fully independent import path. This was considered for its explicit package boundary, but rejected because the helpers are only used within `pkg/workflow` call sites, making an external package an overengineered abstraction. Keeping the helpers unexported (`hasExpressionMarker`, `containsExpression`, `isExpression`) in `expression_patterns.go` preserves encapsulation without adding an unnecessary dependency edge.

#### Alternative 3: Retain the dual version state with better documentation

The `pkg/cli` version variable could have been kept but annotated with comments requiring manual sync with `workflow.GetVersion()`. This was rejected because documentation alone cannot enforce invariants. The actual synchronization logic (`init()` and local `version` variable) was already causing confusion; removing the indirection entirely is safer and simpler.

### Consequences

#### Positive
- Expression-check semantics are defined once; all call sites share the same behavior, eliminating silent divergence.
- Version state has a single owner (`pkg/workflow`); the CLI layer is a transparent pass-through with no independent state to maintain.
- Reduced surface area for future contributors who previously had to know which file's helper was the "correct" one.
- Test coverage can now target the helpers directly in `expression_patterns_test.go` instead of being spread across consumer tests.

#### Negative
- All call sites in `pkg/workflow` must be updated when helper semantics change; there is no local override capability per file.
- The `expression_patterns.go` file is already large (regex patterns + helper functions); additional consolidation could make it a maintenance hotspot.

#### Neutral
- `pkg/cli` becomes a thin delegation layer with no independent business logic for version management; callers of `cli.GetVersion()` and `cli.SetVersionInfo()` are unaffected at the API level.
- New tests (`expression_patterns_test.go`, `version_test.go`) were added to cover the consolidated logic and the delegation contract.

---

## Part 2 — Normative Specification (RFC 2119)

> The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this section are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### Expression Detection Helpers

1. All GitHub Actions expression-detection predicates used within `pkg/workflow` **MUST** be defined in `pkg/workflow/expression_patterns.go`.
2. Individual `pkg/workflow` source files **MUST NOT** define their own local boolean helpers that check for `${{` or `}}` patterns.
3. The three canonical helpers **MUST** provide the following distinct semantics:
   - `hasExpressionMarker(s string) bool` — returns `true` if `s` contains the substring `${{` (permissive, partial-expression check).
   - `containsExpression(s string) bool` — returns `true` if `s` contains a complete non-empty expression (a `${{` followed by at least one character before `}}`).
   - `isExpression(s string) bool` — returns `true` if the entire string `s` is a single expression (starts with `${{` and ends with `}}`).
4. Callers **MUST** select the helper whose semantics match their intent; using `hasExpressionMarker` where `containsExpression` or `isExpression` is needed (or vice versa) constitutes non-conformance.
5. New expression-detection predicates **SHOULD** be added to `expression_patterns.go` rather than introduced inline in consumer files.

### CLI Version State

1. `pkg/cli` **MUST NOT** maintain its own local version variable or an `init()` function that synchronizes version state from `pkg/workflow`.
2. `cli.GetVersion()` **MUST** delegate directly to `workflow.GetVersion()` with no local caching or transformation.
3. `cli.SetVersionInfo(v string)` **MUST** delegate directly to `workflow.SetVersion(v)` with no local storage.
4. All version state **MUST** be owned exclusively by `pkg/workflow`; `pkg/cli` **MAY** expose wrapper functions for backwards-compatible API surface but **MUST NOT** shadow or duplicate the underlying state.

### Conformance

An implementation is considered conformant with this ADR if it satisfies all **MUST** and **MUST NOT** requirements above. Failure to meet any **MUST** or **MUST NOT** requirement — in particular, defining expression-detection helpers outside `expression_patterns.go` or maintaining independent version state in `pkg/cli` — constitutes non-conformance.

---

*This is a DRAFT ADR generated by the [Design Decision Gate](https://github.com/github/gh-aw/actions/runs/24632327310) workflow. The PR author must review, complete, and finalize this document before the PR can merge.*
