# ADR-68806: Make PR bundle history recovery explicit and validate bundles offline

**Date**: 2026-10-14
**Status**: Proposed
**Deciders**: PR #68806 maintainers (author: @pelikhan)

---

### Context

Safe-output pull-request handlers (`create_pull_request` and `push_to_pull_request_branch`) import agent-produced Git bundles and validate their prerequisites, history, and changed content before publishing. [PR #68717](https://github.com/github/gh-aw/pull/68717), merged as [96b2d13](https://github.com/github/gh-aw/commit/96b2d13e5d3abe1240deed2a93f7b26db7d2a8e0), changed the push handler to fetch the live base-branch tip from the target checkout's origin so validation could distinguish trusted base changes from agent changes; automatic bundle-prerequisite recovery already existed. [PR #68795](https://github.com/github/gh-aw/pull/68795), commit [392e085](https://github.com/github/gh-aw/commit/392e085cc15345c181314bb6841184fc99ce34d4), reverted that change because fetching the live tip in massive shallow monorepos could download unrestricted reachable history. Partial clones also risk implicit downloads during local object inspection, while fork PRs may require prerequisites available only from the head repository under different credentials. The design must preserve validation without implicit object downloads, distinguish target-base provenance from head-history recovery, and make any potentially unbounded recovery a deliberate choice.

### Decision

History recovery will require `allow-full-history-fetch: true` on the specific `create-pull-request` or `push-to-pull-request-branch` handler; this default-false option accepts only literal booleans, not strings or expressions, and is not a global `safe-outputs` setting. The opt-in explicitly permits potentially unbounded prerequisite fetch and unshallow downloads, so authenticated provisioning of the necessary objects before validation is preferred.

Trusted base-tip discovery will always use a validated, exact `refs/heads/` ref and metadata-only `git ls-remote --refs --exit-code` against the trusted target origin with its own credentials, pinning the validated returned SHA; neither setting of the opt-in permits an object fetch solely to establish this optional baseline. Explicit prerequisite recovery for a fork PR will instead use the validated PR head remote and its head credentials, keeping recovery authentication separate from target-origin baseline authentication; head metadata, bundle metadata, and workflow-start SHAs do not independently establish a trusted base.

Bundle import via `git bundle unbundle`, prerequisite checks, optional baseline ancestry/tree inspection, and the push handler's bundle diff/history checks will use `GIT_ALLOW_PROTOCOL=""` together with `GIT_NO_LAZY_FETCH=1`, so protocol denial also blocks implicit lazy downloads on Git versions that do not honor lazy-fetch suppression. If trusted metadata or local ancestry cannot establish the optional baseline, the push handler will conservatively retain PR-head validation without excluding unverified base history; missing objects required for validation will fail rather than bypass checks. The primary drivers are credential isolation and avoiding implicit download costs, not a byte limit or a fully network-free handler.

### Alternatives Considered

#### Alternative 1: Keep automatic recovery and live base-tip fetching with depth limits, blob filters, or exact SHAs

Retain the convenience of automatic prerequisite recovery and restore the live-tip fetch from #68717, reducing transfer with `--depth`, `--filter=blob:none`, or exact-SHA requests. These controls can reduce some fetches, but none is a byte bound: a wide commit can contain many objects, filtered clones may later download missing content, and fetching an exact commit can include its reachable history. This alternative does not remove the implicit-download risk that motivated #68795.

#### Alternative 2: A single global `safe-outputs.allow-full-history-fetch` flag

One top-level switch would be simpler to document and set once per workflow. It would, however, enable recovery for both publishing handlers even when only one needs it. Per-handler settings make the download authorization explicit at each consumer, at the cost of repeated configuration when both handlers require recovery.

#### Alternative 3: Always require a deep (non-shallow) checkout for bundle-publishing workflows

Requiring `fetch-depth: 0` for every publishing workflow would simplify history handling, but unnecessarily downloads history when a shallow checkout and its bundle already contain the required objects. Full history alone also does not guarantee that partial-clone blobs or fork-only prerequisites are present. Authenticated, repository-aware provisioning remains the preferred operational approach; the opt-in is retained for workflows that need handler-side recovery rather than imposing a deep checkout on all workflows.

### Consequences

#### Positive
- Bundle import and local validation do not fetch objects implicitly, including from partial-clone promisor remotes.
- Metadata-only base discovery preserves trusted target-origin provenance without downloading optional baseline objects or borrowing fork-head credentials.
- Per-handler authorization makes potentially unbounded recovery explicit; diagnostics can direct maintainers to authenticated provisioning or the opt-in without exposing raw remote errors.

#### Negative
- Shallow or partial checkouts can fail when required commits, trees, or blobs are missing, replacing automatic recovery with an explicit provisioning decision.
- Conservative PR-head fallback can count legitimate base-branch changes as agent changes and reject a valid base merge; it intentionally does not relax protected-file or history checks.
- Opted-in recovery has no promised byte bound and can still be expensive or fail when the selected remote or credentials cannot supply the prerequisites. Fork-aware remote selection and credential separation add implementation and regression-test complexity.
- Both handlers need their own opt-in, and future local Git inspection calls must preserve the offline environment to avoid reintroducing implicit downloads.

#### Neutral
- A shallow checkout is not itself an error if required objects and checks are available locally; offline validation does not deepen or unshallow it.
- Metadata discovery, ordinary PR-head fetch/push, and existing signed-push recovery are separate from offline bundle validation. This decision does not claim that the complete handler performs no network I/O or that the option bounds all handler transfers.
- The option must remain consistent across the JS handlers, Go configuration/compiler/registry, workflow schema, tests, and reference documentation.

---

*This proposal records the reviewed implementation contract for PR #68806. Maintainer acceptance remains pending.*
