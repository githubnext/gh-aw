# ADR-29985: Experiment Per-Run State, OTEL Integration, and Schema Extensions

**Date**: 2026-05-03
**Status**: Draft
**Deciders**: pelikhan, copilot-swe-agent

---

## Part 1 — Narrative (Human-Friendly)

### Context

ADR-29534 introduced the `experiments:` frontmatter field with a cumulative-count-based `state.json`, and ADR-29618 extended the schema with weighted selection and date gating. After these changes, three gaps remained: (1) the cumulative-count state made it impossible to know with certainty which variant was selected on any specific run — the audit reporter inferred this with a fragile "max count wins" heuristic that breaks on ties or on the first invocation; (2) experiment assignments were invisible to distributed-tracing backends such as Honeycomb or Grafana because no OTEL context was emitted; (3) the schema had no fields to declare the intended statistical test, attach governance labels, or route significance alerts.

### Decision

We will extend the experiment infrastructure in three coordinated directions. First, `pick_experiment.cjs` will append a per-run record `{ run_id, timestamp, assignments }` to a `runs` array inside `state.json` on every invocation; the Go audit reporter will read the last record directly when present, falling back to the existing heuristic only for legacy state files that pre-date this change. Second, after variant selection `pick_experiment.cjs` will export experiment assignments into `OTEL_RESOURCE_ATTRIBUTES` as `experiment.<name>=<variant>` key-value pairs, appending to any pre-existing value so other OTEL instrumentation is not clobbered. Third, the `experiments:` object form (introduced in ADR-29618) will gain three optional fields: `analysis_type` (enum of statistical test names), `tags` (free-form string array), and `notify` (object with `discussion` and `issue` integer fields), all propagated through the JSON schema, Go `ExperimentConfig` struct, and `GH_AW_EXPERIMENT_SPEC` env var.

### Alternatives Considered

#### Alternative 1: Separate Artifact for Per-Run Records

Store per-run assignment records in a separate GitHub Actions artifact or log file rather than inside `state.json`. This would avoid growing `state.json` unboundedly and keep the count-only file simple. It was rejected because `state.json` is already the canonical persistence file for the experiment runtime; introducing a second state artifact creates two sources of truth that must be kept in sync and complicates the audit reporter's discovery logic (which already handles `state.json` path resolution).

#### Alternative 2: GitHub Actions Step Outputs Instead of OTEL Attributes

Emit experiment assignments as GitHub Actions step outputs (`core.setOutput`) for downstream consumption instead of as `OTEL_RESOURCE_ATTRIBUTES`. This is simpler and does not require any OTEL infrastructure. It was rejected because step outputs are only consumable by subsequent steps that explicitly reference them; `OTEL_RESOURCE_ATTRIBUTES` propagates automatically to every span and log emitted anywhere in the job without requiring callers to opt in, making it a zero-friction observability integration.

#### Alternative 3: New Top-Level Schema Section for Analysis Metadata

Add a separate `experiments_analysis:` top-level key for `analysis_type`, `tags`, and `notify` rather than extending the per-experiment object form. This avoids increasing the complexity of the `oneOf` branch already required by ADR-29618. It was rejected because separating variant lists from their analysis metadata across two top-level keys harms co-location and readability — the same reason the parallel `experiments_config:` key was rejected in ADR-29618.

### Consequences

#### Positive
- Per-run records make every variant assignment definitively traceable to a specific run ID and timestamp, eliminating the fragile max-count heuristic in the audit reporter.
- OTEL resource attributes enable experiment cohort filtering in any distributed-tracing backend (Honeycomb, Grafana, Jaeger) without workflow-level wiring.
- The `analysis_type` field enables downstream reporting tooling to select the appropriate statistical test automatically rather than requiring manual specification outside the workflow file.
- The `notify` field co-locates significance-alert routing with the experiment definition, reducing the risk of orphaned experiments with no notification path.

#### Negative
- `state.json` grows unboundedly because there is no pruning mechanism for the `runs` array; long-lived experiments accumulate one record per CI run, which may reach hundreds of kilobytes over months.
- Legacy state files (no `runs` field) require special-case handling in both the JS runtime (`loadState`) and the Go audit reporter (`extractExperimentData`), adding a permanent compatibility branch.
- Appending to `OTEL_RESOURCE_ATTRIBUTES` assumes the environment variable is not consumed destructively by other steps between experiment selection and job end; this assumption is not enforced.

#### Neutral
- The `extractIntField` Go helper extracted in the compiler removes duplicated numeric-coercion switch statements for `issue` and `min_samples`; it is not a semantic change but reduces maintenance surface.
- The step summary table is simplified from four columns (experiment, selected variant, all variants, cumulative counts) to three (experiment, variant, `current/total`); consumers of the step summary output format will observe this change.
- `ExperimentRunRecord` and `ExperimentNotify` are new exported Go types in `pkg/workflow/frontmatter_types.go` and `pkg/cli/audit_report_experiments.go` respectively.

---

## Part 2 — Normative Specification (RFC 2119)

> The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this section are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### Per-Run State in `state.json`

1. When at least one experiment is assigned, `pick_experiment.cjs` **MUST** append a record to the `runs` array in `state.json` before persisting state. Each record **MUST** contain three fields: `run_id` (string, the value of `GITHUB_RUN_ID` or `""` when absent), `timestamp` (ISO-8601 UTC string), and `assignments` (object mapping experiment name to selected variant string).
2. `loadState` **MUST** initialise `runs` to an empty array when the field is absent in an existing state file, preserving backward compatibility with legacy state files.
3. When no experiments are assigned, `pick_experiment.cjs` **MUST NOT** append a run record or write `state.json`.

### Audit Reporter Run-Record Lookup

4. When `state.runs` is non-empty and the last record's `assignments` map is non-empty, the Go audit reporter **MUST** use that record's assignments directly as the current-run experiment data, bypassing the cumulative-count heuristic.
5. When `state.runs` is empty, absent, or the last record's `assignments` map is empty, the audit reporter **MUST** fall back to the existing max-count heuristic defined in ADR-29534.

### OTEL Resource Attributes

6. After variant selection, when at least one experiment is assigned, `pick_experiment.cjs` **MUST** call `core.exportVariable("OTEL_RESOURCE_ATTRIBUTES", …)` with a value of the form `experiment.<name>=<variant>` (comma-separated when multiple experiments are active).
7. When `OTEL_RESOURCE_ATTRIBUTES` is already set in the environment, the implementation **MUST** append the experiment key-value pairs to the existing value with a comma separator, not overwrite it.
8. When no experiments are assigned, `pick_experiment.cjs` **MUST NOT** modify `OTEL_RESOURCE_ATTRIBUTES`.

### Schema Extension: `analysis_type`, `tags`, and `notify`

9. The `analysis_type` field, when present, **MUST** be one of the string literals: `t_test`, `mann_whitney`, `proportion_test`, `bayesian_ab`; any other value **MUST** be rejected by JSON Schema validation.
10. The `tags` field, when present, **MUST** be an array of strings; implementations **MAY** use it for dashboard filtering and **MUST NOT** treat it as affecting variant selection.
11. The `notify` field, when present, **MUST** be an object; if it contains a `discussion` or `issue` key, each **MUST** be a positive integer (minimum 1). The `notify` object **MUST NOT** contain properties other than `discussion` and `issue`.
12. All three new fields (`analysis_type`, `tags`, `notify`) are **OPTIONAL**; their absence **MUST NOT** affect variant selection or state persistence.
13. The compiler **MUST** propagate all three fields through `ExperimentConfig` and into the `GH_AW_EXPERIMENT_SPEC` JSON blob so the JS runtime receives them.

### Conformance

An implementation is considered conformant with this ADR if it satisfies all **MUST** and **MUST NOT** requirements above. Failure to meet any **MUST** or **MUST NOT** requirement constitutes non-conformance. This ADR extends ADR-29618 and ADR-29534; in case of conflict, this ADR takes precedence for the `runs` state field, OTEL export, and the `analysis_type`/`tags`/`notify` schema fields.

---

*This is a DRAFT ADR generated by the [Design Decision Gate](https://github.com/github/gh-aw/actions/runs/25287740913) workflow. The PR author must review, complete, and finalize this document before the PR can merge.*
