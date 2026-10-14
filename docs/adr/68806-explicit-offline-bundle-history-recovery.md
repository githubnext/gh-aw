# ADR-68806: Make PR bundle history recovery explicit and validate bundles offline

**Date**: 2026-10-14
**Status**: Draft
**Deciders**: Unknown (PR #68806 authored by @pelikhan)

---

### Context

Safe-output pull-request handlers (`create_pull_request` and `push_to_pull_request_branch`) import agent-produced Git bundles and validate them before publishing. A previous change (#68717, reverted in #68795) made these handlers recover missing history automatically by fetching, deepening, or unshallowing the repository. In shallow or partially-cloned monorepo checkouts that escalation can trigger effectively unbounded object downloads, and it does so implicitly on a path that runs with checkout credentials. At the same time, validation still needs a trusted baseline (the base-branch tip) and access to enough local history and content to run protected-file, transient-history, octopus-merge, patch-size, and expanded-content checks. The handlers must therefore keep validating correctly in shallow checkouts without silently fetching objects to do so.

### Decision

We will make history recovery an explicit, per-handler opt-in and make bundle validation offline by default. Trusted base tips are resolved with a metadata-only, exact-ref `git ls-remote --refs --exit-code` using the target-checkout credentials (never fork-head credentials), with the returned SHA validated and pinned; no object fetch is ever performed to establish an optional baseline. Bundles are imported with `git bundle unbundle` under `GIT_ALLOW_PROTOCOL=""` and `GIT_NO_LAZY_FETCH=1`, so import, prerequisite checks, and ancestry/tree inspection cannot reach the network even on Git versions that ignore lazy-fetch suppression. When trusted metadata or local history/content is insufficient, validation falls back conservatively to PR-head validation, and missing required content fails rather than bypassing a check. Automatic fetch/deepen/unshallow escalation is removed from both handlers and replaced by an `allow-full-history-fetch` boolean that lives on each applicable handler (not on the global `safe-outputs` object), defaults to `false`, and accepts only literal booleans. The primary driver is security and cost predictability: network access and unbounded downloads must be a deliberate, locally-scoped configuration choice.

### Alternatives Considered

#### Alternative 1: Restore automatic prerequisite recovery with bounds (depth limits, blob filters, fetch-by-SHA)

This is the reverted #68717 approach, optionally constrained by `--depth`, `--filter=blob:none`, or fetching exact SHAs. It was considered because it keeps shallow checkouts working with zero configuration. It was rejected because none of those knobs is a byte bound — a depth-limited or filtered fetch in a large monorepo can still transfer arbitrarily much — and because implicit network access on a credentialed handler path is exactly the behaviour the revert was meant to stop.

#### Alternative 2: A single global `safe-outputs.allow-full-history-fetch` flag

One top-level switch would be simpler to document and to set once per workflow. It was rejected because it couples two independently risky handlers: enabling recovery for `create-pull-request` would silently enable it for `push-to-pull-request-branch` as well. Per-handler settings keep the blast radius of the opt-in scoped to the handler that actually needs it, and the schema change (`pkg/parser/schemas/main_workflow_schema.json`) rejects the global/string/expression forms explicitly.

#### Alternative 3: Always require a deep (non-shallow) checkout for bundle-publishing workflows

Documenting "use `fetch-depth: 0`" and failing otherwise would remove recovery logic entirely. It was rejected as too blunt: many shallow checkouts already contain the required objects and validate fine offline, and forcing full clones would impose the worst-case download cost on every run instead of only on the runs that need it.

### Consequences

#### Positive
- Bundle import and validation perform no network I/O by default, so a malicious or malformed bundle cannot induce fetches with checkout credentials.
- Download cost becomes predictable: unbounded history transfer only happens when a maintainer sets `allow-full-history-fetch: true` on a specific handler.
- Baseline discovery is metadata-only and pinned to an exact ref, eliminating object transfer for optional baselines and removing fork-head credential use from the trusted path.
- Errors are actionable and credential-safe: they explain how to provision objects during authenticated checkout or how to enable the opt-in, using fixed diagnostics without raw command output.

#### Negative
- Partial clones can now fail offline when required blobs/trees are absent even though the commit objects exist, turning a previously self-healing case into a workflow failure that needs configuration.
- Conservative fallback to PR-head validation means some runs validate against a weaker baseline than the true merge base, reducing the precision of history-based checks.
- The opt-in adds configuration surface duplicated across two handlers, so maintainers can enable it on one handler and still hit failures on the other.
- `GIT_ALLOW_PROTOCOL=""` plus `GIT_NO_LAZY_FETCH=1` must be threaded through every Git invocation on these paths; any future call site that forgets them silently reopens the network hole.

#### Neutral
- Shallow checkouts with locally available prerequisites behave exactly as before; shallow boundaries are unchanged.
- Workflow-start SHAs and agent-supplied metadata are explicitly not treated as trusted provenance.
- Ordinary PR-head fetch/push and the existing signed-push recovery path are out of scope for this change.
- The change spans JS handlers (`actions/setup/js/`), Go compiler/registry code (`pkg/workflow/`), the workflow JSON schema (`pkg/parser/schemas/`), and reference docs, so the new option is validated at compile time as well as at run time.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
