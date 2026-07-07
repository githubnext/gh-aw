# ADR-44037: Read-Only Memory Store Access for Custom Jobs via `restore-memory`

**Date**: 2026-07-07
**Status**: Draft
**Deciders**: pelikhan, copilot-swe-agent

---

### Context

Memory stores (`cache-memory`, `repo-memory`, `comment-memory`) in the gh-aw workflow compiler were exclusively accessible to the built-in agent job. Custom jobs (defined under `jobs:` in a workflow's frontmatter) had no way to read from these stores, making orchestrator patterns architecturally impossible. A common need is a scheduled or dispatch job that reads `cache-memory` to build a task-dispatch list before spawning the agent — but with no access path, authors had to resort to duplicating state or restructuring workflows entirely around the agent job.

### Decision

We will add a `restore-memory` field to the custom job config schema that allows job authors to explicitly opt in to read-only restore steps for any subset of the three memory types. The compiler injects the corresponding setup and restore steps (gh-aw setup action, `actions/cache/restore`, `clone_repo_memory_branch.sh`, or `setup_comment_memory_files.cjs`) directly into the custom job's step list, positioned after GHES host config and before `pre-steps`/`steps`. No write-back, git-commit, or artifact-upload steps are ever emitted for custom jobs — enforcement is structural (using `actions/cache/restore` instead of `actions/cache`), not advisory.

### Alternatives Considered

#### Alternative 1: Full Read-Write Memory Access in Custom Jobs

Allow custom jobs to both read and write to memory stores, mirroring the agent job's full lifecycle. This would give orchestrators maximum flexibility, including the ability to pre-populate or update state.

Rejected because write-back requires coordinated git push logic, artifact uploads, and integrity-check steps that are complex and dangerous when multiple jobs run concurrently. Orchestrator jobs typically only need to read shared state, not modify it. Allowing writes would require the same mutex/conflict-resolution machinery already present in the agent job, adding significant complexity for minimal gain.

#### Alternative 2: Expose Memory via Workflow Outputs or Job Artifacts

Have the agent job export its memory state as a named workflow output or GitHub Actions artifact, which other jobs could then consume via the standard `needs:` dependency mechanism.

Rejected because this approach requires the agent job to run first and explicitly export state — it cannot be used in pre-dispatch orchestrator patterns where the orchestrator runs *before* the agent. It also conflates memory (a persistent side channel) with ephemeral per-run job outputs, muddying the conceptual model. Existing memory restore machinery (`generateRepoMemorySteps`, `generateCacheMemoryRestoreLines`) is already well-tested and reusable, making step injection lower risk than a new artifact-based path.

### Consequences

#### Positive
- Enables orchestrator job patterns: a non-agent job can read `cache-memory` state (e.g., a dispatch list or rate-limit counter) before deciding whether and how to invoke the agent.
- Read-only is enforced structurally — `actions/cache/restore` never auto-saves, so concurrent orchestrators cannot corrupt the memory store.
- Reuses existing, tested step generators (`generateRepoMemorySteps`, `generateCommentMemoryRestoreLines`), minimising new code surface.
- Validation at compile time rejects requests for memory types not declared in `tools:`, surfacing misconfiguration early.

#### Negative
- Increases compiler complexity: a new source file (`compiler_custom_job_memory.go`), new config parsing, validation, and injection plumbing into `configureCustomJobSteps`.
- Job authors must explicitly enumerate every memory type they need under `restore-memory:`; there is no auto-detection based on what the workflow's `tools:` section declares.
- The injected setup action step (required for repo-memory and comment-memory scripts) adds overhead to custom jobs that may not otherwise need the full gh-aw setup.

#### Neutral
- The JSON schema for the custom job `additionalProperties` definition gains a new `restore-memory` object property; existing workflows without this field are unaffected.
- Step ordering is deterministic: GHES host config → gh-aw setup (if needed) → memory restore steps → pre-steps → regular steps.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
