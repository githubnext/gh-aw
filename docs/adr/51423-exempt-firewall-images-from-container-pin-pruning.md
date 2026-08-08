# ADR-51423: Exempt gh-aw-firewall Images from Container Pin Pruning and Surface Unresolved Pins as Resolution Failures

**Date**: 2026-08-08
**Status**: Draft
**Deciders**: pelikhan, copilot-swe-agent

---

### Context

`PruneStaleContainerPins` removes any container pin entry no longer referenced by locally compiled lock files, keeping `actions-lock.json` tidy as image versions advance. When `constants.DefaultFirewallVersion` is bumped in gh-aw, the _previous_ default version's pin becomes unreferenced by local lock files and gets pruned. `make sync-action-pins` then mirrors that loss into the embedded `pkg/actionpins/data/action_pins.json` / `pkg/workflow/data/action_pins.json` catalogs shipped in the binary, silently breaking digest pinning for any consumer workflow that explicitly pins to that now-previous version — with no trace in `resolution_failures`. This same regression has occurred four times: gh-aw#38561 (v0.79.x), gh-aw#43307 (v0.82.2), gh-aw#44040 (v0.82.3), and gh-aw#51248 (v0.86.1). Firewall images (`ghcr.io/github/gh-aw-firewall/*`) are the most security-load-bearing containers in the system because they confine the agent sandbox; a mutable tag there means a silent image swap on the next `download_docker_images.sh` invocation.

### Decision

We will exempt all images under `constants.DefaultFirewallRegistry` from `PruneStaleContainerPins` so that once a firewall image digest is resolved it is retained in `actions-lock.json` permanently, regardless of whether it is still referenced by any local lock file. We will also add a `resolution_failures` manifest entry whenever a firewall image reaches `applyContainerPins` without a cached or embedded pin, making the gap auditable instead of silently emitting a bare tag. Two regression tests — one asserting that all compiled manifest containers carry a `pinned_image`, one asserting that an unresolvable firewall version records a `resolution_failures` entry — catch future occurrences at the source.

### Alternatives Considered

#### Alternative 1: Add a golden test only, leave pruning logic unchanged

A golden/compile test asserting that every `containers[]` entry carries a `pinned_image` would catch the regression in CI, but only _after_ the binary is already broken. The root cause — pruning dropping the pin from the embedded catalog — would remain, requiring a manual re-pin on each `DefaultFirewallVersion` bump.

#### Alternative 2: Introduce a `permanent: true` field on ContainerPin entries

A generic `never_prune` or `permanent` flag on individual `ContainerPin` records would generalize the exemption mechanism beyond the firewall registry prefix. This is more flexible but adds schema complexity, requires migrating existing pin records, and the only current use case is firewall images — making the added abstraction premature.

### Consequences

#### Positive
- Firewall image digest pins survive `DefaultFirewallVersion` bumps without manual intervention, breaking a four-occurrence regression cycle.
- Unresolvable firewall pins are now auditable via `resolution_failures` in the compiled manifest rather than silently shipping bare tags.
- The two new compile-level regression tests make this class of bug detectable before it reaches consumers.

#### Negative
- Historical firewall pin entries accumulate indefinitely in `actions-lock.json` and the embedded catalogs — one entry per resolved version (a few KB each, acceptable but nonzero growth over time).
- The exemption is encoded as a registry-prefix check against `constants.DefaultFirewallRegistry`, introducing a special case in the generic pruning logic that future developers must understand.

#### Neutral
- Existing pins for the current default firewall version are unaffected; only behavior on subsequent `DefaultFirewallVersion` bumps changes.
- The updated `TestPruneStaleContainerPins` and `TestUpdateContainerPins_PrunesStaleEntries` tests now use a generic non-firewall image family to exercise the normal pruning path, separating concerns between the two behaviors.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
