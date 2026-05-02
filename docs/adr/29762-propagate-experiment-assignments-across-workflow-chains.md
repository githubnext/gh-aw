# ADR-29762: Propagate Experiment Assignments Across Workflow Chains via aw_context

**Date**: 2026-05-02
**Status**: Draft
**Deciders**: pelikhan, copilot-swe-agent

---

## Part 1 — Narrative (Human-Friendly)

### Context

The Agentic Workflows (AW) framework supports A/B experiments that assign each workflow run to a variant (e.g. `"concise"` vs `"verbose"`) at activation time. A workflow can dispatch or call child workflows to delegate sub-tasks. Before this change, each workflow in such a chain independently picked its own experiment variants using a state-based selection algorithm. Because variant picking is stateful and non-deterministic, a single end-user interaction could experience different experiment variants at different stages of the workflow chain, breaking the fundamental requirement that a single logical request be treated as a single experimental unit.

### Decision

We will embed the parent workflow's current experiment assignments as a JSON string in the `aw_context` object (under the `experiments` field) and forward that `aw_context` to all dispatched and called child workflows. Child workflows will parse the `experiments` field and inherit the parent's variant for any experiment whose name matches a locally declared experiment and whose inherited variant is still valid in the local spec. Local-only experiments (not present in the parent context) continue to be picked normally via the state-based algorithm.

### Alternatives Considered

#### Alternative 1: Independent per-workflow picking (status quo)

Each workflow picks its own variants based on a shared state file. This was the original behavior and requires no cross-workflow coordination. It was rejected because it makes variant consistency across a workflow chain impossible to guarantee — the same user request could activate variant A in the parent but variant B in the child, violating experiment integrity.

#### Alternative 2: Centralized experiment coordination service

A dedicated service could track which variant is active for a given user/run ID and answer queries from all workflows in a chain. This provides strong consistency but requires persistent infrastructure outside GitHub Actions, adds a network dependency on the critical path of every activation, and significantly increases operational complexity. It was rejected as disproportionate to the problem given that `aw_context` is already forwarded between workflows.

#### Alternative 3: Shared state file via artifact pass-through

The parent workflow could upload the experiment state file as an artifact and the child workflow could download and merge it. This avoids double-encoding JSON but requires artifact upload/download steps (slow, adds cost), is error-prone under concurrent fan-out, and does not compose cleanly with `workflow_call` triggers where `aw_context` is the established input channel. It was not chosen because the `aw_context` forwarding mechanism already exists and is the natural vehicle for this data.

### Consequences

#### Positive
- Experiment variant is consistent across all workflows in a dispatch or call chain, preserving the integrity of A/B experiments.
- State counters are not double-incremented — child workflows that inherit a parent variant skip the state update, preventing skewed assignment distributions.
- The mechanism is framework-managed and transparent to workflow authors: no changes to DESIGN.md files or experiment specs are required.
- Both `workflow_dispatch` and `workflow_call` trigger paths are covered symmetrically.

#### Negative
- Experiment assignments are stored as a JSON string inside `aw_context`, which is itself serialized to JSON, creating double-encoded JSON (`experiments` is a string containing `{"feature":"A"}`). This adds parsing complexity and is an unusual pattern that future developers must understand.
- If the local variant list for an experiment changes after the parent ran (schema drift), the inherited variant may no longer be valid. The code silently falls back to independent picking in that case, which is safe but means the inconsistency is masked rather than surfaced.
- The `aw_context` object grows slightly for workflows that use experiments, increasing the size of inputs passed between workflows.

#### Neutral
- The compiler injects `aw_context` as a `workflow_call` input in compiled workflows, mirroring the existing injection for `workflow_dispatch`. This is handled automatically and requires no manual workflow changes.
- Lock files (`.lock.yml`) for any workflow that uses experiments will be regenerated to include the new `GH_AW_EXPERIMENT_CONTEXT` and `GH_AW_EXPERIMENTS_JSON` env var injections.

---

## Part 2 — Normative Specification (RFC 2119)

> The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this section are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### Experiment Assignment Embedding in aw_context

1. When a workflow with experiments configured reaches its safe-outputs step, implementations **MUST** set the `GH_AW_EXPERIMENTS_JSON` environment variable to the JSON string of experiment assignments produced by the activation job's `pick-experiment` step (i.e., `${{ needs.activation.outputs.experiments }}`), or to an empty string if no experiments are configured.
2. The `buildAwContext()` function **MUST** include an `experiments` field whose value is the content of `GH_AW_EXPERIMENTS_JSON`, or an empty string when that variable is absent or empty.
3. The `experiments` field **MUST** be stored as a primitive string (not as a nested object) to preserve the flat-object constraint on `aw_context`.
4. Implementations **MUST NOT** re-pick experiment variants for an experiment name that already has a valid assignment in the parent `aw_context` (i.e., the inherited variant is present in the local spec's variant list).

### Experiment Context Forwarding

1. For `workflow_dispatch` triggers, the caller's `aw_context` (including `experiments`) **MUST** be forwarded to the dispatched workflow via the `aw_context` input, using the existing dispatch path.
2. For `workflow_call` triggers, the caller's `aw_context` **MUST** be exposed as the `call_workflow_aw_context` output of the safe-outputs job, and **MUST** be passed as the `aw_context` input in all compiler-generated `call-*` fan-out jobs.
3. The compiler **MUST** inject an `aw_context` input declaration into the `workflow_call` trigger block of compiled workflows (mirrors the existing `workflow_dispatch` injection), making the input accessible as `${{ inputs.aw_context }}`.
4. The `pick-experiment` step **MUST** receive the raw `aw_context` JSON string via the `GH_AW_EXPERIMENT_CONTEXT` environment variable, populated from `${{ inputs.aw_context || '' }}`.

### Inheritance Logic in pick-experiment

1. Implementations **MUST** parse `GH_AW_EXPERIMENT_CONTEXT` to extract the `experiments` field from the parent `aw_context` before running the local variant selection algorithm.
2. For each locally declared experiment, if a matching assignment exists in the parent context and the inherited variant is a member of the local `variants` list, implementations **MUST** use the parent's variant and **MUST NOT** update the state counter for that experiment.
3. If the inherited variant is not in the local `variants` list, implementations **MUST** fall through to the normal state-based variant selection algorithm for that experiment.
4. Parsing failures (malformed JSON, missing fields, non-string variant values) **MUST** be handled gracefully by returning an empty assignment map, effectively treating the workflow as having no parent context.
5. Implementations **SHOULD** log a warning when a non-string experiment value is encountered in the parent context.

### Reserved Call-Workflow Inputs

1. The `aw_context` input **MUST** be treated as a reserved, framework-managed input in call-workflow jobs. Implementations **MUST NOT** attempt to forward it from the agent payload JSON.
2. Implementations **MUST** maintain a `reservedCallWorkflowInputs` map (or equivalent) that enumerates all framework-managed inputs (`payload`, `aw_context`) and **MUST** skip these when deriving per-input `with:` bindings from the payload.

### Conformance

An implementation is considered conformant with this ADR if it satisfies all **MUST** and **MUST NOT** requirements above. Failure to meet any **MUST** or **MUST NOT** requirement constitutes non-conformance. In particular: (a) any implementation that re-picks a variant for an experiment already assigned in the parent context, (b) any implementation that omits the `experiments` field from `aw_context` when `GH_AW_EXPERIMENTS_JSON` is set, or (c) any implementation that allows `aw_context` to be overridden by agent payload for call-workflow jobs is non-conformant.

---

*This is a DRAFT ADR generated by the [Design Decision Gate](https://github.com/github/gh-aw/actions/runs/25251362027) workflow. The PR author must review, complete, and finalize this document before the PR can merge.*
