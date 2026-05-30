# ADR-26357: YAML Round-Trip Fidelity for Proxy Env Injection Steps

**Date**: 2026-04-15
**Status**: Draft
**Deciders**: lpcox, Copilot

---

## Part 1 — Narrative (Human-Friendly)

### Context

`injectProxyEnvIntoCustomSteps` (introduced in ADR-26322 / PR #26322) injects DIFC proxy routing env vars into each custom step by parsing the step YAML with `goccy/go-yaml`, merging env vars, and re-serializing. The initial implementation used `yaml.MarshalWithOptions(map[string]any{...})` which has two correctness problems: (1) YAML deserialization strips inline comments (e.g., `uses: actions/upload-artifact@sha # v7` loses `# v7`), causing `gh-aw-manifest` to record bare SHAs instead of human-readable version tags; (2) serializing through an unordered `map[string]any` produces alphabetical field ordering, placing `env:` before `name:` and `uses:` in the output and generating noisy lock-file diffs. The codebase already solves both problems in `DeduplicateRuntimeSetupStepsFromCustomSteps` and `compiler_yaml_step_conversion.go` via established utilities (`OrderMapFields`, `unquoteUsesWithComments`), which `injectProxyEnvIntoCustomSteps` was not using.

### Decision

We will fix `injectProxyEnvIntoCustomSteps` to achieve YAML round-trip fidelity by applying three techniques already proven elsewhere in the compiler. First, version comments (e.g., `# v7`) are extracted from `uses:` lines *before* `yaml.Unmarshal` strips them and re-appended to the `uses` value string after processing (same pre-extraction pattern used in `DeduplicateRuntimeSetupStepsFromCustomSteps`). Second, each step is converted to an ordered `yaml.MapSlice` via `OrderMapFields(constants.PriorityStepFields)` before marshaling, keeping `name`/`uses` ahead of `env:`. Third, `unquoteUsesWithComments` is called on the serialized output to remove the quotes that `goccy/go-yaml` adds around strings containing `#`. Reusing existing utilities ensures consistent behavior across all compiler code paths that round-trip step YAML.

### Alternatives Considered

#### Alternative 1: Comment-Aware YAML Library

Use a YAML library that natively preserves comments during round-trip serialization (e.g., `gopkg.in/yaml.v3` with node-based API, or a dedicated comment-preserving fork). This would avoid the pre-extraction workaround. It was not chosen because no comment-preserving YAML library is currently used in the codebase, adding one introduces a new dependency and a different API surface, and the pre-extraction pattern is already established and tested in `DeduplicateRuntimeSetupStepsFromCustomSteps`.

#### Alternative 2: Raw String Manipulation Instead of Parse-and-Serialize

Inject proxy env vars using regex or line-based string processing directly on the YAML string, bypassing `yaml.Unmarshal`/`yaml.Marshal` entirely. This avoids the comment-stripping and ordering problems at their source. It was not chosen because raw string manipulation of YAML is fragile (indentation-sensitive, breaks on multi-line scalars, hard to reason about correctness), and the parse-and-serialize approach is the established pattern for step mutation in this compiler.

#### Alternative 3: Struct-Based Serialization with Tagged Fields

Define a Go struct for the step schema (with `yaml` struct tags in the desired field order) and marshal through that struct instead of `map[string]any`. This would give deterministic field ordering without `OrderMapFields`. It was not chosen because the step schema is open-ended (custom steps can have any fields), making a fully general struct impractical. `OrderMapFields` with a priority list handles both the known priority fields and unknown fields without requiring schema completeness.

### Consequences

#### Positive
- `gh-aw-manifest` records human-readable version tags (e.g., `"version":"v7"`) instead of bare SHAs when compiled lock files contain annotated `uses:` references.
- Lock-file diffs are stable and readable: `name:` and `uses:` fields appear before `env:` in every compiled step.
- No new dependencies or utility functions: the fix reuses `OrderMapFields`, `unquoteUsesWithComments`, and the version-comment pre-extraction pattern already present in the codebase.

#### Negative
- The pre-extraction pattern is an extra pass over the input string (one `strings.SplitSeq` scan) before parsing. For typical workflow sizes this is negligible, but it adds a code path that must be maintained if the `uses:` comment convention changes.
- Embedding the comment inside the `uses` string value (e.g., `"actions/upload-artifact@sha # v7"`) before re-serialization is a semantic hack: the string briefly holds a value that is not a valid action reference. The `unquoteUsesWithComments` post-processing step is required to make the output valid again, creating a subtle ordering dependency between these two steps.

#### Neutral
- Four workflow lock files (`contribution-check`, `daily-issues-report`, `issue-arborist`, `stale-repo-identifier`) were recompiled; `uses:` lines regain their `# vX` annotations and field ordering normalizes to `name`/`uses` before `env:`.
- The fix is consistent with how `DeduplicateRuntimeSetupStepsFromCustomSteps` handles the same round-trip fidelity problem, so the two functions now share the same approach.

---

## Part 2 — Normative Specification (RFC 2119)

> The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this section are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### Version Comment Preservation

1. Implementations of `injectProxyEnvIntoCustomSteps` **MUST** extract inline version comments (of the form `# vX` or `# vX.Y.Z`) from `uses:` lines in the input string *before* calling `yaml.Unmarshal`, storing them in a map keyed by the action reference (e.g., `actions/upload-artifact@sha`).
2. Implementations **MUST** re-append the extracted comment to the `uses` field value in the parsed step map before re-serialization, so that the serialized output contains the comment.
3. Implementations **MUST** call `unquoteUsesWithComments` on the serialized output to remove YAML quotes added around strings containing `#`.
4. Implementations **MUST NOT** rely on `goccy/go-yaml` (or any other YAML library used in this codebase) to preserve comments during an `Unmarshal`/`Marshal` round-trip.

### Field Ordering

1. Implementations of `injectProxyEnvIntoCustomSteps` **MUST** convert each step map to an ordered `yaml.MapSlice` using `OrderMapFields(constants.PriorityStepFields)` before marshaling.
2. Implementations **MUST NOT** marshal step maps directly as `map[string]any`, as this produces non-deterministic (alphabetical) field ordering.
3. When `constants.PriorityStepFields` is updated, implementations **SHOULD** verify that `injectProxyEnvIntoCustomSteps` output ordering remains consistent with other step-serializing code paths (e.g., `compiler_yaml_step_conversion.go`).

### Error Handling

1. If `yaml.Unmarshal` fails or the parsed result contains zero steps, implementations **MUST** log the error and return the original `customSteps` string unchanged.
2. If `yaml.MarshalWithOptions` fails, implementations **MUST** log the error and return the original `customSteps` string unchanged.
3. Implementations **MUST NOT** return an empty string or a partial result when either parsing or serialization fails.

### Conformance

An implementation is considered conformant with this ADR if it satisfies all **MUST** and **MUST NOT** requirements above. Specifically: version comments present in the input **MUST** appear in the output (unquoted), step field ordering in the output **MUST** place priority fields (`name`, `uses`) before non-priority fields (e.g., `env`), and any parse or serialization failure **MUST** result in the original input being returned unchanged. Failure to meet any **MUST** or **MUST NOT** requirement constitutes non-conformance.

---

*This is a DRAFT ADR generated by the [Design Decision Gate](https://github.com/github/gh-aw/actions/runs/24435732366) workflow. The PR author must review, complete, and finalize this document before the PR can merge.*
