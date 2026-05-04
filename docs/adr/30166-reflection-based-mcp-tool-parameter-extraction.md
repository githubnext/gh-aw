# ADR-30166: Reflection-Based MCP Tool Parameter Extraction

**Date**: 2026-05-04
**Status**: Draft
**Deciders**: pelikhan, copilot-swe-agent

---

## Part 1 — Narrative (Human-Friendly)

### Context

The `mcpToolParams()` function in `pkg/cli/mcp_argument_validation.go` maintained a hardcoded map from MCP tool name to the list of valid JSON parameter names. This map fed the "Did you mean?" suggestion system for unrecognised parameters. The map was already out of sync: the `audit` tool's `experiment` and `variant` fields existed in `auditArgs` but were absent from the hardcoded map. A `// MAINTENANCE:` comment warned contributors to keep the map aligned with the `*Args` structs, but that approach had already failed in practice. The codebase has ten MCP tools, each backed by a dedicated `*Args` struct; the mismatch was invisible until a user hit a confusing suggestion gap.

### Decision

We will replace the hardcoded parameter map in `mcpToolParams()` with a reflection-based approach: each `*Args` struct is elevated from function-local to package scope, and a new `jsonFieldNames(v any) []string` helper uses `reflect` to extract JSON tag names from exported struct fields at runtime. This eliminates the need for any manual map maintenance and ensures the "Did you mean?" suggestions are always derived from the live struct definitions.

### Alternatives Considered

#### Alternative 1: Improve the Hardcoded Map with Stronger Test Coverage

The existing hardcoded map could be kept, but supplemented with a generated "golden file" test that would fail if a struct field appeared in an `*Args` struct but not in the map. This would catch future drift at CI time. However, it still requires a two-step authoring process (add the field, then update the map and/or golden file), and it addresses the symptom rather than the root cause. The current test already failed to catch the `experiment`/`variant` gap, suggesting the feedback loop is not tight enough.

#### Alternative 2: Code Generation (go generate)

A `go generate` script could introspect the `*Args` structs and emit a static Go file containing the map. This avoids reflection at runtime and provides a checked-in artifact that shows exactly what parameters are registered. The downside is build complexity: the generated file must be re-run whenever an `*Args` struct changes, contributors must remember to commit the generated output, and CI must verify it is up to date. For a map this small (10 tools, ~60 parameters total), the overhead of a generation pipeline outweighs the runtime-inspection cost.

### Consequences

#### Positive
- Adding a field to any `*Args` struct automatically includes it in "Did you mean?" suggestions; no secondary change is required.
- Fixes an existing correctness bug where `experiment` and `variant` were silently absent from `audit` tool suggestions.
- Removes the `// MAINTENANCE:` comment and its associated cognitive burden.

#### Negative
- All `*Args` structs are now at package scope in their respective files (`mcp_tools_readonly.go`, `mcp_tools_privileged.go`, `mcp_tools_management.go`), meaning they are technically exported-visible within the `cli` package and accessible from test files — types that were previously encapsulated in function bodies.
- The `reflect` package is imported; `mcpToolParams()` now has an implicit runtime dependency on struct tag formatting conventions (e.g., the `json:"-"` sentinel).

#### Neutral
- The `jsonFieldNames` helper sorts its output; the earlier code sorted the map values in a post-processing loop. Semantics are unchanged, but the sorting is now done inside the helper rather than in the caller.
- Tests have been updated to verify the reflection path, including a new `TestJSONFieldNames` unit test and an extended `TestMCPToolParams` spot-check for `experiment` and `variant`.

---

## Part 2 — Normative Specification (RFC 2119)

> The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this section are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### Args Struct Scope

1. Every `*Args` struct that backs an MCP tool **MUST** be declared at package scope in its respective source file so it is accessible for reflection by `mcpToolParams()`.
2. `*Args` structs **MUST NOT** be declared inside the body of their `register*Tool` function, as function-local types cannot be referenced by `mcpToolParams()`.
3. Each exported field of an `*Args` struct that represents a valid MCP parameter **MUST** carry a `json` struct tag with a non-empty, non-`"-"` name.

### Parameter Extraction

1. The `jsonFieldNames` helper **MUST** use `reflect` to extract JSON tag names and **MUST NOT** rely on any manually maintained list or generated artifact.
2. `jsonFieldNames` **MUST** skip unexported fields, fields with no `json` tag, and fields whose `json` tag name is `"-"`.
3. `jsonFieldNames` **MUST** return a sorted slice to ensure deterministic output in "Did you mean?" suggestions.
4. `mcpToolParams()` **MUST** derive each tool's parameter list by calling `jsonFieldNames` with a zero-value instance of the corresponding `*Args` struct.
5. `mcpToolParams()` **MUST NOT** contain any hardcoded parameter name strings.

### Conformance

An implementation is considered conformant with this ADR if it satisfies all **MUST** and **MUST NOT** requirements above. Failure to meet any **MUST** or **MUST NOT** requirement constitutes non-conformance.

---

*This is a DRAFT ADR generated by the [Design Decision Gate](https://github.com/github/gh-aw/actions/runs/25332888181) workflow. The PR author must review, complete, and finalize this document before the PR can merge.*
