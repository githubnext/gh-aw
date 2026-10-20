# ADR-70653: Use the Unified Session as the Primary Source for Audit Model-Resolution Findings

**Date**: 2026-10-20
**Status**: Draft
**Deciders**: SivaKesava1 (PR author / actor); maintainer review pending

---

### Context

`gh aw audit` only diagnosed model-availability startup failures by matching a message in Pi's `agent-stdio.log`. Other harnesses (for example Copilot and Copilot SDK sub-agent runs 39499460077, 39499464895 and 39508150750) already write fail-closed `model.resolution` records to the unified session (`usage/aw_session.jsonl`) under the T-UAS-074 contract, but audit ignored them, so these failures went undiagnosed. A failed `model.resolution` record does not say whether the failing role was a declared sub-agent or an optional repository agent, so a record alone cannot prove that it caused the run to fail. Routed runs also produce router or classifier traffic and wrapper `state: "started"` markers that can look like agent activity without the native agent ever running.

### Decision

We will read failed `model.resolution` events from the unified session as the primary evidence for model-resolution findings, and keep the Pi `agent-stdio.log` match only as a fallback when the session has no failed resolution events. Resolution records are accepted only with the recorded provenance (`model_resolution` / `agent` / `agent/model-resolution.json`) and fields that satisfy the T-UAS-074 contract. Session files must be regular files (no symlinks) with a `session.format` v1 header and are read through the existing bounded line reader. A failure is attributed as the startup cause only when three conditions hold: the run failed, `agent.execution` exited non-zero, and there is no native agent activity (`session.start`, `assistant.*`, `tool.*` from component `agent`). Otherwise audit emits a single low-severity advisory (`model_resolution_failed`). Failures are grouped by engine, provider, model and cause, and each group is classified only against the same run's `awf-reflect.json`. A run therefore gets at most one `model_unavailable` (high) finding and at most one `model_availability_unknown` finding. Recorded causes other than `model_unavailable` are reported as recorded and never confirmed as unavailability.

### Alternatives Considered

#### Alternative 1: Extend log-message matching to each harness

Add per-engine log patterns, like the existing Pi `agent-stdio.log` match, for Copilot and Copilot SDK output. This would follow the existing pattern, but it ties audit to free-form log text that varies by harness and version. It also ignores the structured, contract-validated records the harnesses already write. It was not chosen because the unified session is the cross-engine source of truth for this data.

#### Alternative 2: Report every failed resolution record as a high-severity failure cause

Treat any failed `model.resolution` record as the reason the run failed, without checking startup evidence. This is simpler, but it misattributes failures. An optional repository agent can fail resolution while the main agent continues and the run succeeds, as in the run 39508154330 negative control. It was not chosen because the record cannot tell sub-agents and repository agents apart, and the producer was deliberately left unchanged.

#### Alternative 3: Change the producer to record the role kind (sub-agent vs repository agent)

Add a field to `model.resolution` so audit can attribute failures directly. This would make attribution more precise. It was not chosen in this PR because it changes the T-UAS-074 producer contract and is outside the `pkg/cli` scope. Historical runs would also still lack the field. [TODO: verify whether this is planned as a follow-up]

### Consequences

#### Positive
- Audit diagnoses model-resolution startup failures for all harnesses that write the unified session, not only Pi.
- Attribution uses a conservative three-condition rule and reports a low-severity advisory in every other case, which avoids false "model unavailable" root causes on successful or continuing runs.
- Classification uses only the same run's reflection inventory, and recorded causes such as `catalog_unavailable` are preserved, so audit does not invent unavailability.
- Strict provenance, schema and file-type validation (no symlinks, bounded reads, v1 header) limits trust in artifact contents.

#### Negative
- Audit now depends on the T-UAS-074 record schema and the provenance constants. Producer changes must be coordinated or records will be silently ignored.
- When several roles fail, audit cannot identify which role stopped startup. It can only say so in the finding text.
- There are two detection paths (session-first, legacy Pi log fallback) to maintain until the legacy path can be removed.
- A new finding code (`model_resolution_failed`) adds to the audit surface. The reference docs (`audit.md`, debug playbook) still describe these findings as Pi-only and must be updated separately.

#### Neutral
- `checkModelAvailability` was extracted from the Pi path and is now shared by both paths, with its messages unchanged.
- Sanitized replay fixtures from real runs live under `pkg/cli/testdata/model_resolution_startup/`. Derived variants are labelled as derived in the test comments.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
