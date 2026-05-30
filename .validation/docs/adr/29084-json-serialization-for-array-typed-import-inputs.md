# ADR-29084: JSON Serialization for Array-Typed Import-Inputs in Compiled Env Vars

**Date**: 2026-04-29
**Status**: Draft
**Deciders**: pelikhan, copilot-swe-agent

---

## Part 1 — Narrative (Human-Friendly)

### Context

Agentic workflows support shared workflow imports with typed `import-schema` fields, including `array`-typed inputs. When a caller passes an array value via `with:` and the shared workflow references it with `${{ github.aw.import-inputs.X }}` inside a step or job `env:` block, the compiler must serialize that value to a string. Prior to this fix, the serialization fell through to Go's `fmt.Sprint` fallback, producing the non-standard Go slice format `[a b]` instead of valid JSON `["a","b"]`. A compounding issue is that `goccy/go-yaml` — the YAML parser used in this repo — may deserialize YAML sequences as typed Go slices (`[]string`) rather than `[]any`, bypassing the existing `case []any:` JSON serialization branch entirely.

### Decision

We will serialize all array- and map-typed import-input values as JSON when writing them into compiled `env:` blocks, using a shared `marshalEnvValue()` helper that handles `[]any`, `map[string]any`, and typed slice/map variants via reflection. This applies consistently across all three serialization sites: `MapToStep` (step-level env), `buildCustomJobs` (job-level env), and `marshalImportInputValue` / `substituteImportInputsInContent` (content substitution). The `fmt.Sprint` fallback is retained only for scalar types (int, bool, float64, etc.) that do not require structured encoding.

### Alternatives Considered

#### Alternative 1: Comma-Separated String Join for Arrays

Join array elements with a comma separator (e.g., `microsoft/apm#main,github/awesome-copilot/skills/foo`) instead of JSON encoding. This would produce a simpler string that some shell scripts could split with `IFS=,`. However, it is not valid JSON, breaks for values containing commas, and is incompatible with `jq --argjson` and other JSON-consuming tools. It also doesn't compose well with map values.

#### Alternative 2: Normalize YAML Deserialization to Always Return `[]any`

Fix the root cause at the YAML parsing layer by wrapping `goccy/go-yaml` to always convert typed slices to `[]interface{}` immediately after deserialization. This would eliminate the need for reflection at the serialization sites. It was not chosen because it would require modifying shared parsing infrastructure used across many code paths, increasing the blast radius of the change. The reflection fallback is more localized and can be removed later if the YAML layer is standardized.

### Consequences

#### Positive
- Shell consumers that use `jq --argjson $VAR` now receive valid JSON arrays and objects.
- All three serialization paths (step env, job env, content substitution) are consistent and produce the same output for the same input type.
- Defense-in-depth: even if a future YAML parser upgrade changes the Go type returned for sequences, the reflection fallback ensures correct JSON output.

#### Negative
- The `reflect` package is now a dependency of three additional files in the hot compilation path, adding marginal complexity.
- Scalar non-string values (int, bool) continue to use `fmt.Sprint`, meaning there is no single uniform serialization strategy across all value types.
- Any tooling that previously relied on the `[a b]` format (unlikely, as it was a bug) would break.

#### Neutral
- The `marshalEnvValue()` helper is defined in `step_types.go` and shared with `compiler_jobs.go` via package scope; `marshalImportInputValue` and `substituteImportInputsInContent` retain their own local reflection logic in the `parser` package due to package boundaries.
- New regression tests were added covering `[]string`, `[]int`, `map[string]any`, and `[]any` inputs.

---

## Part 2 — Normative Specification (RFC 2119)

> The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this section are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### Env Value Serialization

1. Implementations **MUST** serialize array-typed import-input values as valid JSON arrays (e.g., `["a","b"]`) when writing them into compiled `env:` blocks at both step level and job level.
2. Implementations **MUST** serialize map-typed import-input values as valid JSON objects when writing them into compiled `env:` blocks.
3. Implementations **MUST NOT** use Go's `fmt.Sprint` or equivalent default string formatting for slice or map values in `env:` blocks, as this produces non-JSON output such as `[a b]`.
4. Implementations **MUST** handle typed Go slices (e.g., `[]string`, `[]int`) produced by the YAML parser via reflection, normalizing them to `[]any` before JSON marshaling.
5. Implementations **SHOULD** apply this serialization consistently across all env-writing code paths: step-level env (`MapToStep`), job-level env (`buildCustomJobs`), and content substitution (`marshalImportInputValue`, `substituteImportInputsInContent`).
6. Implementations **MAY** retain `fmt.Sprint` as a fallback for scalar non-string types (int, bool, float64) where JSON encoding is not required.

### Serialization Helper Scope

1. Implementations **MUST** use a shared serialization helper (e.g., `marshalEnvValue`) for step-level and job-level env serialization within the same package to avoid code duplication.
2. Implementations **MAY** duplicate the reflection logic in other packages (e.g., `parser`) where package boundaries prevent sharing the helper, provided the behavior is equivalent.

### Conformance

An implementation is considered conformant with this ADR if it satisfies all **MUST** and **MUST NOT** requirements above. Specifically, conformance requires that any array or map value passed as an import-input and referenced in a compiled `env:` block is serialized as a valid JSON string, and that Go's default slice formatting (`[a b]`) never appears as an env value for structured types. Failure to meet any **MUST** or **MUST NOT** requirement constitutes non-conformance.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
