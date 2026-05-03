# ADR-29804: Surface missing_tool and missing_data Signals as Agent Failures

**Date**: 2026-05-02
**Status**: Draft
**Deciders**: pelikhan

---

## Part 1 — Narrative (Human-Friendly)

### Context

The `gh-aw` harness emits structured safe-output signals including `missing_tool` and `missing_data` to communicate when an agent cannot complete its task due to unavailable tools or missing data. Historically these signals created separate GitHub issues, while the primary agent failure issue comment contained only execution errors. This fragmented failure context across multiple issues, making it harder for operators to diagnose why an agent run failed — they had to correlate the failure issue with separate missing-tool or missing-data issues. The harness already had a precedent (`report_incomplete`) for escalating soft signals into the agent failure code path, and `missing_tool`/`missing_data` were natural candidates for the same treatment.

### Decision

We will treat `missing_tool` and `missing_data` safe-output signals as agent failures by default, surfacing their context in the same failure issue comment as other failure indicators rather than routing them to separate issues. The default for `create-issue` on these signal types changes from `true` to `false`. A `report-as-failure` feature flag (default: `true`) on each signal type allows workflows to opt out and restore the previous behavior per signal. The `report_incomplete` signal is not affected by this change.

### Alternatives Considered

#### Alternative 1: Keep Separate Issues for missing_tool and missing_data

The original behavior creates a dedicated issue for each `missing_tool` or `missing_data` signal, independent of the agent failure issue. This was rejected because it scatters failure context: an operator diagnosing a run must look in the failure issue *and* potentially multiple separate signal issues, increasing the cognitive load and time to resolution. The unified failure comment approach better matches how operators already triage issues.

#### Alternative 2: Remove Issue Creation Entirely — Footer-Only Display

A more aggressive option would suppress separate issues entirely and only render missing-tool/missing-data context in the run summary footer, with no failure escalation. This was not chosen because it reduces observability: signals that indicate the agent cannot complete work should activate failure handling so that they appear in the central failure issue that oncall and workflow owners monitor.

#### Alternative 3: Always Escalate Without a Feature Flag

The new behavior could be applied unconditionally, with no opt-out. This was rejected to preserve backwards compatibility for workflows that depend on separate issue creation for `missing_tool` or `missing_data`. The `report-as-failure` flag allows a gradual rollout and gives operators a clear escape hatch.

### Consequences

#### Positive
- Failure context is consolidated: a single agent failure issue now contains `missing_tool` and `missing_data` context alongside other failure reasons, reducing the time-to-triage for operators.
- Consistent failure semantics: signals that indicate the agent could not complete its work now uniformly activate failure handling, matching the behavior of `report_incomplete` and `cache_memory_miss`.
- Feature flag provides safe rollout: existing workflows can opt out via `report-as-failure: false` without code changes.

#### Negative
- Breaking default change: existing workflows that relied on separate `missing_tool` or `missing_data` issues will no longer create those issues by default. Operators who triaged those issues must update their workflows or set `report-as-failure: false`.
- Increased verbosity in failure issues: failure issue comments may now contain additional sections for missing-tool and missing-data context, potentially making the comment longer when both signals are present.

#### Neutral
- Lock files for affected workflows (e.g., `hourly-ci-cleaner`, `poem-bot`, `tidy`) are regenerated to pick up the new `GH_AW_MISSING_TOOL_CREATE_ISSUE: "false"` default.
- The `report_incomplete` signal is explicitly excluded from the `report-as-failure` field to preserve its existing always-escalate semantics.

---

## Part 2 — Normative Specification (RFC 2119)

> The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this section are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### Signal Routing Defaults

1. Implementations **MUST** default `create-issue` to `false` for `missing-tool` and `missing-data` signal types.
2. Implementations **MUST** default `create-issue` to `true` for `report-incomplete` (unchanged).
3. Implementations **MUST** default `report-as-failure` to `true` for `missing-tool` and `missing-data` signal types.
4. Implementations **MUST NOT** expose a `report-as-failure` field on `report-incomplete` configuration; that signal always activates failure handling.

### Failure Escalation

1. When `report-as-failure` is `true` (or unset) for `missing-tool`, implementations **MUST** treat the presence of any `missing_tool` item in agent output as a failure condition, activating the same failure handling code path used for `report_incomplete` and `hasCacheMissMisconfiguration`.
2. When `report-as-failure` is `true` (or unset) for `missing-data`, implementations **MUST** treat the presence of any `missing_data` item in agent output as a failure condition.
3. When `report-as-failure` is `false` for a given signal type, implementations **MUST NOT** activate failure handling based solely on that signal type.
4. Implementations **MUST** include `missing_tool_context` in the agent failure issue and failure comment templates when `missing_tool` items are present.

### Feature Flag Propagation

1. Implementations **MUST** pass `GH_AW_MISSING_TOOL_REPORT_AS_FAILURE` and `GH_AW_MISSING_DATA_REPORT_AS_FAILURE` environment variables to the conclusion job so that the flag is honoured at runtime even when the workflow does not explicitly configure `missing-tool:` or `missing-data:` sections.
2. When neither `missing-tool` nor `missing-data` is explicitly configured in `safe-outputs`, implementations **MUST** default both env vars to `"true"`.
3. Implementations **SHOULD** support GitHub Actions expression syntax (e.g., `${{ inputs.report-as-failure }}`) as the value of `report-as-failure` to enable dynamic per-run opt-out.

### Conformance

An implementation is considered conformant with this ADR if it satisfies all **MUST** and **MUST NOT** requirements above. Failure to meet any **MUST** or **MUST NOT** requirement constitutes non-conformance.

---

*This is a DRAFT ADR generated by the [Design Decision Gate](https://github.com/github/gh-aw/actions/runs/25260257832) workflow. The PR author must review, complete, and finalize this document before the PR can merge.*
