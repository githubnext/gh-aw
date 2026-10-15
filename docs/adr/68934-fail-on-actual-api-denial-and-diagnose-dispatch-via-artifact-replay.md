# ADR-68934: Fail on Actual API Denial Instead of Repository-Role Inference, and Diagnose Work-Queue Dispatch Through Read-Only Artifact Replay

**Date**: 2026-10-15
**Status**: Proposed (implemented in PR #68934; maintainer acceptance pending)
**Decision owner**: @pelikhan, the PR author. Formal architecture approval is not implied by this record.

---

### Context

The [ESLint dispatcher run 38645334879](https://github.com/github/gh-aw/actions/runs/38645334879) committed three Claims but failed before recording any worker launch. The failure path discarded the durable request, commit, and reservation identities from the control receipt and reported only a coarse `queue_control_authority_or_publication_failed` reason, so operators could not tell what had actually gone wrong or which durable state already existed. The exact original exception remains unknown; an absent launch receipt is not proof that no launch occurred.

The dispatch credential preflight in `actions/setup/js/work_queue_dispatch_credential.cjs` inferred token capability from repository-role metadata (`permissions.push` / `maintain` / `admin` / `contents`), which GitHub does not guarantee reflects the permissions of the job's configured credential. GitHub enforces authorization per API call using the selected job/token permissions. The requirement is to remove that inference without weakening credential identity, immutable revision, ledger ownership, or live authorization checks.

Existing activation artifacts capture an earlier ledger state, not necessarily the state after the control step committed Claims or attempted launches. A supported read-only artifact inspection path was missing; re-running a dispatcher to obtain diagnostics can mutate a queue that already holds reservations and immutable Claims.

### Decision

The work queue trusts **actual GitHub API authorization** rather than repository-role metadata: the preflight keeps credential/client binding, repository and principal identity, Claim ownership, and immutable worker-revision checks, prepares the required ref before START, and lets real API denials stop the affected operation. This does not grant permissions, replace the compiler-selected credential, or broaden the agent job's authority.

Preserve durable identities (`request_id`, `commit_id`, reservation state) when launch processing fails after publication, and isolate per-assignment launch failures so independent assignments can continue. Add developer-authored `explanation` and `failure_stage` fields to failure receipts, use those diagnostics in step summaries and Actions annotations, and emit privacy-preserving logs (`work_queue_logging.cjs`, with monotonic `+Nms` deltas). Error classification uses a fixed vocabulary with a generic fallback rather than printing raw exceptions, credentials, or task payloads. A blocked receipt does not release reservations or establish authoritative nonlaunch evidence; recovery still requires ledger inspection and trusted reconciliation.

Add bounded, read-only artifact replay to `gh aw work-queue replay` (`pkg/cli/work_command_replay.go`). `--run-id` downloads a matching Actions activation snapshot and validates repository, workflow, run, and attempt identity; missing, ambiguous, expired, or invalid artifacts fail closed. `--file` reads a downloaded snapshot without network access. Optional scheduling prediction evaluates the captured policy and ledger at capture time, without refreshing live dependencies, validating dispatch credentials, creating tags, granting Claims, or launching workers.

### Alternatives Considered

These alternatives describe tradeoffs assessed for this record, not a claim that each was previously implemented or formally reviewed.

#### Alternative 1: Keep role-based preflight and add more fine-grained role checks

We could have retained the repository-role gate and extended it (for example, by also inspecting installation token scopes). This fails closed earlier and gives a friendlier message, but repository roles describe the *actor's* relationship to the repo, not the scopes of the token the job actually uses; it produces both false denials (fine-grained tokens with `contents: write` but no role metadata) and false confidence. It was rejected because it cannot be made correct, only more elaborate.

#### Alternative 2: Diagnose failures by re-running the dispatcher workflow

Instead of replay, operators could re-run the original workflow with `DEBUG=*` enabled. This requires no new CLI surface, but a re-run is not read-only: it can re-reserve capacity, re-launch workers, and mutate Claim lifecycle state; it also cannot reproduce the exact state captured at the time of the original failure, since the queue has moved on. Rejected because the diagnostic act must not have side effects on a queue holding durable reservations.

#### Alternative 3: Propagate raw exception text into receipts and logs

Simply surfacing `error.message`/stack would have been the cheapest fix for the original "hidden cause" problem. Rejected because exception text from the GitHub client and queue codec can embed tokens, repository internals, and task payloads; a fixed vocabulary of error codes with mapped explanations (`queueControlErrorCode` / `queueControlErrorExplanation`) keeps diagnostics safe and stable to assert on in tests.

#### Alternative 4: Abort the batch or roll back Claims on any launch failure

Stopping the batch leaves independent assignments unprocessed after their Claims are durable. Automatically rolling back or releasing reservations is unsafe when a failed API acknowledgement does not prove that no worker launched. Preserve the durable grant, report the affected assignment's failure, continue independent launch processing, and leave lifecycle recovery to authoritative reconciliation.

### Consequences

#### Positive
- Dispatch no longer rejects legitimate credentials whose permissions are correct but whose repository-role metadata is absent or misleading; authorization outcomes now match what GitHub actually enforces.
- Failures keep durable request/commit/reservation identities and a stable, human-readable explanation, so operators can recover without guessing which Claims were already committed.
- A single blocked assignment no longer aborts the whole batch; independent assignments still process.
- Captured ledger state can be re-examined deterministically, offline with `--file`, without re-launching workers or consuming capacity. Artifact downloads require read access, not dispatch authority.

#### Negative
- Misconfigured tokens now fail later, at the first real API call, instead of during preflight; the symptom surfaces as an API denial rather than an up-front capability error.
- The fixed explanation vocabulary must be maintained alongside new error codes, and deliberately omits the raw exception, so some root causes remain unproven from receipts alone (the original run's exact exception is still unknown).
- New CLI surface (`replay --run-id` / `--file`) adds maintenance and test burden; artifact selection, provenance validation, resource bounds, and read-only behavior must remain correct as Actions attempts and the queue protocol evolve.

#### Neutral
- Replay validates captured ledger structure and provenance, not live authorization or downstream execution. The original artifact captures an earlier ledger; its `native_capacity_blocked` scheduling prediction is not proof of the original launch failure's cause. No live worker completion has been verified for these fixes.
- Logging gains monotonic `+Nms` per-namespace deltas, including for always-visible diagnostics, changing the shape of existing log lines.
- Receipt schema grows optional `explanation`, `failure_stage`, and `commit_id` fields; consumers must tolerate their absence on older receipts.
- Shipped as a minor change (`.changeset/minor-work-queue-artifact-replay.md`) with reference documentation updates in `docs/src/content/docs/reference/work-queue.md`.

---

Implementation: [PR #68934](https://github.com/github/gh-aw/pull/68934). Maintainer acceptance remains pending.
