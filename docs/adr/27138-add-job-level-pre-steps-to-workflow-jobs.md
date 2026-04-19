# ADR-27138: Add `jobs.<job-id>.pre-steps` Support for Custom and Built-in Jobs

**Date**: 2026-04-19
**Status**: Draft
**Deciders**: pelikhan, Copilot

---

## Part 1 — Narrative (Human-Friendly)

### Context

The gh-aw workflow compiler supports a top-level `pre-steps` frontmatter field that injects steps into the agent job before the checkout step. However, workflows can also define additional *custom jobs* and reference *built-in framework jobs* (e.g., `activation`, `pre_activation`) via the frontmatter `jobs` map. No mechanism existed to inject steps at a precise lifecycle position within these per-job step sequences: before checkout or remaining framework steps, but after the compiler-generated setup step. Users who need job-level environment preparation (credential configuration, environment variable injection, pre-flight validation) have no way to insert steps at that specific position without restructuring their entire `steps` list.

### Decision

We will add a `pre-steps` field under `jobs.<job-id>` in the frontmatter schema. For custom jobs, `pre-steps` are inserted after the compiler-generated GHES host configuration step and before any regular `steps`. For built-in jobs (`activation`, `pre_activation`), `pre-steps` are inserted after the step with `id: setup` and before the first `actions/checkout` step. The `pre-activation` key in the `jobs` map is treated as an alias for the internal `pre_activation` built-in job. When both a main workflow and an imported workflow define `pre-steps` for the same job, they are merged deterministically: imported `pre-steps` run first, main workflow `pre-steps` run after. Frontmatter entries that target an already-existing built-in job are treated as customization-only and do not create duplicate jobs.

### Alternatives Considered

#### Alternative 1: Positional metadata on individual steps

A `position: before-checkout` annotation could be added to individual `steps` entries to express ordering intent. This was rejected because it requires users to annotate each step individually, makes the schema more complex, and does not compose cleanly with import merging — the merged order of individually-annotated steps from multiple sources would be ambiguous.

#### Alternative 2: A separate top-level field per job (e.g., `custom-job-pre-steps`)

A new top-level frontmatter key could hold a map from job name to pre-step lists. This was rejected because it fragments step configuration across multiple top-level keys and diverges from the established pattern where all per-job configuration lives under `jobs.<job-id>`. Keeping `pre-steps` nested under `jobs.<job-id>` makes configuration co-located and consistent with `steps`, `runs-on`, `env`, and other job-level fields.

#### Alternative 3: Allow only custom jobs to have `pre-steps`, not built-in jobs

Built-in jobs could remain opaque to `pre-steps` injection, requiring users to define entirely separate custom jobs for any pre-flight logic against built-in job step sequences. This was rejected because built-in jobs such as `activation` and `pre_activation` perform critical lifecycle operations (permission checks, role validation) and legitimate use cases exist for injecting steps at a known position within them (e.g., pre-activation audit logging or environment setup). Treating built-in jobs differently would create an inconsistent extension model.

### Consequences

#### Positive
- Users can inject steps at a well-defined lifecycle point within any job — custom or built-in — without restructuring the entire `steps` list.
- The field name (`pre-steps`) mirrors the existing top-level `pre-steps` convention, making the extension point discoverable.
- Import merging preserves contributions from both imported and main workflows; neither silently drops pre-steps defined by the other.
- Built-in job customization via `jobs.<builtin-name>` is now explicitly recognized: the compiler skips duplicate job creation when a frontmatter entry targets an already-built-in job.

#### Negative
- The step insertion logic depends on detecting the `id: setup` marker in the serialized YAML step string, which is a fragile heuristic. If a future compiler change renames or restructures the setup step, the insertion point will silently fall back to appending before the first checkout step.
- The `pre-activation` → `pre_activation` alias adds an implicit name-mapping layer that must be documented and maintained as built-in job names evolve.
- The import merge strategy for `pre-steps` (concatenate rather than let main take precedence) differs from the behavior of all other conflicting job fields, which increases cognitive overhead for users reasoning about import precedence.

#### Neutral
- The `extractPinnedJobSteps` helper is introduced to share step-extraction and action-pinning logic between `pre-steps` and `steps`, reducing duplication in the compiler.
- The `docs/adr/` filename uses the PR number as the sequence identifier, consistent with existing ADR naming in this repository.

---

## Part 2 — Normative Specification (RFC 2119)

> The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this section are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### Frontmatter Schema

1. Workflows **MAY** define a `pre-steps` field under any `jobs.<job-id>` entry, containing an array of GitHub Actions step definition objects.
2. The `pre-steps` field **MUST** conform to the same step schema as the `steps` field under the same job (i.e., `$ref` to `jobs.additionalProperties.properties.steps`).
3. Implementations **MUST NOT** treat `jobs.<job-id>.pre-steps` as equivalent to the top-level `pre-steps` field; they operate at different scopes (per-job vs. agent job) and **MUST** be processed independently.

### Compilation and Placement — Custom Jobs

1. For custom jobs (jobs not already created as built-in jobs), implementations **MUST** prepend all compiler-generated setup steps (specifically the GHES host configuration step) before `pre-steps`.
2. `pre-steps` **MUST** be placed immediately after the compiler-generated GHES host configuration step and immediately before any `steps` entries.
3. If neither `pre-steps` nor `steps` are defined for a custom job, implementations **MUST NOT** emit the GHES host configuration step.

### Compilation and Placement — Built-in Jobs

1. For built-in jobs, implementations **MUST** insert `pre-steps` after the step identified by `id: setup` and before the first step containing `uses: actions/checkout@`.
2. If no step with `id: setup` is present, implementations **MUST** insert `pre-steps` before the first checkout step.
3. If no checkout step is present, implementations **MUST** append `pre-steps` at the end of the existing step list.
4. Implementations **MUST** recognize `pre-activation` as an alias for the `pre_activation` built-in job when processing `jobs.<job-id>.pre-steps`.

### Duplicate Job Prevention

1. When a `jobs.<job-id>` entry in the frontmatter targets an already-created built-in job, implementations **MUST NOT** create a duplicate custom job for that name.
2. Built-in job entries in the `jobs` map **SHOULD** be treated as customization-only; only `pre-steps` (and other explicitly supported customization fields) are applied to the existing built-in job.

### Import Merge Ordering

1. When a main workflow and one or more imported workflows both define `pre-steps` under the same `jobs.<job-id>` key, implementations **MUST** merge the lists rather than letting the main workflow's definition silently discard the imported definition.
2. Merged `pre-steps` **MUST** be ordered: imported `pre-steps` first, followed by main workflow `pre-steps`.
3. For all other conflicting fields under a job entry, the main workflow **MUST** take precedence (existing behavior is preserved).

### Action Pinning

1. Implementations **MUST** apply action pin resolution (SHA substitution) to `uses:` references within `pre-steps` entries, consistent with how pinning is applied to `steps`.

### Conformance

An implementation is considered conformant with this ADR if it satisfies all **MUST** and **MUST NOT** requirements above. Specifically: custom job `pre-steps` appear after the GHES host step and before `steps`; built-in job `pre-steps` are inserted after the `id: setup` step and before the first checkout step; import merging preserves both imported and main `pre-steps` in the specified order; no duplicate built-in jobs are created; and action pinning is applied to all `pre-steps` entries. Failure to meet any **MUST** or **MUST NOT** requirement constitutes non-conformance.

---

*This is a DRAFT ADR generated by the [Design Decision Gate](https://github.com/github/gh-aw/actions/runs/24630708768) workflow. The PR author must review, complete, and finalize this document before the PR can merge.*
