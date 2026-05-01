# ADR-29628: Add `--experiment` and `--variant` Filter Flags to `gh aw audit`

**Date**: 2026-05-01
**Status**: Draft
**Deciders**: pelikhan

---

## Part 1 — Narrative (Human-Friendly)

### Context

The `gh aw audit` command processes workflow run artifacts and produces structured reports. As A/B experiment infrastructure matured (see ADR-29534 and ADR-29618), runs began carrying `experiment` artifacts that record which experiment variant was active. Engineers running multi-variant experiments need to compare audit outcomes across specific variants without manually scanning every run report. There was no mechanism to restrict `gh aw audit` output to runs belonging to a particular experiment or variant, making cross-variant comparisons tedious and error-prone at scale.

### Decision

We will add two optional CLI flags — `--experiment <name>` and `--variant <value>` — to `gh aw audit` and propagate them as trailing parameters on `AuditWorkflowRun`. When `--experiment` is specified, runs whose experiment artifact lacks that experiment name are skipped with an informational message on stderr; when `--variant` is also specified, only runs assigned that exact variant pass. We will additionally surface the compact `name=variant` experiment label in the run Overview section so operators can confirm experiment context at a glance without reading the full report. `--variant` without `--experiment` is a user error and is rejected with a clear suggestion message.

### Alternatives Considered

#### Alternative 1: External Post-Processing via Pipe/grep

Users could pipe `gh aw audit` output through `grep` or `jq` to filter by experiment. This requires no changes to the CLI or `AuditWorkflowRun` signature and avoids downloading and processing every run before filtering. It was rejected because: (a) it requires users to know the exact output format and field names, (b) it does not emit the structured skip messages that help users understand filtering behaviour, and (c) it cannot prevent full artifact download for skipped runs in the current architecture — both approaches share that cost, so there is no efficiency advantage.

#### Alternative 2: Dedicated `gh aw experiments` Subcommand

A separate subcommand could aggregate experiment data across multiple runs and provide richer cross-variant comparison. This would cleanly separate experiment analysis from per-run auditing and avoid widening the `AuditWorkflowRun` signature. It was rejected because: (a) the immediate need is simple filtering within existing audit workflows, not a new analytics surface; (b) a dedicated subcommand would duplicate most of the artifact-download and report-render pipeline; and (c) the feature can be factored out later if demand warrants it — the helpers introduced here (`experimentMatchesFilter`, `formatExperimentLabel`) are already isolated in `audit_report_experiments.go`.

### Consequences

#### Positive
- Engineers can target specific experiment variants directly from the CLI, making cross-variant comparisons reproducible and scriptable.
- Experiment assignments are surfaced in the Overview section, giving operators immediate visibility into run context without reading the full report body.
- The filtering logic is concentrated in small, independently-testable helper functions that can be reused in future experiment analysis features.

#### Negative
- `AuditWorkflowRun` signature grows by two trailing `string` parameters (`experimentFilter`, `variantFilter`), increasing argument count and requiring every call site to be updated. This makes the function harder to extend in the future without introducing a struct-based options pattern.
- Filtering occurs **after** artifact download and initial processing, not before; runs that will ultimately be skipped still incur the full download cost.

#### Neutral
- All existing call sites of `AuditWorkflowRun` must pass two empty strings as the new trailing parameters, touching several test files.
- The `OverviewData` and `OverviewDisplay` structs gain a new `Experiment string` field; existing JSON consumers that do not expect this field are unaffected because the field is tagged `omitempty`.

---

## Part 2 — Normative Specification (RFC 2119)

> The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this section are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### CLI Flag Semantics

1. The `gh aw audit` command **MUST** accept an `--experiment <name>` flag that filters runs to those with an assignment for the named experiment in their experiment artifact.
2. The `gh aw audit` command **MUST** accept a `--variant <value>` flag that, when combined with `--experiment`, further restricts results to runs assigned that exact variant value.
3. Implementations **MUST NOT** allow `--variant` to be used without `--experiment`; such invocations **MUST** return a non-zero exit code with a human-readable error message that includes a suggestion to add `--experiment`.
4. When a run is skipped because it does not satisfy the active experiment/variant filter, an informational message **MUST** be emitted to stderr identifying the run ID, the experiment name, and (if applicable) the required variant.

### Experiment Label in Overview

1. The run Overview section **MUST** include an `Experiment` field when the run's experiment artifact contains one or more assignments.
2. The experiment label **MUST** be formatted as a comma-separated, alphabetically sorted list of `name=variant` pairs (e.g., `caveman=yes, style=concise`).
3. The `Experiment` field **MUST** be omitted from both console and JSON output when no experiment assignments are present (`omitempty` semantics).

### `AuditWorkflowRun` Filtering Contract

1. Implementations **MUST** apply the experiment/variant filter before calling any report-rendering code; a skipped run **MUST** return `nil` (not an error).
2. Implementations **MUST** apply the filter in both the cached-summary path and the fresh-processing path to ensure consistent behaviour regardless of whether a run summary cache exists.
3. Implementations **SHOULD** extract experiment data at most once per `AuditWorkflowRun` invocation when a filter is active; redundant reads of the experiment artifact **SHOULD** be avoided.
4. When neither `experimentFilter` nor `variantFilter` is set (both empty strings), implementations **MUST NOT** read or parse the experiment artifact solely for filtering purposes; the Overview population path **MAY** still read it for display.

### Conformance

An implementation is considered conformant with this ADR if it satisfies all **MUST** and **MUST NOT** requirements above. Failure to meet any **MUST** or **MUST NOT** requirement constitutes non-conformance.

---

*This is a DRAFT ADR generated by the [Design Decision Gate](https://github.com/github/gh-aw/actions/runs/25234091491) workflow. The PR author must review, complete, and finalize this document before the PR can merge.*
