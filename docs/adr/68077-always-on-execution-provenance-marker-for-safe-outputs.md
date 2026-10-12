# ADR-68077: Always-on execution provenance marker for safe outputs

**Date**: 2026-10-12
**Status**: Draft
**Deciders**: Unknown (draft generated from PR #68077)

---

### Context

Safe-output publishing (issues, pull requests, comments, discussions) currently attributes a published artifact to its originating workflow run through the human-visible footer and an XML grouping marker. Both are presentation concerns: footers can be disabled or customized per workflow, and the workflow/caller grouping marker only identifies *which* workflow ran — it cannot distinguish a specific run attempt, a reusable or matrix job instance, a publishing fallback, or a maintenance replay from the original execution. That makes after-the-fact auditing of "which exact execution created or updated this body" unreliable, and it allows agent-supplied content to spoof attribution markers. The constraint is that attribution must survive footer configuration, must be derived only from trusted runtime context (not from model output), and must fail closed rather than publish an unattributed artifact.

### Decision

We will append a versioned, machine-readable JSON execution marker (`<!-- gh-aw-execution: {"version":1,...} -->`) to every safe-output body — creates, updates, and publishing fallbacks alike — independently of the visible footer, while preserving the existing workflow/caller grouping marker. The marker is built in `actions/setup/js/execution_provenance.cjs` exclusively from validated runtime environment values (`GITHUB_REPOSITORY_ID`, `GITHUB_REPOSITORY`, `GH_AW_WORKFLOW_ID`, `GITHUB_RUN_ID`, `GITHUB_RUN_ATTEMPT`, `GITHUB_JOB`, `GH_AW_JOB_CHECK_RUN_ID`); any missing or malformed value raises before publishing, and reserved markers present in supplied content are stripped first. The compiler (`pkg/workflow/compiler_safe_outputs_steps.go`, `safe_jobs.go`, `maintenance_workflow_yaml_jobs.go`, `side_repo_maintenance.go`) supplies the publishing check-run ID so the marker pins the exact publishing job, and `resolvePublishingJobId` resolves that check run to an Actions job by paginating the recorded run attempt rather than matching on job names.

### Alternatives Considered

#### Alternative 1: Keep attribution in the visible footer only

The existing footer already renders the workflow name and run URL, so extending it would have required no new module. It was rejected because footers are explicitly user-configurable and can be disabled entirely, which would silently remove attribution exactly for the workflows most likely to need auditing; the footer is also free-form text with no stable schema for downstream tooling to parse.

#### Alternative 2: Record the logical job name instead of the publishing check-run ID

Emitting `GITHUB_JOB` alone would avoid threading a new `GH_AW_JOB_CHECK_RUN_ID` through every publishing path (built-in, custom safe jobs, and maintenance). This was a close call, but job names are not unique across matrix legs and reusable workflow instances, so the marker could not identify a single Actions job. The check-run ID is unique per job execution and can be validated against the recorded run ID and attempt, which the resolver now enforces.

### Consequences

#### Positive

- Attribution is preserved regardless of footer mode, for creates, updates, cross-repository publishing, and maintenance replays.
- The marker is structured and versioned, so auditing tools can parse it and evolve with `version` rather than scraping prose.
- Spoofed or stale `gh-aw-execution` / `gh-aw-agentic-workflow` markers in agent-supplied content are stripped before trusted metadata is appended.
- Missing or malformed runtime metadata fails closed, preventing publication of unattributed artifacts.

#### Negative

- Existing compiled workflows must be recompiled to supply the publishing check-run context; stale `.lock.yml` files will fail closed at publish time.
- Every published body grows by an extra HTML comment, and the escaping of `<`, `>`, `&`, and `-` makes the raw marker harder for humans to read.
- `resolvePublishingJobId` adds a paginated Actions API dependency and requires Actions read access in the execution repository.

#### Neutral

- Attribution semantics are defined explicitly: cross-repository outputs are attributed to the execution repository and replays to the publishing maintenance execution.
- AW workflow IDs and Actions workflow IDs, and triggering actor versus publisher identity, are now documented as distinct concepts (`docs/src/content/docs/reference/footers.md`).
- Regression coverage spans footer modes, reusable/matrix identities, fallbacks, spoofing, and fail-closed paths across the JS and Go test suites.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
