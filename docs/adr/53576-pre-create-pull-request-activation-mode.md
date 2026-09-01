# ADR-53576: Create a Steering Issue During Activation

**Date**: 2026-08-18
**Status**: Draft
**Deciders**: pelikhan, copilot-swe-agent

---

### Context

Agentic workflows compiled by gh-aw can run for minutes or hours. Users need a stable place to provide feedback while the agent is running, but creating a pull request early requires a branch, an empty commit, elevated permissions, special checkout behavior, and cleanup. Failures also created a separate issue, splitting steering and diagnostics across two resources.

### Decision

The opt-in `safe-outputs.steer: true` setting creates a run-scoped issue during activation. The injected prompt identifies that issue and directs the agent to read user-authored comments containing `steer`. Pull request creation and checkout continue through the normal safe-output path without a pre-created branch.

On success, the conclusion job closes the steering issue and links the created pull request when available. On failure, the failure handler retitles and updates the same issue with the normal failure report instead of creating a second issue.

### Alternatives Considered

#### Alternative 1: Pre-create a draft pull request

Create an empty commit, a run-scoped branch, a draft pull request, and a check run during activation. This provides early PR visibility but requires special branch validation, checkout overrides, PR reuse logic, and cleanup.

#### Alternative 2: Reuse the triggering issue or pull request

Use the event's issue or pull request for feedback. This avoids creating a resource but is unavailable for scheduled and manually dispatched workflows and can mix unrelated conversations.

### Consequences

#### Positive
- Users have a stable steering surface for every trigger type.
- Agent feedback and failure diagnostics share one issue.
- Pull request branch creation, validation, and checkout retain their normal behavior.
- Activation needs only `issues: write` for steering instead of contents, pull-request, and checks write access.

#### Negative
- Each steered run creates an issue even when no pull request is ultimately produced.
- Cancellation or failure before conclusion can leave the steering issue open for manual triage.
- Steering cannot use a separate `failure-issue-repo` because the run-scoped issue is reused.

#### Neutral
- The compiler requires top-level `issues: read` so the agent's GitHub MCP tools can read steering comments.
- Staged mode does not create a steering issue.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
