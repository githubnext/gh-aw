# ADR-44444: Per-Checkout GitHub App Auth for safe_outputs Git Operations

**Date**: 2026-07-09
**Status**: Draft
**Deciders**: Unknown

---

### Context

The `safe_outputs` job performs git operations (checkout, push, PR creation) on behalf of agents. Until this change, these operations used either the global `safe-outputs.github-app` setting or the default `GITHUB_TOKEN`. In cross-repo workflows—where the target repository belongs to a different organization than the workflow's home repo—the default token lacks push access, and the global safe-outputs token is coarse-grained (applies uniformly to all safe_outputs operations regardless of target). Agents that check out an external repo (e.g., `OrgB/target-repo`) need a way to supply a GitHub App credential scoped specifically to that checkout's safe_outputs git operations without altering the agent's own checkout authentication.

### Decision

We will add a `checkout.safe-output-github-app` field (with `checkout.safe-outputs-github-app` as a backward-compatible alias) to `CheckoutConfig`. This field carries a `GitHubAppConfig` used exclusively by the `safe_outputs` job when minting tokens for checkout/push operations targeting that repository. Token resolution in `resolvePRCheckoutToken` now checks the checkout manager first—preferring an explicit repository match, then `current: true`, then the default checkout—before falling back to the existing global safe-outputs token chain. The agent job's own checkout authentication is unaffected.

### Alternatives Considered

#### Alternative 1: Use the existing global `safe-outputs.github-app` setting

The global `safe-outputs.github-app` field already supports GitHub App credentials for all safe_outputs operations. Workflows could configure it once and rely on it for cross-repo pushes.

This was not chosen because the global setting is a blunt instrument: it applies to every safe_outputs operation regardless of target repository, making it unsuitable when multiple checkouts target different organizations with different app registrations. It also conflates the safe_outputs credential with the overall workflow credential, which may have wider permissions than needed for a specific checkout target.

#### Alternative 2: Add a `github-app` override field directly on each `safe_outputs` operation config

Each operation (`create-pull-request`, `push-to-pull-request-branch`) could accept its own `github-app` override, keeping auth configuration co-located with the operation that needs it.

This was not chosen because it would require duplicating the app config for every operation that targets the same repository, and it breaks the intuitive mapping between "a checked-out repository" and "the credentials used to push back to it." Anchoring the credential to the checkout config aligns with how `checkout.github-app` already works for agent auth, and lets the token-resolution logic leverage checkout ordering and the `current: true` flag.

### Consequences

#### Positive
- Cross-repo `safe_outputs` operations can use a fine-grained GitHub App token scoped to the exact target repository, without elevating the agent's own checkout credential.
- The resolution precedence (explicit repo match → `current: true` → default checkout) mirrors existing patterns in the checkout manager, keeping the mental model consistent.
- `ignore-if-missing` composes with the new field: when the app installation is absent, the resolver falls back to the global safe-outputs token chain transparently.

#### Negative
- Each checkout entry can now carry two separate GitHub App configs (`github-app` for agent auth, `safe-output-github-app` for safe_outputs auth), increasing configuration surface area and the potential for user confusion about which credential applies when.
- The backward-compatible `safe-outputs-github-app` alias adds parser complexity and a permanent dual-key code path that must be maintained.

#### Neutral
- The `resolvePRCheckoutToken` function signature gains a `*CheckoutManager` parameter; all existing callers that passed `nil` previously now pass an empty `NewCheckoutManager(nil)`, preserving existing behavior.
- Tests covering parser, checkout manager, step generator, and token resolution were added alongside the feature code, establishing coverage baselines for the new code paths.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
