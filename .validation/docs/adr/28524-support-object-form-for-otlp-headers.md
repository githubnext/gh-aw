# ADR-28524: Support Object Form for `observability.otlp.headers`

**Date**: 2026-04-26
**Status**: Draft
**Deciders**: pelikhan

---

## Part 1 — Narrative (Human-Friendly)

### Context

Workflow frontmatter supports an `observability.otlp.headers` field for passing HTTP headers to an OTLP collector. The original form accepted only a comma-separated `key=value` string (e.g., `"Authorization=Bearer ${{ secrets.TOKEN }},X-Tenant=acme"`). This forced authors to concatenate multiple secrets into a single expression, which is cumbersome, error-prone, and inconsistent with how the `env` field already accepts maps of `name: value` pairs. As observability adoption grew, users increasingly needed to specify multiple headers with individual secret references.

### Decision

We will extend `OTLPConfig.Headers` from a `string` field to a polymorphic `any` field that accepts either a map of string key-to-value pairs (preferred) or a comma-separated string (deprecated). A new `normalizeOTLPHeaders` helper converts either form into the `key=value,...` format required by the `OTEL_EXPORTER_OTLP_HEADERS` environment variable. The string form is retained for backwards compatibility but emits a deprecation warning to `stderr` on use.

### Alternatives Considered

#### Alternative 1: Keep String Form, Add Multi-Secret Concatenation Helper

A new expression helper (e.g., `${{ otlp.headers(key1=val1, key2=val2) }}`) could be introduced to construct the header string. This would avoid a type change on the Go struct but would require a new expression-language feature, adding significant implementation surface and coupling the feature to the expression evaluator. It was rejected because it adds more complexity than switching to a map.

#### Alternative 2: Introduce a Structured List/Array Form

Headers could be expressed as a list of `{name, value}` objects: `headers: [{name: Authorization, value: "Bearer ${{ secrets.TOKEN }}"}]`. This is more explicit and mirrors patterns in other YAML-based CI systems. However, it is more verbose than a map for the common case and would require a separate deprecation/migration path from the current string form. The map form was preferred as it mirrors the established `env` pattern already familiar to workflow authors.

### Consequences

#### Positive
- Individual header values can reference separate GitHub Actions secrets, improving security hygiene.
- The map syntax is consistent with the `env` field pattern, reducing cognitive overhead for authors.
- Comprehensive test coverage for both forms reduces regression risk during the deprecation window.

#### Negative
- `OTLPConfig.Headers` is now typed as `any` in Go, requiring runtime type assertions wherever the field is read; any code that directly accessed `Headers` as a `string` must be updated.
- Two valid input forms must be supported and tested throughout the deprecation window, increasing maintenance burden.

#### Neutral
- JSON Schema is updated to `oneOf: [object, string]`, which may affect tooling that provides schema-based autocompletion.
- Non-string values inside the map are silently skipped with a debug log rather than producing a validation error; stricter validation may be desirable in future.

---

## Part 2 — Normative Specification (RFC 2119)

> The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this section are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### Header Field Schema

1. The `observability.otlp.headers` field **MUST** accept both a string value and an object (map of string keys to string values).
2. The JSON schema for this field **MUST** express the two accepted types using `oneOf` with separate sub-schemas for the object form and the string form.
3. The object form **MUST** be listed first in the `oneOf` array and **MUST** be the documented preferred form.

### Normalization

1. Implementations **MUST** convert the headers value (regardless of input form) into the `key=value,...` format before injecting it as `OTEL_EXPORTER_OTLP_HEADERS`.
2. When the headers value is a map, implementations **MUST** produce a deterministic output by sorting keys lexicographically.
3. When a map entry's value is not a string, implementations **MUST NOT** include that entry in the normalized output and **SHOULD** emit a debug-level log message identifying the skipped key.
4. When the headers value is a `nil`, empty string, or empty map, implementations **MUST** produce an empty string and **MUST NOT** inject the `OTEL_EXPORTER_OTLP_HEADERS` variable.

### Deprecation

1. When the string form is used, implementations **MUST** emit a deprecation warning to `stderr` directing authors to use the map form.
2. Implementations **MUST NOT** reject or fail compilation when the string form is provided; the string value **MUST** be passed through unchanged to `OTEL_EXPORTER_OTLP_HEADERS`.
3. Implementations **SHOULD NOT** remove the string form without a documented removal timeline and a major version bump.

### Go Type Constraint

1. The `OTLPConfig.Headers` struct field **MUST** be typed as `any` (Go `interface{}`).
2. All read sites of `OTLPConfig.Headers` **MUST** route through the `normalizeOTLPHeaders` helper rather than performing inline type assertions.

### Conformance

An implementation is considered conformant with this ADR if it satisfies all **MUST** and **MUST NOT** requirements above. Failure to meet any **MUST** or **MUST NOT** requirement constitutes non-conformance.

---

*This is a DRAFT ADR generated by the [Design Decision Gate](https://github.com/github/gh-aw/actions/runs/24943747500) workflow. The PR author must review, complete, and finalize this document before the PR can merge.*
