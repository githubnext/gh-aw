# ADR-28290: Base Branch Restore Must Precede Pre-Agent Steps in Workflow Compilation

**Date**: 2026-04-24
**Status**: Draft
**Deciders**: Unknown (bot-authored fix, see PR #28290)

---

## Part 1 — Narrative (Human-Friendly)

### Context

The gh-aw workflow compiler (`generateMainJobSteps`) emits a fixed sequence of CI steps into every generated GitHub Actions main job. Two of those steps interact in a safety-critical way: (1) "Restore agent config folders from base branch" snapshots trusted base-branch files (including `.github/skills/`) back over the PR checkout, and (2) `pre-agent-steps` (APM restore) writes agent-managed skill files into `.github/skills/`. When the base-restore step ran *after* pre-agent-steps, it silently clobbered any skills that APM had just placed there. In `workflow_dispatch` runs this bug was invisible because the base-restore step is skipped entirely (no PR checkout); the regression only surfaced in `pull_request` triggers.

### Decision

We will enforce a strict ordering invariant in the compiler: "Download activation artifact" and "Restore agent config folders from base branch" must be emitted **before** `generatePreAgentSteps` in every generated main job. This ordering guarantees that the base snapshot is fully applied first, so APM-restored skills written by pre-agent-steps are never overwritten. The new canonical order is: download artifact → base restore → pre-agent-steps → MCP setup → agent execution.

### Alternatives Considered

#### Alternative 1: Double-restore (re-run APM after base restore)

Run pre-agent-steps as before (before base restore), then re-run a second APM restore pass after the base restore completes. This would ensure APM-restored files survive. It was rejected because pre-agent-steps can be expensive and non-idempotent (e.g., package installs, network calls), making a double-run impractical and error-prone.

#### Alternative 2: Selective base restore (skip APM-owned files)

Track which files APM writes and exclude them from the base restore. This would require APM contributors to explicitly declare all file paths they own — a coordination burden that scales poorly as new APM skills are added and creates an implicit contract between two otherwise independent systems. It was rejected because it trades a simple ordering rule for a fragile manifest-based contract.

### Consequences

#### Positive
- APM-restored skills (e.g., `.github/skills/`) survive `pull_request` runs, eliminating the silent-clobber regression.
- Behavior is now consistent between `pull_request` and `workflow_dispatch` triggers.
- The ordering invariant is explicitly codified in two new test cases (`TestImportedPreAgentStepsRunAfterPRBaseRestore`, `TestImportedPreAgentStepsRunAfterPRBaseRestoreCopilot`), preventing future regressions.

#### Negative
- The activation artifact download now happens earlier in the job (before MCP setup), so any artifact-download latency is on the critical path before MCP starts. Previously, a failed artifact download was deferred.
- The step ordering in `generateMainJobSteps` carries more implicit constraints; future contributors adding steps in this function must understand the ordering invariant or risk reintroducing the bug.

#### Neutral
- All generated lock files (golden files) required regeneration to reflect the new step positions; this is a large diff but purely mechanical.
- The fix applies uniformly to all engines (claude, copilot, etc.) since step emission goes through the same `generateMainJobSteps` function.

---

## Part 2 — Normative Specification (RFC 2119)

> The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this section are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### Step Ordering in Generated Main Jobs

1. Implementations **MUST** emit the "Download activation artifact" step before any `pre-agent-steps` in the generated main job.
2. Implementations **MUST** emit the "Restore agent config folders from base branch" step before any `pre-agent-steps` in the generated main job when `ShouldGeneratePRCheckoutStep` is true.
3. Implementations **MUST NOT** insert any step that writes to `.github/` agent folders (e.g., `.github/skills/`) between the base-restore step and its own completion.
4. Implementations **SHOULD** maintain the canonical ordering: artifact download → base restore → pre-agent-steps → MCP setup → agent execution.
5. Implementations **MAY** add steps before the artifact download provided those steps do not depend on activation artifact content or agent config folder state.

### Test Coverage

1. Implementations **MUST** include a test that verifies the "Restore agent config folders from base branch" step index is strictly less than the APM restore step index in a generated `pull_request` workflow.
2. Implementations **MUST** include this ordering test for each supported engine (e.g., `claude`, `copilot`) that can be triggered by a `pull_request` event.

### Conformance

An implementation is considered conformant with this ADR if it satisfies all **MUST** and **MUST NOT** requirements above. Failure to meet any **MUST** or **MUST NOT** requirement constitutes non-conformance.

---

*This is a DRAFT ADR generated by the [Design Decision Gate](https://github.com/github/gh-aw/actions/runs/24896846777) workflow. The PR author must review, complete, and finalize this document before the PR can merge.*
