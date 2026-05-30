# ADR-29618: Rich Experiment Metadata Schema Extension with Weighted Selection and Date Gating

**Date**: 2026-05-01
**Status**: Draft
**Deciders**: pelikhan, copilot-swe-agent

---

## Part 1 — Narrative (Human-Friendly)

### Context

ADR-29534 introduced the `experiments:` frontmatter field with a bare-array form (`caveman: [yes, no]`) and a least-used round-robin selection algorithm. In practice, teams running A/B experiments needed capabilities the bare-array form could not express: non-uniform probability splits (e.g., 70/30 for a high-risk variant), automatic deactivation after an end date, machine-readable metadata (description, linked issue, metric name) for governance tooling, and backward compatibility with the many existing workflows already using the bare-array form. Extending the schema to support these requirements while keeping the existing syntax working required an explicit decision about how to version the schema and how the selection algorithm should adapt to the richer spec.

### Decision

We will extend the `experiments:` frontmatter schema to accept two mutually exclusive forms via a JSON Schema `oneOf`: (1) the legacy bare-array form, unchanged, and (2) a new object form whose only required field is `variants`, plus optional fields `description`, `metric`, `weight`, `issue`, `start_date`, and `end_date`. The runtime normalizes both forms into a typed `ExperimentConfig` struct at parse time. When a `weight` array of the same length as `variants` is provided, the JS runtime uses weighted-random selection instead of round-robin; when `start_date` or `end_date` is provided and today falls outside the window, the control variant (first variant) is returned without incrementing any counter. This decision extends ADR-29534; its Rule 5 ("implementations **MUST** select the variant with the lowest cumulative invocation count") now applies only when no `weight` is provided.

### Alternatives Considered

#### Alternative 1: Separate Top-Level `experiments_config` Map

Add a parallel top-level key (`experiments_config:`) for metadata while keeping `experiments:` as bare arrays. This would avoid schema `oneOf` complexity and keep the selection algorithm single-mode. It was rejected because splitting variant lists from their metadata across two top-level keys makes frontmatter harder to read, breaks co-location, and doubles the number of keys a reader must reconcile to understand a single experiment.

#### Alternative 2: Break Backward Compatibility and Require the Object Form

Require all experiments to use the new object form (dropping bare-array support). This produces a simpler `additionalProperties` schema (no `oneOf`) and eliminates the normalization layer in the compiler. It was rejected because there are existing compiled lock files and live workflows using the bare-array form; a breaking change would require a coordinated migration of all callers without delivering user-visible value.

#### Alternative 3: Embed Weights in a Separate Env Var at Compile Time

Pass weights and metadata as additional env vars (e.g., `GH_AW_EXPERIMENT_WEIGHTS_<NAME>`) rather than embedding them in the existing `GH_AW_EXPERIMENT_SPEC` JSON blob. This avoids touching the spec format but multiplies the number of env vars injected per experiment and complicates the JS runtime which must rejoin them. It was rejected in favour of enriching the existing spec JSON, which is already a structured object.

### Consequences

#### Positive
- Full backward compatibility: all existing bare-array workflows continue to work without modification.
- Weighted selection enables statistically-designed experiments where one variant carries greater risk and should be shown less frequently.
- Date-range gating automates experiment lifecycle — no manual intervention needed to deactivate an experiment after its end date.
- Machine-readable metadata (`description`, `metric`, `issue`) enables governance tooling to discover and audit experiments without reading compiler internals.

#### Negative
- The `oneOf` in the JSON schema adds complexity to schema validation errors: consumers receive less precise error messages when the value matches neither branch.
- The compiler now calls `extractExperimentConfigsFromFrontmatter` and `extractExperimentsFromFrontmatter` separately, adding a redundant parse pass over the same frontmatter map (two passes instead of one).
- Rule 5 of ADR-29534 is partially superseded: "must use least-used selection" is now conditional on the absence of `weight`. This inconsistency between the two ADRs must be resolved before either is marked Accepted.
- Weighted random selection is non-deterministic across runs (unlike round-robin), which may produce statistically unbalanced results over small sample sizes without the experimenter understanding the difference.

#### Neutral
- The `normalizeConfig()` function in `pick_experiment.cjs` encapsulates the coercion from bare array to object form; callers in `main()` operate only on `ExperimentConfig` objects after this point.
- The `ExperimentConfig` struct is added to `frontmatter_types.go` alongside `FrontmatterConfig` and `WorkflowData`; its JSON tags use snake_case to match the YAML field names and the JS runtime's property names.
- `FrontmatterConfig.ExperimentConfigs` is tagged `json:"-"` so it does not appear in any serialized frontmatter output.

---

## Part 2 — Normative Specification (RFC 2119)

> The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this section are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### Frontmatter Schema (extends ADR-29534 §Frontmatter Schema)

1. Each value in the `experiments` map **MUST** conform to exactly one of two forms: (a) a bare array of two or more variant strings, or (b) an object with a required `variants` field (array of ≥ 2 strings) and the optional fields defined below.
2. The `variants` field in the object form **MUST** satisfy the same constraints as the bare-array form: it **MUST** contain at least two string entries.
3. If present, the `weight` field **MUST** be an array of non-negative integers whose length equals the length of `variants`; any other length **MUST** be treated as absent (i.e., fall back to round-robin).
4. If present, `start_date` and `end_date` **MUST** be ISO-8601 date strings matching the pattern `YYYY-MM-DD`; non-conforming values **SHOULD** be ignored (treated as absent) rather than causing a hard error.
5. The `description`, `metric`, and `issue` fields are **OPTIONAL** and carry no runtime effect on variant selection; implementations **MAY** surface them in step summaries or artifact metadata.

### Variant Selection (amends ADR-29534 Rule 5)

6. When `weight` is provided and its length equals the length of `variants`, implementations **MUST** use weighted-random selection: each variant is chosen with probability proportional to its weight value.
7. When all weight values are zero, implementations **MUST** fall back to the first variant (control) rather than erroring.
8. When `weight` is absent or its length does not match `variants`, implementations **MUST** use the least-used (round-robin) selection algorithm defined in ADR-29534 Rule 5.
9. Weighted-random selection **MUST NOT** increment any variant counter; counter state is only updated by round-robin selection.

### Date-Range Gating

10. When `start_date` is provided and the current date (UTC, `YYYY-MM-DD`) is strictly before `start_date`, implementations **MUST** return the control variant (first entry in `variants`) and **MUST NOT** increment any counter.
11. When `end_date` is provided and the current date (UTC, `YYYY-MM-DD`) is strictly after `end_date`, implementations **MUST** return the control variant and **MUST NOT** increment any counter.
12. Date comparison **MUST** use UTC date; local timezone offsets **MUST NOT** affect the result.
13. When both `start_date` and `end_date` are provided and the current date is within `[start_date, end_date]` (both endpoints inclusive), the experiment is active and normal selection applies.

### Compiler Integration

14. The compiler **MUST** parse both bare-array and object-form experiments in a single pass and expose the result via `WorkflowData.ExperimentConfigs` (a `map[string]*ExperimentConfig`) in addition to the existing `WorkflowData.Experiments` (`map[string][]string`).
15. `buildExperimentSpecJSON` **MUST** embed the full `ExperimentConfig` JSON object (including metadata fields) when a config is available, so that the JS runtime receives all fields in `GH_AW_EXPERIMENT_SPEC`.
16. When no config is available for a name (legacy code path), `buildExperimentSpecJSON` **MUST** fall back to emitting a bare variants array for backward compatibility.

### Conformance

An implementation is considered conformant with this ADR if it satisfies all **MUST** and **MUST NOT** requirements above. Failure to meet any **MUST** or **MUST NOT** requirement constitutes non-conformance. This ADR amends ADR-29534; in case of conflict between the two, this ADR takes precedence for the `weight` and date-range fields, and ADR-29534 governs all other aspects of the experiments feature.

---

*This is a DRAFT ADR generated by the [Design Decision Gate](https://github.com/github/gh-aw/actions/runs/25232335913) workflow. The PR author must review, complete, and finalize this document before the PR can merge.*
