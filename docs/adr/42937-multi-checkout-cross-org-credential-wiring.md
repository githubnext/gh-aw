# ADR-42937: Multi-Checkout Cross-Org Credential Wiring Strategy

**Date**: 2026-07-02
**Status**: Draft
**Deciders**: Unknown

---

### Context

The workflow compiler (`pkg/workflow`) must support workflows that check out repositories from multiple organizations in a single GitHub Actions job. Each organization may require distinct authentication credentials — either a pre-existing Personal Access Token (PAT, via `github-token`) or a GitHub App installation token (via `github-app`). Agentic workflows that span org boundaries are a core use case. The critical constraint is that GitHub Actions step output references (`${{ steps.<id>.outputs.token }}`) are forward-only: a step may only reference outputs from steps that have already executed, so any token-minting step must be emitted before the checkout step that consumes it.

### Decision

We will use an **index-based, heterogeneous credential wiring strategy** in the workflow compiler. For each entry in the `checkout:` frontmatter list, the compiler will:

1. If the entry uses `github-token`, inject the PAT directly into the checkout step's `token:` field — no intermediate step is emitted.
2. If the entry uses `github-app`, emit an app-token minting step with ID `checkout-app-token-N` (where `N` is the zero-based list index) *before* the corresponding `actions/checkout` step, scoped to the repository's org component as `owner:`.
3. Emit a checkout manifest as env-vars (`GH_AW_CHECKOUT_MANIFEST_COUNT`, `GH_AW_CHECKOUT_REPO_N`, `GH_AW_CHECKOUT_TOKEN_N`) so that safe-outputs handlers can resolve the correct on-disk path and credential for each checked-out repo without additional network calls.
4. Disable `persist-credentials` on all cross-org checkout steps to prevent credential leakage to subsequent steps.

### Alternatives Considered

#### Alternative 1: Single shared cross-org token

Use one organization-level token (or a GitHub App with broad installation) for all checkouts in the same workflow run. Rejected because it requires over-broad permissions that violate the principle of least privilege, and it cannot support heterogeneous auth types (PATs and GitHub Apps coexisting for different orgs).

#### Alternative 2: Runtime credential resolution

Defer credential selection to runtime — pass a credential selector into each checkout step as an input and resolve it dynamically. Rejected because it complicates the execution environment, cannot leverage GitHub Actions' type-safe step output mechanism, and moves credential routing logic out of the compiler where it is most easily audited and tested.

#### Alternative 3: Flat pre-minting (mint all app tokens before any checkout)

Emit all `checkout-app-token-N` steps as a block before any `actions/checkout` step, regardless of order. Rejected because it concentrates all credential operations at the start of the job, which makes the compiled YAML harder to audit and does not generalize cleanly when checkout steps are interleaved with other job steps.

### Consequences

#### Positive
- Each checkout is authenticated with the minimum required credentials for its target org (least privilege).
- The checkout manifest gives safe-outputs handlers deterministic, zero-network token resolution at runtime.
- The integration test suite (four scenarios: PATs-only, Apps-only, mixed, step ordering) provides compiler regression coverage for all credential combinations.
- Step ordering is enforced at compile time — runtime forward-reference errors on `steps.<id>.outputs.token` are structurally impossible.

#### Negative
- Compiled YAML grows by one step per `github-app` checkout entry, increasing job startup latency slightly for app-heavy workflows.
- Index-based step IDs (`checkout-app-token-N`) are position-sensitive: reordering the `checkout:` list in a workflow definition changes the IDs of emitted steps and breaks any hardcoded references to them in workflow bodies.
- The checkout manifest env-var schema (`GH_AW_CHECKOUT_*`) is a new convention that all safe-outputs handlers must be updated to understand.

#### Neutral
- `persist-credentials: false` becomes the enforced default for all cross-org checkouts; this is consistent with security best practice but diverges from `actions/checkout`'s own default (`true`).
- Mixed-auth workflows (PATs and GitHub Apps coexisting) are fully supported, but only GitHub App entries incur the extra minting step — PAT entries emit no additional steps.
- The `indexOf` helper used in the ordering test (`TestMultiCheckoutCrossOrgStepOrdering`) operates on raw compiled YAML bytes rather than a parsed AST; this is intentional to test the actual emitted text, not an intermediate representation.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
