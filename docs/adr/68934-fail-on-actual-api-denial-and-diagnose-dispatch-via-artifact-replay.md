# ADR-68934: Fail on Actual API Denial Instead of Repository-Role Inference, and Diagnose Work-Queue Dispatch Through Read-Only Artifact Replay

**Date**: 2026-10-20
**Status**: Draft
**Deciders**: @pelikhan (PR author); maintainer acceptance pending

---

### Context

An ESLint dispatcher run committed three Claims but failed before recording any worker launch. The failure path discarded the durable request, commit, and reservation identities from the control receipt and reported only a coarse `queue_control_authority_or_publication_failed` reason, so operators could not tell what had actually gone wrong or which durable state already existed. At the same time, the dispatch credential preflight in `actions/setup/js/work_queue_dispatch_credential.cjs` inferred token capability from repository-role metadata (`permissions.push` / `maintain` / `admin` / `contents`), which GitHub does not guarantee reflects the permissions of the job's configured credential — the API enforces authorization per call. Finally, there was no supported way to re-examine a past dispatch run: diagnosing the original failure required re-running the workflow, which is not safe for a queue with reservations and immutable Claims.

### Decision

We will make the work queue trust **actual GitHub API authorization** rather than repository-role metadata: the preflight keeps credential/client binding, repository and principal identity, Claim ownership, and immutable worker-revision checks, prepares the required ref before START, and lets real API denials stop processing. We will preserve durable identities (`request_id`, `commit_id`, reservation state) on failure, isolate per-assignment launch failures so independent assignments continue, and attach fixed, human-readable `explanation` and `failure_stage` fields plus privacy-preserving logging (`work_queue_logging.cjs`, with monotonic `+Nms` deltas) to receipts, step summaries, and Actions annotations without printing raw exceptions, credentials, or task payloads. We will add a bounded, read-only `gh aw work-queue replay` mode (`pkg/cli/work_command_replay.go`) that reconstructs state from an exact Actions activation artifact (`--run-id`) or a downloaded snapshot (`--file`), with optional scheduling prediction at capture time, so failures can be diagnosed without re-executing effects.

### Alternatives Considered

#### Alternative 1: Keep role-based preflight and add more fine-grained role checks

We could have retained the repository-role gate and extended it (for example, by also inspecting installation token scopes). This fails closed earlier and gives a friendlier message, but repository roles describe the *actor's* relationship to the repo, not the scopes of the token the job actually uses; it produces both false denials (fine-grained tokens with `contents: write` but no role metadata) and false confidence. It was rejected because it cannot be made correct, only more elaborate.

#### Alternative 2: Diagnose failures by re-running the dispatcher workflow

Instead of replay, operators could re-run the original workflow with `DEBUG=*` enabled. This requires no new CLI surface, but a re-run is not read-only: it can re-reserve capacity, re-launch workers, and mutate Claim lifecycle state; it also cannot reproduce the exact state captured at the time of the original failure, since the queue has moved on. Rejected because the diagnostic act must not have side effects on a queue holding durable reservations.

#### Alternative 3: Propagate raw exception text into receipts and logs

Simply surfacing `error.message`/stack would have been the cheapest fix for the original "hidden cause" problem. Rejected because exception text from the GitHub client and queue codec can embed tokens, repository internals, and task payloads; a fixed vocabulary of error codes with mapped explanations (`queueControlErrorCode` / `queueControlErrorExplanation`) keeps diagnostics safe and stable to assert on in tests.

### Consequences

#### Positive
- Dispatch no longer rejects legitimate credentials whose permissions are correct but whose repository-role metadata is absent or misleading; authorization outcomes now match what GitHub actually enforces.
- Failures keep durable request/commit/reservation identities and a stable, human-readable explanation, so operators can recover without guessing which Claims were already committed.
- A single blocked assignment no longer aborts the whole batch; independent assignments still process.
- Past failures can be re-examined offline and deterministically from the exact Actions artifact, with no risk of re-launching workers or consuming capacity.

#### Negative
- Misconfigured tokens now fail later, at the first real API call, instead of during preflight; the symptom surfaces as an API denial rather than an up-front capability error.
- The fixed explanation vocabulary must be maintained alongside new error codes, and deliberately omits the raw exception, so some root causes remain unproven from receipts alone (the original run's exact exception is still unknown).
- New CLI surface (`replay --run-id` / `--file`) plus ~530 lines in `pkg/cli` add maintenance and test burden, and the replay path must stay bounded and read-only as the queue protocol evolves.

#### Neutral
- Replay is explicitly *not* a verification tool: it does not validate live credentials, reconcile reservations, launch workers, or repeat effects, and a snapshot cannot show boundaries that occurred after capture.
- Logging gains monotonic `+Nms` per-namespace deltas, including for always-visible diagnostics, changing the shape of existing log lines.
- Receipt schema grows optional `explanation`, `failure_stage`, and `commit_id` fields; consumers must tolerate their absence on older receipts.
- Shipped as a minor change (`.changeset/minor-work-queue-artifact-replay.md`) with reference documentation updates in `docs/src/content/docs/reference/work-queue.md`.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
