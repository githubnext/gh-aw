# ADR-42257: Auto-Derive and Warn on Missing `path:` for Cross-Repo Checkout Entries

**Date**: 2026-06-29
**Status**: Draft
**Deciders**: Unknown

---

### Context

The gh-aw compiler emits a workflow manifest containing `GH_AW_CHECKOUT_PATH_N` environment variables for each checkout entry. When a `checkout:` block specifies a cross-repository entry (i.e., a non-empty `repository:` field) without an explicit `path:` field, the compiler was emitting `GH_AW_CHECKOUT_PATH_N: ""`. This empty value caused safe-outputs handlers such as `push_to_pull_request_branch` to fail at runtime with "Repository not found in workspace." The `actions/checkout` action itself recommends supplying a `path:` when checking out multiple repositories simultaneously, making explicit paths the expected convention. Dynamic repository expressions (e.g., `${{ github.event.inputs.repo }}`) cannot be resolved at compile time and must be excluded from this check.

### Decision

We will add a compiler validation pass (`validateCrossRepoCheckoutPaths`) that runs before checkout step generation. For each cross-repo checkout entry missing an explicit `path:`, the compiler will auto-derive a path from the repository-name segment of the `owner/repo` slug (e.g., `githubnext/gh-aw-side-repo` → `gh-aw-side-repo`), mutate the `CheckoutConfig` in place so that both the emitted `actions/checkout` step and the `GH_AW_CHECKOUT_PATH_N` manifest variable are non-empty, and emit a deprecation warning directing workflow authors to add an explicit `path:`. Dynamic expressions (those containing `${{`) are skipped because their target repository cannot be determined at compile time.

### Alternatives Considered

#### Alternative 1: Hard compilation error

The compiler could reject workflow definitions where a cross-repo checkout entry lacks an explicit `path:`, treating it as an error rather than a warning. This would eliminate all ambiguity and force authors to be explicit upfront. It was rejected because it would break existing workflows that omit `path:` and currently expect the compiler to handle the case implicitly, creating a disruptive migration burden without a safe upgrade path.

#### Alternative 2: Silent auto-derivation (no warning)

The compiler could silently auto-derive the path without emitting any deprecation warning. This would be fully backward-compatible and produce no noise for existing users. It was rejected because it would perpetuate implicit behavior indefinitely — authors would never be incentivized to move to explicit `path:` declarations, and the implicit derivation rule would become a permanent, undocumented contract in the compiler.

### Consequences

#### Positive
- Eliminates the `GH_AW_CHECKOUT_PATH_N: ""` root cause, preventing runtime "Repository not found in workspace" failures in safe-outputs handlers.
- Provides actionable, inline deprecation warnings that guide authors toward explicit `path:` configuration, improving long-term workflow hygiene.

#### Negative
- Dynamic repository expressions (e.g., `${{ github.event.inputs.repo }}`) are silently skipped: if such an expression resolves to a cross-repo target at runtime, the manifest may still contain an empty path, leaving the runtime failure mode in place for that class of dynamic checkouts.
- Introduces implicit compiler mutation of `CheckoutConfig` objects during the validation phase, which may surprise contributors who expect validation to be purely read-only and side-effect-free.

#### Neutral
- The auto-derivation logic (`strings.LastIndex(repo, "/") + 1`) mirrors the convention used by `actions/checkout` itself for multi-repo scenarios, so the derived paths are consistent with what authors would write manually.
- The deprecation warning is emitted to stderr as a compiler warning (not an error), keeping the existing warning-count mechanism and not changing exit codes for currently-valid workflow definitions.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
