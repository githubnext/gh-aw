# ADR-26292: `checkout` Field Support in Importable Shared Workflows with Append-After-Main Merge Semantics

**Date**: 2026-04-14
**Status**: Draft
**Deciders**: pelikhan, Copilot

---

## Part 1 — Narrative (Human-Friendly)

### Context

The GitHub Agentic Workflows (gh-aw) compiler allows shared workflow files to be imported by main workflows, enabling reusable configuration for steps, tools, permissions, and similar fields. However, the `checkout` field — used to configure additional repository checkouts for SideRepoOps workflows — could only be declared in the main workflow file. This forced every workflow that needed to check out a shared target repository to duplicate an identical `checkout:` block, making shared workflows less self-contained and violating the DRY principle across the many SideRepoOps patterns in the codebase.

### Decision

We will allow the `checkout` field to be declared in importable shared workflow files. Imported checkout entries are appended *after* the main workflow's checkout entries so that the existing `CheckoutManager` deduplication logic — which uses the `(repository, path)` key pair and a first-seen-wins strategy — naturally gives the main workflow's entries unconditional precedence over any imported value. If the main workflow sets `checkout: false`, all checkout configuration, including any entries sourced from imported files, is suppressed entirely. Internally, imported checkout configs are accumulated as newline-separated JSON values (one per imported file) in a new `MergedCheckout` field on `ImportsResult`, then parsed and appended in the compiler orchestrator.

### Alternatives Considered

#### Alternative 1: Continue Requiring Main Workflow to Declare All Checkout Config (Status Quo)

Each main workflow consuming a shared SideRepoOps pattern must repeat the same `checkout:` block. This is the simplest implementation but contradicts the goal of making shared workflows fully self-contained and creates drift risk when the target repo or branch changes across multiple consumer workflows.

#### Alternative 2: First-Import-Wins Strategy (Like `github-app`)

Accept only the first `checkout:` found across all imported files and discard any subsequent ones. This mirrors the strategy used for `github-app`. It was rejected because `checkout` is a list field that may legitimately aggregate distinct repository entries from multiple independent imports (e.g., one shared workflow contributes repo-a, another contributes repo-b). Discarding all but the first import would silently drop valid configurations.

#### Alternative 3: Error on Duplicate `(repository, path)` Pairs Across Imports (Like `env`)

Surface a hard compilation error when two imported files both define a checkout for the same `(repository, path)` key. This was considered for consistency with the `env` merge semantics, but rejected because the `CheckoutManager`'s existing first-seen-wins deduplication is already the documented and tested contract for checkout merging. Adding an error here would constrain valid use cases (e.g., an import that happens to duplicate a checkout already present in the main workflow) and is unnecessary given that the main workflow already has clear override authority.

#### Alternative 4: Introduce a Dedicated `shared-checkout:` Field

Add a separate frontmatter field (e.g., `shared-checkout:` or `imported-checkout:`) to avoid conflating the local checkout intent with the inherited one. This was rejected because it introduces unnecessary naming complexity, would require documentation and parser changes for a new field, and the `checkout:` field name already carries the right semantic meaning regardless of origin.

### Consequences

#### Positive
- Shared workflow files for SideRepoOps patterns can now centralize the `checkout:` block, eliminating repetition across every consumer workflow.
- The main workflow retains full override authority: its entries always take precedence via `CheckoutManager`'s `(repository, path)` deduplication (first-seen-wins), consistent with the "main workflow is the source of truth" invariant established for other merged fields.
- `checkout: false` in the main workflow continues to act as a hard suppress, disabling all checkout regardless of what imports define.
- The implementation reuses the existing newline-separated JSON serialization convention already used for other multi-import fields (`MergedJobs`, `MergedEnv`, etc.).

#### Negative
- The merge semantics (append-after-main, silent deduplication) are subtler than a simple override or an explicit error — workflow authors must understand that duplicate `(repository, path)` pairs from imports are silently dropped, not flagged.
- `checkout: false` now suppresses imported checkout entries, which may be surprising to authors who expect the main workflow's `checkout: false` to only affect its own locally declared config.
- `ImportsResult`, `importAccumulator`, and the compiler orchestrator each gain new fields and logic, increasing the structural surface area of the compiler pipeline.

#### Neutral
- The new behavior is additive: existing workflows without `checkout:` in their shared imports are entirely unaffected; no migration is needed.
- The JSON-per-line accumulation pattern in `MergedCheckout` is consistent with `MergedJobs` and `MergedCaches`, keeping the internal serialization approach uniform.

---

## Part 2 — Normative Specification (RFC 2119)

> The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this section are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### Checkout Field Allowance in Shared Imports

1. Shared workflow imports **MUST** be permitted to declare a `checkout:` field; the compiler **MUST NOT** treat `checkout` as a forbidden field in shared workflow files.
2. The `checkout` key **MUST NOT** appear in `SharedWorkflowForbiddenFields`.
3. A shared workflow's `checkout:` field **MAY** be a single object or an array of objects; the extractor **MUST** handle both forms.
4. A shared workflow's `checkout: false` value **MUST** be silently ignored by the import extractor (the `false` suppression semantics apply only to the main workflow's declaration).

### Checkout Merge Semantics

1. Imported checkout entries **MUST** be appended after the main workflow's checkout entries in `workflowData.CheckoutConfigs` so that the `CheckoutManager`'s first-seen-wins deduplication on `(repository, path)` pairs gives the main workflow's entries unconditional precedence.
2. When the main workflow declares `checkout: false`, the compiler **MUST NOT** append any imported checkout entries; `workflowData.CheckoutDisabled` **MUST** remain `true` regardless of what imported files define.
3. When the main workflow does not declare `checkout: false`, imported checkout entries **MUST** be parsed and appended to `workflowData.CheckoutConfigs` after the main workflow's entries, in the order they appear across imports.
4. Duplicate `(repository, path)` pairs across imports **MUST** be resolved by the existing `CheckoutManager` deduplication logic (first-seen-wins); the compiler **MUST NOT** return an error for such duplicates.

### Internal Data Model

1. `ImportsResult` **MUST** expose a `MergedCheckout string` field containing newline-separated JSON-encoded checkout values accumulated from all imported files.
2. The `importAccumulator` struct **MUST** maintain a `checkouts []string` slice, where each element is the raw JSON of a single imported `checkout:` value (object or array).
3. Implementations **MUST** serialize `MergedCheckout` as `strings.Join(acc.checkouts, "\n")`, consistent with the newline-separated JSON convention used for other multi-import accumulated fields.
4. Implementations **MUST NOT** include `"null"` or `"false"` JSON values in the `checkouts` slice; such values from imported files **MUST** be silently skipped.

### Conformance

An implementation is considered conformant with this ADR if it satisfies all **MUST** and **MUST NOT** requirements above. Specifically: the `checkout` field is accepted in shared imports without warning; `checkout: false` in a shared import is silently ignored; imported checkout entries are appended after the main workflow's entries so the main workflow takes precedence; `checkout: false` in the main workflow suppresses all imported checkout entries; and the internal representation uses newline-separated JSON in `ImportsResult.MergedCheckout`. Failure to meet any **MUST** or **MUST NOT** requirement constitutes non-conformance.

---

*This is a DRAFT ADR generated by the [Design Decision Gate](https://github.com/github/gh-aw/actions/runs/24424945242) workflow. The PR author must review, complete, and finalize this document before the PR can merge.*
