# ADR-29269: Label-Triggered Jobs for Agentic Maintenance Workflow

**Date**: 2026-04-30
**Status**: Draft
**Deciders**: pelikhan, copilot-swe-agent

---

## Part 1 — Narrative (Human-Friendly)

### Context

The agentic maintenance workflow previously lacked a way for maintainers to operationally control agentic workflows (disable them or replay their safe outputs) directly from the GitHub Issues UI. The only existing control paths required either command-line access or navigating the GitHub Actions `workflow_dispatch` interface — both of which are cumbersome for routine maintenance tasks. The system already tracks agentic workflow metadata (run URLs, workflow IDs) inside XML comment markers in issue bodies, making issues the natural locus for maintainer-initiated control actions.

### Decision

We will extend the agentic maintenance workflow with two label-triggered jobs: `label_disable_agentic_workflow` and `label_apply_safe_outputs`. When a maintainer with admin or maintainer permissions applies the `agentic-workflows:disable` or `agentic-workflows:apply-safe-outputs` label to an issue created by an agentic workflow, the corresponding job fires, reads the workflow metadata from the issue body's XML comment markers, and performs the requested operation via the GitHub REST API. Both jobs are controlled by a single `label_triggers` boolean flag in `aw.json` and default to enabled. We chose the GitHub Issues label mechanism because it is accessible from the standard GitHub UI, is auditable, and requires no additional tooling for the triggering party.

### Alternatives Considered

#### Alternative 1: `workflow_dispatch` manual operation inputs

The workflow already supports a `workflow_dispatch` trigger with an `operation` input. Extending this to cover disable and replay operations was considered. It was rejected because it requires the operator to know the target workflow ID or run URL in advance, navigate to the Actions tab, and fill in a form — all steps that move away from the natural issue context where the metadata already lives. It also does not provide the same discoverability as a label applied to the relevant issue.

#### Alternative 2: Slash commands in issue comments (`/disable-workflow`, `/apply-safe-outputs`)

Slash commands parsed from issue comments are a common GitHub automation pattern. They were not chosen because the repository's existing automation stack uses labels as the primary trigger mechanism for maintainer actions, and adding a comment-parsing layer would require a separate event handler, introduce ambiguity around comment timing and re-entrancy, and increase the attack surface for command injection via crafted comment bodies.

### Consequences

#### Positive
- Maintainers can disable a misbehaving agentic workflow or replay its safe outputs directly from the issue UI without leaving GitHub.
- Labels are self-documenting and visible in issue timelines, making operational actions auditable by default.
- Shared helpers (`label_trigger_helpers.cjs`) encapsulate permission checking and label lifecycle, making future label-triggered jobs easier to add safely.
- The `label_triggers: false` opt-out in `aw.json` gives repo owners a single switch to remove both jobs from the generated workflow if the pattern is not wanted.

#### Negative
- Adding `issues: [labeled]` to the workflow trigger causes the workflow to fire on every issue label event, even for issues unrelated to agentic workflows; the jobs' `if:` conditions short-circuit quickly, but there is a small cost in workflow invocation overhead.
- Each job requires a permission gate (`check_team_member.cjs`) as the first step; if that helper has a bug or the team membership API is unavailable, the jobs fail rather than degrading gracefully.
- Workflow IDs and run URLs embedded in issue bodies are relied upon as the source of truth; if an issue body is edited to remove or corrupt the XML comment markers, the jobs will fail silently with a "missing marker" comment rather than finding the data another way.

#### Neutral
- The `agentic-workflows:disable` and `agentic-workflows:apply-safe-outputs` labels are created lazily on first job run rather than during general label setup (`create_labels.cjs`), which keeps label creation scoped to the feature but means the labels may not exist in a repository until the jobs run for the first time.
- The `label_disable_agentic_workflow` job uses only `contents: read` plus `actions: write`; the `label_apply_safe_outputs` job requires broader write permissions (`contents: write`, `pull-requests: write`, etc.) matching those needed by the safe-outputs replay logic.

---

## Part 2 — Normative Specification (RFC 2119)

> The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this section are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### Label-Triggered Job Activation

1. Label-triggered jobs **MUST** fire only on `issues: [labeled]` events, never on pull request events.
2. Each label-triggered job **MUST** verify that the actor holds admin or maintainer permissions before performing any mutation, using `check_team_member.cjs` or an equivalent gate.
3. Label-triggered jobs **MUST NOT** fire when `label_triggers` is set to `false` in the `maintenance` object of `aw.json`; the `issues: [labeled]` trigger and both jobs **MUST** be omitted from the generated workflow in that case.
4. Each job **MUST** remove the triggering label from the issue after a successful operation to prevent the job from re-running if the label is re-applied inadvertently.
5. Each job **SHOULD** post a comment to the issue describing the outcome (success, failure, or missing marker) so the triggering actor has immediate feedback.

### Workflow Metadata Extraction

1. Implementations **MUST** extract workflow identifiers exclusively from XML comment markers in the issue body (`<!-- gh-aw-workflow-id: ... -->`, `<!-- gh-aw-agentic-workflow: ... -->`, or `<!-- gh-aw-workflow-call-id: ... -->`); they **MUST NOT** parse free-form issue body text.
2. Extracted workflow IDs **MUST** be validated against `isValidWorkflowId()` (alphanumeric characters, hyphens, underscores, and dots; maximum 100 characters; no `..` path segments) before use.
3. Implementations **MUST NOT** spread `process.env` into subprocess calls; only explicitly required environment variables **MAY** be passed to subprocesses.

### Label Lifecycle

1. The `agentic-workflows:disable` and `agentic-workflows:apply-safe-outputs` labels **MUST** be created idempotently by their respective jobs on first run using `ensureLabelExists()`; they **MUST NOT** be added to the general `create_labels` label set.
2. Label creation **MUST** be scoped exclusively to the job that uses the label; no cross-job label dependencies are permitted.

### Conformance

An implementation is considered conformant with this ADR if it satisfies all **MUST** and **MUST NOT** requirements above. Specifically: label-triggered jobs fire only on issue events; actor permissions are verified before any mutation; label-triggered jobs are omitted when `label_triggers: false`; workflow IDs are validated before use; subprocesses receive only explicitly scoped environment variables; and per-operation labels are created by their owning job rather than centrally. Failure to meet any **MUST** or **MUST NOT** requirement constitutes non-conformance.

---

*This is a DRAFT ADR generated by the [Design Decision Gate](https://github.com/github/gh-aw/actions/runs/25181104179) workflow. The PR author must review, complete, and finalize this document before the PR can merge.*
