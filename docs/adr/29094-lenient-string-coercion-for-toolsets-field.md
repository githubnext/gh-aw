# ADR-29094: Lenient String Coercion for `tools.github.toolsets` Field

**Date**: 2026-04-29
**Status**: Draft
**Deciders**: pelikhan, Copilot

---

## Part 1 — Narrative (Human-Friendly)

### Context

The `tools.github.toolsets` field in workflow frontmatter configures which GitHub toolsets are available to the agent. The field is semantically a list, so the schema declared it as `type: array`. Users writing the natural YAML scalar shorthand `toolsets: "default"` received a terse "got string, want array" validation error with no example and no list of valid values. This was a frequent friction point for single-toolset configurations, where wrapping a single name in brackets feels redundant. The error gave no guidance on what values were valid or that `[default]` was the required spelling.

### Decision

We will accept a bare string for `tools.github.toolsets` (and its alias `tools.github.toolset`) and coerce it to a one-element slice at parse time. The raw config map is immediately normalized from `string` → `[]any` so that compiled GitHub Actions YAML always emits an array, regardless of what the user wrote. The JSON schema is updated to allow both forms via `oneOf` (string | array). A field-specific hint is added to `knownOneOfFieldHints` in the error-formatting layer, so that any remaining type error (e.g., the user passes an integer) surfaces valid toolset names and both accepted syntax forms.

### Alternatives Considered

#### Alternative 1: Improve the error message only — keep strict `type: array`

Keep the schema requiring an array but rewrite the validation error to list valid toolset names and suggest the `[default]` bracket syntax. This eliminates the unhelpful error without changing the schema or parser. It was not chosen because it still forces users to change their YAML even when their intent is clear: a single string unambiguously means a one-element list. The UX improvement is incomplete if the user's YAML still fails.

#### Alternative 2: Accept string only — deprecate the array form

Simplify to a single `type: string` that names one toolset per workflow. This is the most restrictive approach and breaks all existing configs that use multiple toolsets (`toolsets: [default, repos]`). It was not chosen because multiple-toolset configurations are a documented and used feature.

### Consequences

#### Positive
- Single-toolset configurations can use natural YAML scalar syntax (`toolsets: default`) without brackets.
- Error messages for invalid types now include valid toolset names and both accepted syntax forms, reducing trial-and-error debugging.
- The same `oneOf` pattern already used for `/engine` is now applied consistently to `/tools/github/toolsets`.

#### Negative
- The parser has two code paths for the same field (`[]any` branch and `string` branch), increasing maintenance surface.
- The parser mutates the raw config map (normalizing `string` → `[]any`) as a side effect of parsing, coupling the parsing step to downstream serialization concerns.
- `oneOf` schemas can produce multi-branch validation noise for unrelated errors; the existing `cleanOneOfMessage` cleanup function mitigates this but adds indirection.

#### Neutral
- The singular `toolset` field receives identical treatment for consistency.
- The `github_toolset_name` definition is shared in the schema `$defs` section to avoid duplication between the string and array branches.
- No migration is required: existing array-form configurations continue to work unchanged.

---

## Part 2 — Normative Specification (RFC 2119)

> The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this section are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### Toolset Field Parsing

1. Implementations **MUST** accept a bare YAML string for `tools.github.toolsets` and `tools.github.toolset` and treat it as equivalent to a one-element array containing that string.
2. Implementations **MUST** accept a YAML array of strings for `tools.github.toolsets` and `tools.github.toolset`.
3. Implementations **MUST NOT** accept any other type (integer, boolean, object, etc.) for these fields; such values **MUST** produce a validation error.
4. Implementations **MUST** normalize the raw config map entry for `toolsets` or `toolset` from a bare string to a `[]any` slice before serialization so that compiled output always emits an array.

### Schema Definition

1. The JSON schema for `tools.github.toolsets` **MUST** declare the field using `oneOf` with exactly two branches: one `type: string` branch and one `type: array` branch whose items reference a shared `github_toolset_name` definition.
2. The `github_toolset_name` definition **MUST** be placed in the top-level `$defs` section of the schema and **MUST** be referenced (not inlined) from both branches to avoid duplication.

### Error Hints

1. When schema validation produces an `oneOf` type-conflict error for the path `/tools/github/toolsets`, the error formatter **MUST** append a hint listing valid toolset names and both accepted syntax forms (string and array).
2. Error hints **SHOULD** include a concrete YAML example demonstrating each accepted form.

### Conformance

An implementation is considered conformant with this ADR if it satisfies all **MUST** and **MUST NOT** requirements above. Failure to meet any **MUST** or **MUST NOT** requirement constitutes non-conformance.

---

*This is a DRAFT ADR generated by the [Design Decision Gate](https://github.com/github/gh-aw/actions/runs/25109741675) workflow. The PR author must review, complete, and finalize this document before the PR can merge.*
