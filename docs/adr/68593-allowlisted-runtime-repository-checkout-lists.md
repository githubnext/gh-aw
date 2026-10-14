# ADR-68593: Allowlisted Runtime Repository Checkout Lists

**Date**: 2026-10-13
**Status**: Draft
**Deciders**: Unknown (drafted from PR #68593 evidence)

---

### Context

Scheduled agentic workers need to operate over a set of repositories that changes between runs, but `checkout:` previously resolved every repository at compile time, so changing the set required editing and recompiling the workflow. Allowing a GitHub Actions expression to supply the repository list at runtime moves repository selection into data that can be produced by other steps or external state, which is a privileged decision: the checkout token, LFS endpoints, and submodule URLs are all attacker-reachable if the selection is unconstrained. The compiler must therefore decide, at compile time, what runtime selections are admissible, where the resulting working trees live, and how credentials are cleaned up across an arbitrarily deep expanded path layout. This change spans the Go compiler (`pkg/workflow/checkout_*.go`), the generated JavaScript action (`actions/setup/js/checkout_dynamic.cjs`), the frontmatter schema, and the agent credential-cleanup shell step.

### Decision

We will support runtime repository lists by accepting a newline-separated set of repository slugs from a GitHub Actions expression in `checkout.repository` (or its new `repo` alias), and we will require that any such expression be paired with a mandatory `allowed-repos` policy and a literal, non-root `path` prefix — with a single exemption for the standalone `${{ github.repository }}` expression, which keeps today's root-checkout behaviour. Selections are validated as a batch before any checkout runs: canonical names are resolved through the API, scoped wildcard patterns (`owner/*`, `*/repo`) filter the batch, unrestricted `*` policies and wildcards inside selectors are rejected, and public workflow repositories may not select private or internal targets — unverifiable metadata fails closed. Each selected repository is expanded to `prefix/owner/repo`, recorded in the shared checkout manifest, and `lfs: true` and enabled submodules are rejected for runtime lists at both compile and run time because repository-controlled endpoints could otherwise exfiltrate the checkout token. Agent finalization cleans and verifies each expanded checkout's git config (and nested submodule configs) using the validated paths rather than bounded filesystem discovery, and checkout preparation failures are propagated into the existing agent-failure-issue reporting path.

The policy may contain literal patterns or standalone references to repository or organization variables. Caller-controlled inputs and computed expressions are not accepted as policy sources: a caller must not be able to authorize its own repository selections. Both the agent and safe-output jobs build manifests from their finalized checkouts, and dynamic safe-output authentication follows the effective PR checkout token while preserving per-entry overrides.

### Alternatives Considered

#### Alternative 1: Allow runtime repository expressions without a mandatory allowlist

The simplest design would trust whatever slugs the expression yields, matching how other frontmatter expressions are interpolated today. It was rejected because the checkout token is handed to `actions/checkout` for each selected repository; without a compile-time policy, any step or external data source that can influence the expression can redirect the token at an arbitrary repository, and no review of the workflow file would reveal the blast radius. Requiring `allowed-repos` keeps the set of reachable repositories visible in the compiled workflow.

#### Alternative 2: Keep compile-time repository lists and recompile when the set changes

The existing static behaviour already supports multiple checkouts and has no runtime trust problem at all. It was rejected because the motivating use case is scheduled workers whose repository set is maintained outside the workflow; requiring a commit and `gh aw compile` per change makes the workflow unusable as a worker and pushes users to hand-rolled `run:` steps with worse credential hygiene than the generated checkout path.

#### Alternative 3: Checkout every selected repository at the workspace root (or a flat path) instead of `prefix/owner/repo`

A flat layout would avoid introducing a mandatory `path` prefix and the associated breaking change. It was rejected because repository names collide across owners, root checkouts would clobber the workflow repository, and a deterministic `prefix/owner/repo` layout is what lets the manifest, safe-output jobs, and the credential-cleanup step address each tree by a validated path.

### Consequences

#### Positive
- Scheduled workers can process a changing repository set without recompilation, while the compiled workflow still states the full policy boundary via `allowed-repos`.
- Repository selection is fail-closed: unresolvable metadata, visibility mismatches, wildcard selectors, and unrestricted policies all stop the job before any selected checkout runs.
- Credential cleanup is driven by validated checkout paths and explicitly verified, closing a deep-prefix bypass that bounded filesystem discovery missed (including with `force-clean-git-credentials: true`).
- Checkout preparation failures reuse the existing agent-failure-issue reporting, so operators see them without raw dependency diagnostics being published.

#### Negative
- This is a breaking change: existing workflows using any expression other than standalone `${{ github.repository }}` in `checkout.repository` now fail to compile until an `allowed-repos` policy and a non-root `path` prefix are added.
- `lfs: true` and submodules are unavailable for runtime lists, so users who need them must fall back to static checkouts or accept a reduced working tree.
- The feature adds a non-trivial amount of new surface (a dynamic checkout action, batch validation, manifest changes, finalization verification) that must stay in sync across Go, JavaScript, schema, and docs.

#### Neutral
- `repo` is introduced as an alias for `checkout.repository`, giving two spellings for the same field.
- Parsing and compiler string-API entry points now propagate checkout validation errors that were previously ignored, which may surface pre-existing invalid configurations as new compile failures.
- The `${{ github.repository }}` exemption is deliberately narrow: compound expressions that merely contain `github.repository` do not qualify.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
