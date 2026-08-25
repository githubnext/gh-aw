# ADR-55496: Split awf_config.go into Types, Schema, Build, and Policy Files

**Date**: 2026-08-24
**Status**: Draft
**Deciders**: Unknown

---

### Context

`pkg/workflow/awf_config.go` had grown to 1,090 lines, mixing three distinct concerns: Go type definitions for the AWF configuration file, JSON schema validation, AWF config JSON construction (`BuildAWFConfigJSON`), and model-policy and domain-list resolution. It was the repository's second-largest non-test Go file and was under active churn. The `pkg/workflow` package follows a "one file per functionality" convention (e.g., `awf_helpers.go`, `awf_enclaves.go`), which this monolithic file violated. Reviewers had to scan the entire file to locate the concern they cared about.

### Decision

We will split `pkg/workflow/awf_config.go` into four focused files — `awf_config.go` (type definitions), `awf_config_schema.go` (embedded JSON schema, schema compilation/validation, `buildAWFConfigSchemaURL`), `awf_config_build.go` (`BuildAWFConfigJSON` and all build/extract helpers), and `awf_config_policy.go` (`resolveModelPolicyForAWFConfig`, `intersectModelPolicyRules`, `unionModelPolicyRules`, `splitDomainList`) — matching the "one file per functionality" convention already established in `pkg/workflow`. This is a pure code move with no logic changes.

### Alternatives Considered

#### Alternative 1: Keep the monolithic file

Add package-level or function-group comments to orient readers within the single 1,090-line file. This requires no structural change and carries no migration risk, but it does not resolve the difficulty of locating and reviewing specific concerns under ongoing churn. The file would continue to grow as new AWF config sections are added.

#### Alternative 2: Extract into a separate Go package

Move AWF config logic into `pkg/workflow/awfconfig` (a new sub-package). This would provide stronger encapsulation and cleaner import boundaries. However, it would require renaming exported types, updating all call sites across the repo, and deciding which types remain in `pkg/workflow` to avoid circular imports — significant cost for what amounts to a readability improvement.

### Consequences

#### Positive
- Each file now has a single, clearly named responsibility that matches the `pkg/workflow` "one file per functionality" convention, reducing the mental surface area for reviewers.
- Future changes to schema validation, model policy, or build logic touch only the relevant file, making diffs easier to read and review.
- Each new file carries a short cross-reference header pointing to its siblings, so navigating the split is self-documenting.

#### Negative
- `BuildAWFConfigJSON` at 339 lines remains un-decomposed inside `awf_config_build.go`; linting still flags it. Decomposing it was explicitly out of scope here to keep this a mechanical, reviewable move.
- Any tooling or documentation that enumerates `awf_config.go` as the single AWF integration file (e.g., skill manifests, README appendices) must now list all four files and must be kept in sync when further files are added.

#### Neutral
- All four files remain in the same Go package (`package workflow`), so no exported symbols are renamed, no call sites change, and existing tests compile unchanged.
- The AWF release integrator skill (`SKILL.md`) and `pkg/workflow/README.md` were updated in this PR to reflect the new file layout.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
