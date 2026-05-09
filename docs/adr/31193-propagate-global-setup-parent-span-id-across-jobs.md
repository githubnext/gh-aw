# ADR-31193: Propagate a Global Setup Parent Span ID Across Jobs

**Date**: 2026-05-09
**Status**: Draft
**Deciders**: Unknown (PR author + reviewers — please update)

---

## Part 1 — Narrative (Human-Friendly)

### Context

Compiled gh-aw workflows already share a single OTLP `trace-id` across jobs (pre-activation, activation, agent, cache, safe-outputs, unlock, notify) by piping `${{ steps.setup.outputs.trace-id }}` through `needs.<upstream>.outputs.setup-trace-id`. Each per-job `gh-aw.<job>.setup` span, however, was emitted with whatever parent span the local job could resolve from `aw_info.context`, which is not stable across the GitHub Actions job graph. As a result OTLP backends rendered the setup phase as a fan of orphaned siblings rather than a single coherent setup tree, making setup-phase debugging difficult and trace consistency across the activation → agent → downstream chain unreliable. The change must remain compatible with the existing trace-id propagation contract and must not require new infrastructure beyond standard GitHub Actions job outputs. [TODO: verify whether any external OTLP consumer already depends on the previous parent-span behavior.]

### Decision

We will introduce an explicit `parent-span-id` input and `span-id` / `parent-span-id` outputs on `actions/setup`, and the gh-aw compiler will wire `setup-span-id` and `setup-parent-span-id` job outputs through the entire built-in job graph so that every `gh-aw.<job>.setup` span is parented to the same global setup span emitted by the root setup job. Inside `sendJobSetupSpan`, the parent span ID is resolved in a fixed precedence order: explicit option (`options.parentSpanId`) → action input (`INPUT_PARENT_SPAN_ID`) → propagated context (`aw_info.context.otel_parent_span_id`). The primary driver is trace-tree coherence: with one shared parent, OTLP UIs render a single setup subtree across all jobs.

### Alternatives Considered

#### Alternative 1: Status quo — propagate only `setup-trace-id`

Keep the current behavior where each job's setup span derives its parent locally from `aw_info.context.otel_parent_span_id` (or none). This is simple and requires no compiler or action contract changes, but it leaves setup spans orphaned at the trace-tree level even though they share a `trace-id`. Rejected because it fails to deliver the user-visible benefit (a coherent setup hierarchy) that motivated the PR.

#### Alternative 2: Use W3C `traceparent` header propagation end-to-end

Adopt the standard W3C trace context format (`00-<trace-id>-<span-id>-<flags>`) as the single propagated value across jobs instead of separate `trace-id` / `parent-span-id` outputs. This is more aligned with OTEL ecosystem conventions and would let external tooling consume the value directly. Rejected for this PR because it would require parsing/validation across every callsite and would be a larger contract change than the focused setup-tree fix; can be revisited as a follow-up. [TODO: confirm whether a future ADR should track migration to traceparent.]

#### Alternative 3: Emit only one setup span at the root job

Have only pre-activation (or activation) emit a setup span and have downstream jobs skip per-job setup spans entirely. This trivially produces a single setup hierarchy but loses per-job setup timing data, which is one of the main reasons setup spans exist. Rejected because the visibility loss outweighs the simplification.

### Consequences

#### Positive
- OTLP backends render a single coherent setup subtree across activation, agent, cache, safe-outputs, unlock, and notify, improving setup-phase debugging.
- Per-job setup timing remains visible — each job still emits its own `gh-aw.<job>.setup` span, just correctly parented.
- The resolution chain (option → action input → context) gives both compiler-driven propagation and ad-hoc programmatic use a clear contract.

#### Negative
- The `actions/setup` public contract grows: one new input (`parent-span-id`) and two new outputs (`span-id`, `parent-span-id`). This is a permanent surface-area cost for any external consumer of the action.
- Every compiler callsite that builds a downstream setup step now duplicates the `needs.<upstream>.outputs.setup-parent-span-id || needs.<upstream>.outputs.setup-span-id` fallback expression; the helper `setupParentSpanNeedsExpr` partially mitigates this but the pattern is repeated across ~7 builders.
- Wasm golden fixtures and any compiled-workflow snapshot tests must be regenerated to include the new outputs and `parent-span-id` expressions.

#### Neutral
- `setup.sh` now also forwards `INPUT_PARENT_SPAN_ID` to `action_setup_otlp.cjs`, keeping shell and node code paths in sync.
- `sendJobSetupSpan` now returns `parentSpanId` in addition to `traceId` / `spanId` so callers can propagate it onward; pre-existing callers that only destructure `{ traceId, spanId }` continue to work.

---

## Part 2 — Normative Specification (RFC 2119)

> The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this section are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### Setup Action Contract

1. The `actions/setup` action **MUST** expose a `parent-span-id` input accepting a 16-character lowercase hexadecimal OTLP span ID.
2. The `actions/setup` action **MUST** expose `span-id` and `parent-span-id` step outputs containing the IDs used for the emitted `gh-aw.<job>.setup` span.
3. The `parent-span-id` input **MUST** be optional and **MUST** default to the empty string.
4. When the `parent-span-id` input value is not a valid 16-character lowercase hex span ID, the setup action **MUST** treat it as absent rather than fail.

### Parent Span Resolution

1. `sendJobSetupSpan` **MUST** resolve the parent span ID using the following precedence: (a) `options.parentSpanId`, (b) `process.env.INPUT_PARENT_SPAN_ID` (or hyphenated `INPUT_PARENT-SPAN-ID`), (c) `aw_info.context.otel_parent_span_id`.
2. `sendJobSetupSpan` **MUST** return the resolved `parentSpanId` in its result object alongside `traceId` and `spanId`.
3. When no valid parent span ID is resolved, the emitted setup span **MUST** be sent without a `parentSpanId` field (treated as a root span within the shared trace).
4. The resolved `parentSpanId` **MUST** be written to `GITHUB_OUTPUT` as `parent-span-id=<hex>` only when it is a valid span ID.

### Compiler Wiring

1. The compiler **MUST** declare `setup-span-id` and `setup-parent-span-id` job outputs on every built-in job that emits a setup span (pre-activation, activation, agent/main, cache, safe-outputs, unlock, notify, experiments).
2. Downstream setup steps **MUST** receive `parent-span-id` from the upstream job using the expression `${{ needs.<upstream>.outputs.setup-parent-span-id || needs.<upstream>.outputs.setup-span-id }}`.
3. The pre-activation (root) setup step **MUST NOT** be passed a `parent-span-id` value; its setup span is the root of the shared setup subtree.
4. The compiler **SHOULD** route the fallback expression through a single helper (e.g., `setupParentSpanNeedsExpr`) rather than re-spelling it at each callsite.
5. Compiled workflow golden fixtures **MUST** be updated to include the new `setup-span-id` / `setup-parent-span-id` outputs and `parent-span-id` step inputs.

### Conformance

An implementation is considered conformant with this ADR if it satisfies all **MUST** and **MUST NOT** requirements above. Failure to meet any **MUST** or **MUST NOT** requirement constitutes non-conformance.

---

*ADR generated as a Draft by the Design Decision Gate from the PR diff. Review and finalize before changing status from Draft to Accepted.*
