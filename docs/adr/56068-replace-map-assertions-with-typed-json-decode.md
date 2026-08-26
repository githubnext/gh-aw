# ADR-56068: Replace Manual map[string]any Assertions with Typed JSON Decode

**Date**: 2026-08-26
**Status**: Draft
**Deciders**: Unknown

---

### Context

Bootstrap manifest actions and engine auth/request configurations in `pkg/cli/bootstrap_profile_manifest.go` and `pkg/workflow/engine_config_parser.go` were decoded field-by-field using manual `map[string]any` type assertions (e.g., `actionMap["field"].(string)`). When a configuration field had a wrong type or an unexpected value, the assertion silently failed and the field was left at its zero-value. This meant malformed user-provided manifests and engine configs could pass validation without producing an actionable error, causing subtle runtime failures instead of clear configuration-time feedback. The affected structs (`repositoryPackageBootstrapAction`, `AuthDefinition`, `EngineAuthConfig`, `RequestShape`) already had well-defined field types, making per-field assertion redundant.

### Decision

We will replace all per-field `map[string]any` type assertions with a single JSON round-trip decode: marshal the incoming map to JSON with `json.Marshal`, then decode into the target typed struct using `json.NewDecoder`. For bootstrap manifest actions, `DisallowUnknownFields()` is applied to reject unrecognised keys. Type mismatches are caught by `json.UnmarshalTypeError` and surfaced as field-specific manifest errors. Engine config parsers adopt the same decode path but log-and-continue on error to preserve backward compatibility for unrecognised engine configurations.

### Alternatives Considered

#### Alternative 1: Add Explicit Per-Field Type Checks with Error Returns

Extend the existing manual assertion code to check the asserted type explicitly and return an error when the assertion fails, rather than silently skipping the field. This would avoid the JSON round-trip but requires adding a matching error-path for each of the 20+ assertion sites. The result is more code, not less, and still requires manual maintenance as new fields are added to the structs.

#### Alternative 2: Use a Third-Party mapstructure Decoder

Use `github.com/mitchellh/mapstructure` (already a common Go ecosystem choice) to decode `map[string]any` directly into typed structs with struct-tag support. This avoids the JSON serialisation cost and supports custom decode hooks. However, it adds a new external dependency, requires a separate tag format alongside existing `yaml:` and `json:` tags, and the project already uses `encoding/json` pervasively — making the standard library approach the lower-friction choice.

### Consequences

#### Positive
- Bootstrap action parsing now returns field-specific, user-readable errors for type mismatches (e.g., `config[0].prompt has an invalid type`) instead of silently accepting garbage values.
- Approximately 120 lines of repetitive per-field assertion code are removed, and new struct fields automatically participate in decoding without requiring assertion additions.

#### Negative
- Engine config decoders (`parseAuthDefinition`, `parseEngineAuthConfig`, `parseRequestShape`) log and return a zero-value struct on decode error rather than propagating the error, preserving silent-failure for invalid engine configurations. This is intentionally asymmetric with bootstrap action handling but means misconfigured engine auth is still not surfaced at configuration time.
- `DisallowUnknownFields()` is applied only to bootstrap action decoding, not to engine config decoding, creating inconsistent strictness between the two code paths. Unrecognised engine config keys are silently ignored.

#### Neutral
- JSON struct tags (`json:"..."`) are added to `AuthDefinition`, `EngineAuthConfig`, and `RequestShape` fields; the existing `yaml:` tags on those structs are unchanged, so YAML-based loading paths are unaffected.
- The JSON round-trip (marshal then decode) has negligible performance impact for configuration objects that are parsed once at startup.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
