# ADR-29963: Introduce Episode Lineage Context for Multi-Workflow OTEL Tracing

**Date**: 2026-05-03
**Status**: Draft
**Deciders**: mnkiefer

---

## Part 1 — Narrative (Human-Friendly)

### Context

Agentic workflows in this system can dispatch child workflows via `workflow_call` and `workflow_dispatch` triggers. Until this change, each workflow run was an isolated observability unit: OTEL traces started fresh at each hop, making it impossible to correlate parent and child executions into a single trace. The existing `aw_context` object carried per-run identity (`run_id`, `workflow_id`, `workflow_call_id`) but had no concept of "this run is a continuation of a previous automation session." As the system gained multi-hop workflows (a dispatcher calling a worker, a worker calling another workflow), the lack of shared lineage meant Grafana/OTLP dashboards showed disconnected fragments rather than end-to-end traces.

### Decision

We will extend `aw_context` with a canonical episode lineage model — `episode_id`, `hop_id`, `parent_hop_id`, `origin_event`, `root_repo`, `root_workflow_id`, `root_run_id` — and propagate it automatically across every workflow boundary. The `episode_id` is minted at the first hop and frozen thereafter; each subsequent hop records its own `hop_id` and the `parent_hop_id` of its caller. The setup action receives the current workflow name and ref as `GH_AW_SETUP_WORKFLOW_NAME`/`GH_AW_CURRENT_WORKFLOW_REF` env vars so OTEL spans can emit `gh-aw.episode.*` and `gh-aw.hop.*` attributes that correctly link parent and child spans. The `workflow_call` trigger now receives an `aw_context` input (in addition to `workflow_dispatch`) so reusable workflow callees can inherit lineage without code changes.

### Alternatives Considered

#### Alternative 1: Use GitHub's Native Run Chaining

GitHub provides `github.run_id` and `github.workflow` in every job context. We considered deriving lineage purely from these built-in values without a custom propagation mechanism. This was rejected because GitHub reuses a single `run_id` for all jobs in a `workflow_call` chain, making caller and callee indistinguishable without the workflow ref suffix; moreover, `repository_dispatch` starts a completely new run, severing any native chain entirely.

#### Alternative 2: Propagate Episode Context Only Through OTEL Baggage

We considered using OpenTelemetry Baggage (W3C Baggage header) to propagate the `episode_id` and `hop_id` through the OTLP endpoint, keeping the GitHub event payload clean. This was rejected because GitHub Actions has no mechanism to inject HTTP headers into workflow trigger payloads; the `aw_context` JSON object passed as a workflow input is the only available out-of-band channel between dispatcher and callee.

#### Alternative 3: Emit Separate Root Spans and Link Them Post-Hoc

We considered emitting one OTEL trace per workflow run and linking them using OTLP `links` (span links referencing the parent run's trace ID). This would require a post-processing step or a side-channel registry to look up parent trace IDs at trace-collection time. The approach was rejected because it introduces infrastructure complexity and delays: the linkage would only be visible after a collection pass, whereas the episode model makes parent-child relationships visible immediately at emit time using standard parent-span-id attribution.

### Consequences

#### Positive
- Multi-hop workflow runs appear as a single connected trace in OTLP backends; parent/child span relationships are correct without post-processing.
- The `episode_id` is stable across retries and re-runs of the same automation session, enabling idempotency checks and replay analysis.
- `workflow_call` triggers gain `aw_context` propagation automatically via the compiler, eliminating per-workflow boilerplate.
- Legacy `workflow_call_id` is kept as an alias of `hop_id`, so consumers of the old field continue to work without migration.

#### Negative
- Every setup step in every compiled lock file gains two new env vars (`GH_AW_SETUP_WORKFLOW_NAME`, `GH_AW_CURRENT_WORKFLOW_REF`), bloating compiled output and requiring mass-regeneration of all `.lock.yml` files.
- `aw_context` payload size grows: each forwarded invocation now carries seven additional fields that must be serialized into workflow inputs, which are subject to GitHub's 64 KB input size limit.
- The lineage chain relies on callers correctly forwarding `aw_context`; a caller that strips or ignores the input silently severs the episode, producing a partial trace with no error or warning.

#### Neutral
- The `workflow_call` trigger injection is compiler-driven; workflow authors do not need to manually declare the `aw_context` input.
- `buildWorkflowCallId` / `buildCurrentWorkflowCallId` are now exported from their respective modules, making them available for reuse in other telemetry consumers.

---

## Part 2 — Normative Specification (RFC 2119)

> The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this section are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### Episode Lineage Fields

1. Every `aw_context` object **MUST** contain the fields `episode_id`, `hop_id`, `parent_hop_id`, `origin_event`, `root_repo`, `root_workflow_id`, and `root_run_id`.
2. `episode_id` **MUST** be set to the `hop_id` of the first workflow in the chain and **MUST NOT** be overwritten by child hops.
3. `hop_id` **MUST** uniquely identify the current workflow invocation using the format `{run_id}-{run_attempt}:{workflow_ref}`, or `{run_id}-{run_attempt}` when the workflow ref is unavailable.
4. `parent_hop_id` **MUST** be set to the caller's `hop_id` when the workflow was invoked by another agentic hop, and **MUST** be an empty string for the root hop.
5. `workflow_call_id` **MUST** be kept as a legacy alias equal to the current `hop_id` for backwards compatibility.

### Context Propagation

1. The `call_workflow` safe-output handler **MUST** inject a serialized `aw_context` JSON string into every outbound workflow payload unless the caller has already supplied an `aw_context` key.
2. The workflow compiler **MUST** inject an `aw_context` input declaration into both `workflow_dispatch` and `workflow_call` trigger blocks for every compiled lock file.
3. The `aw_context` input **MUST NOT** be injected if it is already present in the trigger's inputs block (idempotency).
4. Setup steps **MUST** receive `GH_AW_SETUP_WORKFLOW_NAME` and `GH_AW_CURRENT_WORKFLOW_REF` as environment variables so the setup action can record the current workflow identity in OTEL spans.

### OTEL Span Attributes

1. OTEL spans emitted by the setup action **MUST** include `gh-aw.episode.id` set to the inherited `episode_id`.
2. OTEL spans **MUST** include `gh-aw.hop.id` set to the current `hop_id` and `gh-aw.hop.parent_id` set to `parent_hop_id`.
3. When a parent span ID is available, child hops **SHOULD** set the OTEL `parentSpanId` field to establish the standard parent-child span relationship.
4. The `gh-aw.workflow_call.id` and `gh-aw.workflow_call.parent_id` attributes **MUST** be emitted as legacy aliases of `gh-aw.hop.id` and `gh-aw.hop.parent_id` respectively.

### Conformance

An implementation is considered conformant with this ADR if it satisfies all **MUST** and **MUST NOT** requirements above. Failure to meet any **MUST** or **MUST NOT** requirement constitutes non-conformance.

---

*This is a DRAFT ADR generated by the [Design Decision Gate](https://github.com/github/gh-aw/actions/runs/25285401406) workflow. The PR author must review, complete, and finalize this document before the PR can merge.*
