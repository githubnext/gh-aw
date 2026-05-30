# ADR-27325: Organize pkg/workflow Helpers by Semantic Responsibility

**Date**: 2026-04-20
**Status**: Draft
**Deciders**: pelikhan, Copilot

---

## Part 1 — Narrative (Human-Friendly)

### Context

The `pkg/workflow` package had grown large umbrella helper files — most notably `awf_helpers.go` and `validation_helpers.go` — that mixed unrelated concerns in a single file. Engine API target resolution (hostname extraction, base path parsing, Copilot/Gemini target lookup) lived alongside AWF command and argument construction in `awf_helpers.go`. Similarly, a generic `extractStringSliceField` utility in `validation_helpers.go` duplicated the responsibility of `parseStringSliceAny`, and a string-slice config helper existed in both `safe_outputs_config_helpers.go` and needed by `config_helpers.go`. The mixture made it difficult to locate related logic, understand module boundaries, or reason about which helper was canonical.

### Decision

We will organize `pkg/workflow` internal helpers by semantic responsibility rather than grouping unrelated utilities into shared catch-all files. Specifically: engine API target resolution is extracted into a dedicated `engine_api_targets.go` file; the duplicate `extractStringSliceFromConfig` is consolidated in `config_helpers.go` and removed from `safe_outputs_config_helpers.go`; and `extractStringSliceField` in `validation_helpers.go` is removed in favor of routing call sites through `parseStringSliceAny` via a focused `parseOptionalStringSliceField` helper in `role_checks.go`. This establishes a pattern: each file owns one semantic domain, and canonical coercion helpers are reused rather than re-implemented.

### Alternatives Considered

#### Alternative 1: Keep helpers in umbrella files, add documentation

All helpers could remain in their current locations while inline comments clarify which function is canonical for each use case. This avoids touching call sites and keeps the number of files small. It was rejected because documentation cannot enforce module boundaries structurally; contributors would continue adding unrelated helpers to the nearest convenient file, perpetuating the mixed-concern pattern over time.

#### Alternative 2: Extract all helpers into a single `helpers.go` file

A single `helpers.go` within `pkg/workflow` could consolidate all small utility functions. This reduces file count but still mixes concerns — engine API parsing, string coercion, config extraction, and role logic would all reside in one place. The pattern trades one large umbrella (`awf_helpers.go`) for another (`helpers.go`), without improving discoverability.

#### Alternative 3: Move engine API helpers to a separate sub-package (`pkg/workflow/engineapi`)

A dedicated sub-package would enforce a hard import boundary and make the API target helpers reusable from outside `pkg/workflow`. This was considered as a stronger modular boundary, but rejected because the helpers are used exclusively within `pkg/workflow` and exposing them as a sub-package would introduce an unnecessary dependency edge and force exported symbols for what are logically internal concerns.

### Consequences

#### Positive
- `awf_helpers.go` is now focused exclusively on AWF command and argument construction; API target resolution is independently readable in `engine_api_targets.go`.
- Removes a duplicate helper definition (`extractStringSliceFromConfig`) between `safe_outputs_config_helpers.go` and `config_helpers.go`.
- All string-slice coercion for role/bot fields now routes through `parseStringSliceAny`, eliminating semantic drift between copies.
- Establishes a clear file-organization precedent for future helpers: file name should reflect the semantic concern.

#### Negative
- Increases the number of files in `pkg/workflow/`, which may add navigation overhead for contributors unfamiliar with the semantic clustering convention.
- Call sites of the removed `extractStringSliceField` must be updated; contributors who learned the old function name must discover the replacement.

#### Neutral
- No public API surface changes; `GetCopilotAPITarget`, `GetGeminiAPITarget`, and `DefaultGeminiAPITarget` retain their signatures and are re-exported from the new file.
- The refactor is behavior-preserving: empty-string filtering and type coercion semantics are maintained across all moved and renamed helpers.

---

## Part 2 — Normative Specification (RFC 2119)

> The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this section are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### File Organization

1. Engine API target resolution helpers (`extractAPITargetHost`, `extractAPIBasePath`, `GetCopilotAPITarget`, `GetGeminiAPITarget`, `DefaultGeminiAPITarget`) **MUST** reside in `pkg/workflow/engine_api_targets.go`.
2. New engine-specific API target helpers added in the future **MUST** be placed in `engine_api_targets.go` and **MUST NOT** be added to `awf_helpers.go` or other umbrella helper files.
3. `awf_helpers.go` **MUST NOT** contain engine API target resolution logic; it **MUST** be limited to AWF command construction and argument assembly.
4. String-slice config extraction helpers (`extractStringSliceFromConfig`) **MUST** be defined in `pkg/workflow/config_helpers.go` and **MUST NOT** be duplicated in other files within the package.

### String-Slice Parsing

1. All string-slice coercion from `any` input types within `pkg/workflow` **MUST** route through `parseStringSliceAny` as the canonical implementation.
2. New helper functions that parse string lists from frontmatter or config maps **MUST NOT** reimplement string-slice coercion inline; they **SHOULD** delegate to `parseStringSliceAny` or to a focused wrapper such as `parseOptionalStringSliceField`.
3. The `extractStringSliceField` function **MUST NOT** be re-introduced; call sites that previously used it **MUST** use `parseOptionalStringSliceField` (for role/bot fields with empty-string filtering) or `parseStringSliceAny` directly (for general coercion).

### Conformance

An implementation is considered conformant with this ADR if it satisfies all **MUST** and **MUST NOT** requirements above. Failure to meet any **MUST** or **MUST NOT** requirement — in particular, adding engine API target logic to `awf_helpers.go`, duplicating `extractStringSliceFromConfig`, or re-introducing `extractStringSliceField` — constitutes non-conformance.

---

*This is a DRAFT ADR generated by the [Design Decision Gate](https://github.com/github/gh-aw/actions/runs/24667638374) workflow. The PR author must review, complete, and finalize this document before the PR can merge.*
