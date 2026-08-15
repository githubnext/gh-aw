# ADR-52541: Add `approve-workflow-run` Safe Output Type

**Date**: 2026-08-13
**Status**: Draft
**Deciders**: pelikhan, gh-aw maintainers

---

### Context

GitHub's fork pull request approval gate blocks workflow runs from untrusted forks until a repository maintainer explicitly approves them. AI agents operating in the gh-aw system currently cannot unblock these workflow runs programmatically — a human must visit the GitHub Actions UI and click "Approve and run" manually. This creates a bottleneck in automated workflows where an AI agent is expected to handle routine fork PR triage but cannot advance the CI pipeline without human intervention.

The safe outputs system is the established extension point for granting AI agents opt-in, permission-scoped GitHub write capabilities. Adding `approve-workflow-run` as a safe output type follows the existing pattern used by other mutation operations (e.g., `close-pull-request`, `merge-pull-request`).

### Decision

We will add `approve-workflow-run` as a new experimental opt-in safe output type backed by a dedicated handler (`approve_workflow_run.cjs` / `approve_workflow_run.go`). The compiler emits an experimental feature warning when a workflow enables it. The handler fetches the workflow run from the GitHub API, verifies that its `event` is `pull_request`, it has an associated pull request, its `status` is `waiting`, and that pull request is the triggering pull request or explicitly configured in `allowed-pull-requests`; it then calls `actions.approveWorkflowRun`. The operation requires `actions: write` and an explicit external `github-token` or GitHub App token because `github.token` cannot approve fork pull-request workflow runs. It is classified as a non-reviewable mutation under threat-detection warn mode, consistent with `merge-pull-request` and `close-pull-request`.

### Alternatives Considered

#### Alternative 1: Direct API call via a pre-authorized token in the workflow

Agents could call the GitHub REST API directly using a token with `actions: write` injected as a workflow secret, bypassing the safe output layer entirely.

This was not chosen because it skips the safe output sandboxing guarantees — per-handler max counts, staged preview mode, GitHub App token rotation, and threat-detection classification — all of which exist to make AI-driven mutations auditable and rate-limited. Circumventing these controls for one operation would create an inconsistent security posture.

#### Alternative 2: Disable the fork pull request approval gate

Repositories could simply turn off GitHub's fork PR approval requirement, removing the need for any approval step at all.

This was rejected because the approval gate is a meaningful security control: it prevents untrusted code from fork PRs running in CI with access to repository secrets. Disabling it trades a workflow convenience problem for a real supply-chain risk.

#### Alternative 3: Expose a generic `dispatch-workflow` or `re-run` safe output

A more general "re-run workflow" or "dispatch workflow" primitive could be repurposed to approve runs by triggering a new run. However, GitHub's approval endpoint is semantically distinct from re-running; repurposing a generic action would be misleading and would not work for runs blocked by the approval gate (which require the dedicated `approveWorkflowRun` API endpoint).

### Consequences

#### Positive
- AI agents can unblock fork PR workflow runs without human intervention, enabling fully automated fork PR triage flows.
- The handler accepts only positive workflow run IDs and enforces server-side eligibility checks before approving, so agents cannot accidentally approve completed, non-PR, or unauthorized PR runs.
- `actions: write` is granted only when `approve-workflow-run` is explicitly enabled in the workflow's `safe-outputs` config, preserving the principle of least privilege for workflows that do not need this capability.
- Staged mode (`staged: true`) previews the requested run without GitHub API access or max-count consumption, supporting safe rollout.

#### Negative
- `actions: write` is a broad GitHub permission scope — a compromised or misbehaving agent with this safe output enabled could approve workflow runs from malicious forks.
- The approval check (fetch run → verify conclusion → approve) is not atomic; a time-of-check/time-of-use (TOCTOU) race is theoretically possible if a run's status changes between the GET and the approval POST, though the practical risk is low since the approval endpoint itself validates eligibility server-side.

#### Neutral
- `approve_workflow_run` is added to `THREAT_WARNING_ABORT_TYPES` in the handler manager, placing it in the same threat-detection category as `merge-pull-request` and `close-pull-request`.
- The default `max` is 1, consistent with other high-impact single-shot operations; operators can raise this via the `max` config field if their use case requires approving multiple runs per workflow execution.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
