# ADR-43345: Introduce TemplatableBoolOrInt for Config Fields Accepting Boolean, Integer, or GitHub Actions Expressions

**Date**: 2026-07-04
**Status**: Draft
**Deciders**: Unknown

---

### Context

The `deduplicate-by-title` field in `CreateIssuesConfig` was typed as `any`, which caused the field's value to be silently dropped when building the emitted handler config for the `create_issue` safe output. Additionally, users need to pass GitHub Actions expression strings (e.g., `${{ inputs.dedup }}`) to this field so that deduplication mode can be configured dynamically at runtime; the previous `any` type and schema definition only accepted booleans and integers, rejecting expression strings at validation time. A type-safe, serialization-aware representation was needed to correctly parse all three input forms (boolean, integer 0–100, or expression string) and emit them as their appropriate JSON types (JSON boolean, JSON number, or JSON string respectively).

### Decision

We decided to introduce a new string-backed type `TemplatableBoolOrInt` in `pkg/workflow/templatables.go` with custom YAML and JSON marshaling. The type stores all three forms internally as a Go string (`"true"`, `"false"`, a decimal integer, or a `${{ … }}` expression), then emits the correct native JSON value via `MarshalJSON`/`ToValue`. `CreateIssuesConfig.DeduplicateByTitle` was changed from `any` to `*TemplatableBoolOrInt`, and the handler registry was updated to call a new `AddTemplatableBoolOrInt` builder method, completing the emission path that was previously silently broken.

### Alternatives Considered

#### Alternative 1: Keep `any` type and add post-parse type-switch conversion

At the time, the `AddBoolOrInt` builder method performed a type-switch on `any` values to emit booleans or integers. The emission bug could have been fixed by simply wiring `DeduplicateByTitle` into `AddBoolOrInt` without changing the field type. However, `any` carries no type constraints at parse time, so schema validation alone blocks expression strings from entering the field. Expression strings would have required additional ad-hoc handling that bypasses the type system and is invisible in struct definitions.

#### Alternative 2: Explicit union struct (`struct { Bool *bool; Int *int; Expr *string }`)

A struct with three optional fields makes the valid value space explicit in the type system with no custom marshalers beyond implementing `yaml.Unmarshaler`. This is more verbose and requires callers to inspect which field is set. It also complicates the JSON schema `$defs` entry and adds branching in every emission site. The string-backed approach centralises all conversion logic inside the type and allows the schema to reference a single `$defs/templatable_bool_or_int` entry.

### Consequences

#### Positive
- `deduplicate-by-title` is now correctly emitted in the handler config for all three input forms, fixing the silent-drop bug that caused deduplication to never activate when configured.
- Users can pass GitHub Actions expression strings (e.g., `${{ inputs.dedup }}`) to `deduplicate-by-title`; the value is stored and emitted as a JSON string for runtime evaluation, enabling dynamic configuration without a schema change.
- The JSON schema `$defs/templatable_bool_or_int` definition is reusable for future fields that need the same three-way acceptance.

#### Negative
- `TemplatableBoolOrInt` introduces a custom string type with non-trivial YAML and JSON unmarshal/marshal logic that must be maintained and kept in sync with the schema definition.
- Integer range validation (0–100) is baked into the type's unmarshal methods, coupling the generic type to a specific field's constraints; reusing the type for a field with a different integer range would require a new type or a parameterised constructor.

#### Neutral
- All existing callsites that previously used `typeutil.ParseIntValue` to read `DeduplicateByTitle` have been updated to access the `*TemplatableBoolOrInt` pointer directly.
- Four new unit tests cover the boolean, integer, expression, and nil cases end-to-end through the config-generation pipeline.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
