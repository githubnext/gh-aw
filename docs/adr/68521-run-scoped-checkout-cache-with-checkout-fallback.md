# ADR-68521: Run-Scoped Checkout Cache with Pinned Checkout Fallback

**Date**: 2026-10-13
**Status**: Draft
**Deciders**: pelikhan [TODO: verify additional deciders]

---

### Context

The CI workflows `cjs.yml`, `cgo.yml`, and `CWI.yml` share a single `checkout-cache` producer job that saves the whole working tree (`path: .`) to the Actions cache; every downstream job restores that cache instead of running `actions/checkout` itself. The cache key was `checkout-<family>-${{ github.head_ref || github.ref_name }}-${{ github.sha }}`, which is shared by every concurrent run of the same branch and commit, and all consumers set `fail-on-cache-miss: true`. In run 38426660613 the producer job succeeded but logged `Failed to save: Unable to reserve cache with key checkout-cjs-... another job may be creating this cache`, so seven consumer jobs failed at `Restore checkout` before executing any real check. A successful cache-save step does not guarantee that a cache entry exists, and the per-branch cleanup step deletes the same shared key, so partial reruns can also find the cache gone. The constraint is that consumers must always start from the exact tested revision (`github.sha`) without persisted credentials, and real checkout failures must still fail the job.

### Decision

We will treat the checkout cache as a best-effort optimization rather than a correctness requirement. Producer, restore, and cleanup keys are scoped to the workflow family plus `github.run_id` plus `github.sha`, so reservations and cleanup cannot collide across concurrent runs of the same commit, while staying stable across attempts of one run. Every one of the 29 consumer step definitions drops `fail-on-cache-miss: true`, gives its restore step `id: restore-checkout`, and gains an immediately following pinned `actions/checkout@v7.0.1` step guarded by `if: steps.restore-checkout.outputs.cache-hit != 'true'` with `ref: ${{ github.sha }}`, `persist-credentials: false`, and the producer's fetch depth (2 for CJS/CGO, 1 for CWI). A YAML-parsing contract test (`pkg/cli/ci_checkout_cache_workflow_contract_test.go`) enforces key agreement, fallback presence and placement, pinned revision and depth, credential isolation, absence of `restore-keys`/`continue-on-error`, and single save/cleanup steps.

### Alternatives Considered

#### Alternative 1: Keep `fail-on-cache-miss: true` and make the producer save reliable

Harden the producer (for example by retrying the save or serializing with concurrency groups) so the cache always exists before consumers run. Rejected because the reservation failure originates in the hosted cache service and cannot be fully prevented from the workflow; the logs do not even identify the competing reservation's owner, so no workflow-level change can guarantee an entry exists. It would also leave partial (failed-jobs-only) reruns broken after cleanup deletes the cache.

#### Alternative 2: Add `restore-keys` prefix fallback to the restore steps

Let a near-miss restore fall back to a cache from an earlier commit on the same branch. Rejected because it would silently run checks against a stale revision rather than the tested merge commit, trading a loud failure for a wrong-but-green result. The contract test explicitly asserts `restore-keys` is empty for this reason.

#### Alternative 3: Drop the checkout cache entirely and run `actions/checkout` in every job

Simplest and most obviously correct. Rejected to preserve the existing performance benefit of the cache on the common path; the chosen design keeps the fast path and only pays for checkout when the cache is genuinely missing. [TODO: verify measured cache vs. checkout timing]

### Consequences

#### Positive
- Transient cache-reservation failures no longer fail unrelated CI jobs; the job falls back to a real checkout of the same revision.
- Failed-job-only reruns work even after the cleanup step deleted the run's cache entry.
- `github.run_id` scoping removes cross-run reservation and cleanup collisions for the same commit.
- The contract test prevents silent drift as new consumer jobs are added to the three workflows.

#### Negative
- Each of the 29 consumer steps now carries a duplicated fallback block, increasing workflow verbosity and the cost of future edits.
- A genuine cache miss is now slower (full checkout) and no longer surfaces as an explicit failure, so cache regressions can go unnoticed without monitoring.
- `github.run_id` in the key means caches are never reused across runs, raising cache churn and storage turnover.
- The contract test couples the Go test suite to the exact YAML shape of three CI workflows, so benign refactors of those files can fail tests.

#### Neutral
- `persist-credentials: false`, existing `contents: read` permissions, and all action pins are unchanged; no PAT or credential escalation is introduced.
- Fetch depth differs intentionally by workflow family (2 for CJS/CGO, 1 for CWI), mirroring each producer.
- CWI retains `enableCrossOsArchive: true` on both save and restore.
- Hosted cache service behaviour is not simulated locally; validation relies on workflow contract tests and `actionlint`.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
