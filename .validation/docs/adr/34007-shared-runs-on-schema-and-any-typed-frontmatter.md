# ADR-34007: Shared `runs-on` Schema Across Top-Level and Jobs With `any`-Typed Frontmatter Field

**Date**: 2026-05-22
**Status**: Draft
**Deciders**: Unknown

---

## Part 1 — Narrative (Human-Friendly)

### Context

The custom workflow schema in `pkg/parser/schemas/main_workflow_schema.json` defined two independent `runs-on` shapes: a rich top-level definition that accepted string, array, and object (`{ group, labels }`) forms, and a duplicate `jobs.*.runs-on` definition whose object branch had `additionalProperties: false` with no declared properties, so any object value was rejected at schema-validation time. The compiler and runtime already understood the object form everywhere (it round-trips through GitHub Actions natively), so the gap was purely a validation artefact that blocked users from writing `jobs.<id>.runs-on: { group: arc-custom }` even though the generated workflow would have worked. On the Go side, `FrontmatterConfig.RunsOn` was typed `string`, which caused JSON unmarshalling to fail for the array and object forms before validation could even report a useful error, so the typed parser was the second copy of the same restriction.

### Decision

We will replace both `runs-on` schemas with a single `$ref` to a shared `#/$defs/github_actions_runs_on` definition that allows string, array, and object forms, and we will widen `FrontmatterConfig.RunsOn` from `string` to `any` so the typed frontmatter parser preserves whichever shape the user supplied. The serializer in `pkg/workflow/frontmatter_serialization.go` is updated to emit the field whenever it is non-`nil` (replacing the old non-empty-string check), so round-trip through `ToMap` is fidelity-preserving for all three forms. The driving principle is "validate against one source of truth that matches GitHub Actions semantics" rather than maintaining two parallel definitions that can drift.

### Alternatives Considered

#### Alternative 1: Fix the job-level object branch in place, keep two schemas

Add `properties: { group, labels }` to the inline `jobs.*.runs-on` object branch so it matches the top-level definition, leaving the two schemas physically separate. Rejected because the duplication is precisely what produced the original divergence: every future addition to `runs-on` semantics (e.g., a new GitHub Actions property) would need to be applied in two places, and the regression test would only catch drift that the author also remembered to write twice.

#### Alternative 2: Introduce a typed `RunsOn` union struct in Go

Replace the `string` field with a custom type that implements `json.Unmarshaler` / `yaml.Unmarshaler` and exposes typed accessors (`AsString`, `AsLabels`, `AsGroup`). Rejected because the field is passed through to YAML emission as-is — consumers of `FrontmatterConfig` do not branch on the runs-on shape, they only need to preserve it. A typed union would add unmarshaling code, generated-workflow code paths, and tests for zero functional benefit over `any` with round-trip preservation.

#### Alternative 3: Restrict job-level `runs-on` to string and array only

Leave the schema rejecting object form at job level and document the restriction. Rejected because the compiler and runtime already accept the object form for jobs, and the PR description shows that real users (ARC custom runner groups) need exactly this shape; the schema would remain the only thing standing in the way of a working workflow.

### Consequences

#### Positive
- `jobs.<id>.runs-on: { group: <name>, labels: [...] }` is accepted at schema-validation time, matching the compiler's existing behaviour and unblocking ARC / runner-group setups for job-level configuration.
- Top-level `runs-on` no longer fails JSON unmarshalling when the user supplies array or object forms — the field is preserved end-to-end through the typed config.
- The shared `$defs/github_actions_runs_on` definition becomes the single source of truth for runs-on validation; future additions need to be made in exactly one place.
- The new tests (`TestValidateMainWorkflowFrontmatterWithSchemaAndLocation_AcceptsJobRunsOnObjectForm`, the three sub-tests in `frontmatter_types_test.go`) lock in regression coverage for all three forms at both positions.

#### Negative
- `FrontmatterConfig.RunsOn` is now `any`, so any future consumer that wants to branch on the runs-on shape must type-assert (`switch v := fc.RunsOn.(type)`). Compile-time guarantees about the field's type are lost.
- The serializer's "emit when non-nil" rule means a caller that explicitly sets `fc.RunsOn = ""` will now serialise an empty string instead of omitting the key. Existing call sites use the zero value (untouched `any`), which is `nil`, so this is latent rather than active risk.
- The schema is slightly less self-contained at the point of use: a reader of `jobs.*.runs-on` now has to follow a `$ref` to see what is allowed, instead of reading the shape inline.

#### Neutral
- The `$defs/github_actions_runs_on` definition is a verbatim copy of the previous top-level inline schema, including descriptions and examples; behaviour for top-level `runs-on` is unchanged.
- The `examples` array stays at the top-level use site rather than moving into `$defs`, which keeps top-level documentation visible without forcing the same examples onto the job-level binding.
- The PR also updates the actions lockfile and one workflow lock to a newer `docker/metadata-action@v6` pin; that change is unrelated to the schema decision and is not covered by this ADR.

---

## Part 2 — Normative Specification (RFC 2119)

> The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this section are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### Schema Definition

1. The workflow schema **MUST** declare a single `$defs/github_actions_runs_on` definition whose top-level `oneOf` accepts: a `string`, an `array` of `string`, and an `object` with `additionalProperties: false` and optional properties `group` (`string`) and `labels` (`array` of `string`).
2. The top-level `runs-on` property **MUST** reference `#/$defs/github_actions_runs_on` and **MUST NOT** inline its own `oneOf` shape.
3. The `jobs.<job_id>.runs-on` property **MUST** reference `#/$defs/github_actions_runs_on` and **MUST NOT** inline its own `oneOf` shape.
4. The schema **MUST** validate a workflow where `jobs.<job_id>.runs-on` is the object `{ "group": "<name>" }` as conformant.
5. The schema **MUST** validate a workflow where `jobs.<job_id>.runs-on` is the object `{ "group": "<name>", "labels": ["<a>", "<b>"] }` as conformant.

### Typed Frontmatter Parsing

1. The `RunsOn` field on `FrontmatterConfig` **MUST** be declared with Go type `any` and tagged `json:"runs-on,omitempty"`.
2. `ParseFrontmatterConfig` **MUST** preserve the user-supplied shape of `runs-on` such that:
   - a string input is exposed as a `string` value,
   - an array input is exposed as a `[]any` value,
   - an object input is exposed as a `map[string]any` value.
3. `FrontmatterConfig.ToMap` **MUST** include the `"runs-on"` key when `fc.RunsOn` is non-`nil` and **MUST** omit it when `fc.RunsOn` is `nil`.
4. `FrontmatterConfig.ToMap` **MUST** round-trip the runs-on value such that the value at key `"runs-on"` is identical (by type and content) to `fc.RunsOn`.

### Regression Coverage

1. The parser test suite **MUST** contain a test that calls `ValidateMainWorkflowFrontmatterWithSchemaAndLocation` with a workflow whose `jobs.<job_id>.runs-on` is `map[string]any{"group": "<name>"}` and asserts that no validation error is returned.
2. The frontmatter typed-parsing test suite **MUST** contain tests that round-trip `runs-on` through `ParseFrontmatterConfig` + `ToMap` for the string, array, and object forms and assert the value is preserved.

### Conformance

An implementation is considered conformant with this ADR if it satisfies all **MUST** and **MUST NOT** requirements above. Failure to meet any **MUST** or **MUST NOT** requirement constitutes non-conformance.

---

*This is a DRAFT ADR generated by the [Design Decision Gate](https://github.com/github/gh-aw/actions/runs/26291961745) workflow. The PR author must review, complete, and finalize this document before the PR can merge.*
