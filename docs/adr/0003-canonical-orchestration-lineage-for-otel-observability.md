# ADR-0003: Canonical Orchestration Lineage for OTEL Observability

**Date**: 2026-05-03
**Status**: Draft
**Deciders**: mnkiefer, Copilot

---

## Part 1 — Narrative (Human-Friendly)

### Context

gh-aw already propagates `aw_context` across multiple workflow transports, including `workflow_call`, `workflow_dispatch`, and `repository_dispatch`. The propagated data carries useful trigger metadata such as item type/number, comment identifiers, deployment state, workflow run conclusion, and OTEL trace continuation fields. However, the current observability model overloads `workflow_call_id` as the main cross-workflow correlation primitive for OTEL episode attributes.

That coupling is too narrow for the repository's actual orchestration model. In practice, lineage spans multiple trigger and handoff styles: reusable workflows, dispatch relays, repository dispatches, and future orchestration flows that may not look like `workflow_call` at all. At the same time, the codebase also has other chaining mechanisms such as temporary IDs, deferred safe-output retries, synthetic updates, `assign_to_agent`, and `create_agent_session`. Those mechanisms are important for observability, but they are not lineage primitives.

To observe automation sessions at scale, OTEL needs a small, stable, transport-agnostic identity model, while the richer execution graph details remain available as bounded summary attributes and span events.

### Decision

We will adopt a canonical orchestration lineage model inside the existing flat `aw_context` envelope. The flat shape is preserved for backward compatibility with current validators and consumers. The canonical lineage fields are:

- `episode_id`
- `hop_id`
- `parent_hop_id`
- `origin_event`
- `root_repo`
- `root_workflow_id`
- `root_run_id`

The existing `workflow_call_id` field will remain during migration as a legacy alias of `hop_id`.

Each workflow invocation is a hop. The first hop in an automation session creates a new `episode_id`; child hops inherit the `episode_id` and set `parent_hop_id` to the caller's `hop_id`. Root metadata and `origin_event` are preserved across the chain. OTEL span enrichment will prefer these canonical lineage fields and continue to emit legacy `workflow_call` attributes as compatibility aliases during migration.

Temporary ID resolution, deferred retries, synthetic updates, `assign_to_agent`, and similar safe-output execution mechanics remain observable, but they are explicitly treated as execution-graph details rather than lineage identifiers. They should be represented through bounded OTEL summary attributes and span events, not as primary correlation keys.

### Alternatives Considered

#### Alternative 1: Keep `workflow_call_id` as the Primary Episode Identifier

This was rejected because `workflow_call_id` encodes one transport mechanism and does not generalize cleanly to `repository_dispatch`, workflow relays, or future orchestration paths. It would keep OTEL correlation coupled to transport-specific naming and force additional exceptions over time.

#### Alternative 2: Introduce a Nested `aw_context.lineage` Object Immediately

This was rejected for the first migration phase because current `aw_context` validation in the repository accepts only flat, primitive-valued objects and explicitly rejects nested objects. A nested shape would require a wider migration of validators, tests, and consumers before the lineage model could land.

#### Alternative 3: Reuse Temporary IDs or Safe-Output Execution State as Episode Correlation

This was rejected because temporary IDs, deferred retries, and synthetic updates solve local dependency resolution inside a single safe-output processing run. They describe execution ordering, not cross-workflow lineage. Treating them as lineage primitives would conflate two different architectural layers and produce unstable OTEL correlation.

### Consequences

#### Positive

- OTEL correlation becomes transport-agnostic and scales across reusable workflows, dispatch relays, and repository dispatches.
- Existing `aw_context` consumers remain compatible because the envelope stays flat and `workflow_call_id` is preserved as an alias.
- Root trigger metadata, current-hop metadata, and execution-graph details become easier to reason about separately.

#### Negative

- The flat schema is an incremental compromise; it does not yet express the lineage model as a nested structured object.
- OTEL enrichment must temporarily emit both canonical lineage fields and legacy `workflow_call` aliases during the migration window.
- Repository-dispatch receivers and other consumers need to read `aw_context` from more than one payload location to preserve lineage end to end.

#### Neutral

- `workflow_call_id` remains present during migration but is no longer the conceptual center of the model.
- Safe-output execution mechanisms such as temporary IDs and `assign_to_agent` remain unchanged in behavior; only their observability role is clarified.

---

## Part 2 — Normative Specification (RFC 2119)

> The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this section are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### Canonical Lineage Schema

1. Implementations **MUST** represent orchestration lineage in `aw_context` using flat primitive fields.
2. Implementations **MUST** populate the following canonical lineage fields when building `aw_context`: `episode_id`, `hop_id`, `parent_hop_id`, `origin_event`, `root_repo`, `root_workflow_id`, and `root_run_id`.
3. Implementations **MUST** keep `workflow_call_id` during migration as a legacy alias of `hop_id`.
4. Implementations **MUST NOT** require nested objects inside `aw_context` for the first migration phase.

### Lineage Generation Rules

1. A root workflow invocation **MUST** create a new `episode_id` and `hop_id`.
2. A child workflow invocation **MUST** inherit `episode_id` from inbound `aw_context` when available.
3. A child workflow invocation **MUST** set `parent_hop_id` to the caller's `hop_id` when available, falling back to the legacy `workflow_call_id` only for backward compatibility.
4. Implementations **MUST** preserve `root_repo`, `root_workflow_id`, `root_run_id`, and `origin_event` from inbound `aw_context` when present.
5. Implementations **MUST** use current workflow metadata to populate the current hop's `repo`, `run_id`, `run_attempt`, `workflow_id`, `event_type`, and `hop_id`.

### Propagation Rules

1. Implementations **MUST** propagate the same canonical `aw_context` envelope across supported cross-workflow transports.
2. Implementations **MUST** support reading inbound `aw_context` from workflow inputs and repository-dispatch `client_payload`.
3. Implementations **MUST NOT** treat `workflow_call` as the only valid transport for lineage propagation.

### OTEL Mapping Rules

1. OTEL span enrichment **MUST** prefer canonical lineage fields over legacy `workflow_call_id` when constructing correlation attributes.
2. OTEL span enrichment **MUST** emit `gh-aw.episode.id` from `episode_id` when available.
3. OTEL span enrichment **MUST** emit current-hop identity from `hop_id` and immediate parent lineage from `parent_hop_id` when available.
4. Implementations **MAY** emit legacy `gh-aw.workflow_call.id` and `gh-aw.workflow_call.parent_id` attributes as compatibility aliases during migration.
5. Implementations **MUST NOT** treat temporary IDs, deferred retries, or safe-output side-effect operations as lineage identifiers.

### Execution-Graph Observability Rules

1. Temporary ID registrations and resolutions **SHOULD** be represented as bounded OTEL summary attributes or span events.
2. Deferred safe-output retries and synthetic updates **SHOULD** be represented as bounded OTEL summary attributes or span events.
3. `assign_to_agent`, `create_agent_session`, and similar safe-output operations **SHOULD** be observable through bounded OTEL summary attributes or span events.
4. Implementations **MUST NOT** serialize full temporary ID maps or unbounded execution-detail payloads directly into OTEL attributes.

### Conformance

An implementation is considered conformant with this ADR if it satisfies all **MUST** and **MUST NOT** requirements above. Failure to meet any **MUST** or **MUST NOT** requirement constitutes non-conformance.

---

*ADR created by [adr-writer agent] and adapted to the repository's flat `aw_context` compatibility requirements.*