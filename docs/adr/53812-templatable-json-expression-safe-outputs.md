# ADR-53812: Two-Pass Serialization for Templatable JSON Array Expressions in Safe-Outputs Config

**Date**: 2026-08-18
**Status**: Draft
**Deciders**: Unknown

---

### Context

Safe-output configuration fields such as `allowed_pull_requests` accept GitHub Actions template expressions (e.g., `${{ needs.approval_allowlist.outputs.eligible_pull_request_numbers }}`). These expressions evaluate to JSON arrays at workflow runtime. The config is embedded as a JSON string within a larger YAML workflow file, so Go's `json.Marshal` serializes expressions as quoted JSON strings. When the handler reads the config, it receives a string value (`"[\"123\",\"456\"]"`) instead of the expected JSON array (`["123","456"]`), causing the handler to fail to interpret `allowed_pull_requests` correctly and preventing the handler from starting.

### Decision

We will introduce a `templatableJSONExpression` marker type and a two-pass serialization helper (`marshalSafeOutputsConfig`). The first pass uses standard `json.Marshal`, which serializes each expression as a quoted JSON string containing a per-instance placeholder (a sentinel prefix plus a unique numeric id assigned when the expression is created). The second pass uses `bytes.ReplaceAll` to strip the surrounding quotes and placeholder, leaving the raw expression token embedded directly in the JSON output (unquoted, so it is treated as a JSON value—not a string—when expressions are substituted at runtime). The unique id makes each placeholder collision-proof against user-controlled expression text, independent of map traversal order. Builder callers use the new `AddTemplatableJSONSlice` method to opt into this behavior for fields whose expression value evaluates to a JSON array.

Before splicing, the expression itself is rewritten to `${{ toJSON(<expr>) }}` (unless already wrapped in `toJSON(...)`). This guarantees the runtime substitution is always valid JSON: `toJSON` on an array yields the array's JSON text, `toJSON` on a JSON-text string yields a quoted string (still parsed by `parseAllowedPullRequests`), and `toJSON` on an empty string yields `""` rather than an empty, unparseable token.

### Alternatives Considered

#### Alternative 1: Accept string-or-array in the handler at runtime

Change the handler's deserialization logic to detect when `allowed_pull_requests` is a JSON string that starts with `[` and attempt `json.Unmarshal` on it. This keeps the config emission unchanged. Why not chosen: it pushes a compile-time encoding concern into the runtime handler, complicates the handler's type model, and silently accepts malformed configs (strings that happen to look like arrays).

#### Alternative 2: Custom streaming JSON encoder

Replace the `json.Marshal` call with a fully custom JSON encoder that understands template expression tokens and emits them unquoted in-stream. Why not chosen: this would require a complete rewrite of the config serialization path, significantly increasing complexity, and would need to be maintained alongside future config fields. The two-pass approach achieves the same goal with minimal surface area.

### Consequences

#### Positive
- Config fields whose template expressions evaluate to JSON arrays are deserialized correctly by the handler without any handler-side changes.
- The fix is fully contained within the compiler layer (`compiler_safe_outputs_builder.go`, `safe_outputs_config_generation.go`, `safe_outputs_config_runtime.go`); no changes to runtime handler logic or the config schema are required.

#### Negative
- The intermediate JSON produced by the first pass (before post-processing) is temporarily malformed — it contains quoted sentinel strings that would break any tool inspecting or logging the raw config between the two passes.
- Callers must explicitly choose `AddTemplatableJSONSlice` over `AddTemplatableStringSlice` for fields whose expression evaluates to a JSON array; using the wrong builder method silently restores the original broken behavior.

#### Neutral
- `marshalSafeOutputsConfig` replaces direct `json.Marshal` calls in two places (`safe_outputs_config_generation.go` and `safe_outputs_config_runtime.go`), requiring a second traversal of the config map to collect sentinel strings before the replace step.
- The change introduces a new named type (`templatableJSONExpression`) and sentinel constant visible to any future builder authors who add config fields.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
