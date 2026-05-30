# ADR-29336: Relocate Misplaced Functions Identified by Semantic Clustering Analysis

**Date**: 2026-04-30
**Status**: Draft
**Deciders**: pelikhan, Copilot

---

## Part 1 — Narrative (Human-Friendly)

### Context

Semantic function clustering analysis flagged 6 functions living in files whose names misrepresent their semantic domain. In `pkg/cli`, `GetBinaryPath` and `logAndValidateBinaryPath` resided in `mcp_validation.go` despite being path-resolution utilities with no validation logic. In `pkg/workflow`, `parseStringSliceAny` and `preprocessProtectedFilesField` lived in `validation_helpers.go` despite being data-coercion and preprocessing functions rather than validators. Also in `pkg/workflow`, `ValidateHeredocContent` and `ValidateHeredocDelimiter` were the only two exported validation functions outside the package's 44 `*_validation.go` files, stranded in `strings.go` alongside unrelated string utilities. This misplacement violated the file-naming convention established in ADR-27325 and extended by ADR-28282, making the codebase harder to navigate for contributors relying on file names to locate related logic.

### Decision

We will relocate each misplaced function to a file whose name accurately communicates its semantic domain: `GetBinaryPath` and `logAndValidateBinaryPath` move to a new `pkg/cli/mcp_helpers.go`; `parseStringSliceAny` and `preprocessProtectedFilesField` move to a new `pkg/workflow/parse_helpers.go`; `ValidateHeredocContent` and `ValidateHeredocDelimiter` move to a new `pkg/workflow/heredoc_validation.go`. All moves are intra-package; no call sites change and no behavior is altered. This applies the same outlier-relocation convention from ADR-28282 to a new set of functions surfaced by automated semantic clustering.

### Alternatives Considered

#### Alternative 1: Leave Functions in Place with Cross-File Documentation

Inline comments could document that a function in file A primarily supports a concern owned by file B. This was rejected because documentation without structural enforcement degrades: contributors continue adding similar misplaced functions to the nearest convenient file rather than the semantically correct one, and the problem compounds over time. This alternative was also explicitly rejected in ADR-28282 for the same reason.

#### Alternative 2: Consolidate Into a Single Umbrella `helpers.go`

All small utilities could be gathered into a single per-package `helpers.go` to reduce file count and eliminate placement decisions. This was rejected because it recreates the catch-all pattern that ADR-27325 explicitly discourages — a function's location would again fail to communicate its purpose, defeating the file-naming convention the codebase has adopted.

### Consequences

#### Positive
- `GetBinaryPath` and `logAndValidateBinaryPath` are co-located in `mcp_helpers.go`, making binary path resolution logic discoverable in a single file.
- All data-coercion and preprocessing helpers are consolidated in `parse_helpers.go`, clearly separated from the validators in `validation_helpers.go`.
- `ValidateHeredocContent` and `ValidateHeredocDelimiter` join their peers in `heredoc_validation.go`, completing the naming convention where all validation functions in `pkg/workflow` live in `*_validation.go` files.
- Reinforces the semantic file-organization convention from ADR-27325 and ADR-28282 across new areas of the codebase.

#### Negative
- Increases file count in `pkg/cli/` (by 1) and `pkg/workflow/` (by 2), adding marginal navigation overhead.
- Adds churn to `git log` for the relocated functions; `git blame` will surface the relocation commit rather than original authorship without `--follow`.

#### Neutral
- No public API surface changes; all moved functions retain their signatures.
- The now-unused `errors` import is removed from `strings.go` as a side effect of moving `ValidateHeredocContent`.
- The `validation_helpers.go` header comment is updated to remove stale references to the relocated parse helpers.

---

## Part 2 — Normative Specification (RFC 2119)

> The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this section are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### Binary Path Utilities (`pkg/cli`)

1. `GetBinaryPath` and `logAndValidateBinaryPath` **MUST** reside in `pkg/cli/mcp_helpers.go`.
2. New binary path resolution helpers used by MCP server or validation logic **MUST** be added to `mcp_helpers.go` and **MUST NOT** be placed in `mcp_validation.go`.
3. `mcp_validation.go` **MUST NOT** contain path-resolution utility functions; its scope **MUST** be limited to MCP server configuration validation logic.

### Parse and Preprocessing Helpers (`pkg/workflow`)

1. `parseStringSliceAny` and `preprocessProtectedFilesField` **MUST** reside in `pkg/workflow/parse_helpers.go`.
2. New functions that coerce or preprocess raw configuration data before validation **MUST** be added to `parse_helpers.go` and **MUST NOT** be placed in `validation_helpers.go`.
3. `validation_helpers.go` **MUST NOT** contain data-coercion or preprocessing functions; its scope **MUST** be limited to validation logic that checks constraints and returns errors.

### Heredoc Validation (`pkg/workflow`)

1. `ValidateHeredocContent` and `ValidateHeredocDelimiter` **MUST** reside in `pkg/workflow/heredoc_validation.go`.
2. New exported validation functions for heredoc safety **MUST** be added to `heredoc_validation.go` and **MUST NOT** be placed in `strings.go` or other non-validation files.
3. `strings.go` **MUST NOT** contain exported validation functions; its scope **MUST** be limited to string transformation and normalization utilities.

### General File-Naming Convention

1. All exported validation functions within `pkg/workflow` **MUST** reside in files whose names match the `*_validation.go` pattern.
2. Functions whose semantic domain is misrepresented by their current file **SHOULD** be relocated to a file whose name accurately communicates that domain when identified by code review or automated analysis.

### Conformance

An implementation is considered conformant with this ADR if it satisfies all **MUST** and **MUST NOT** requirements above. Failure to meet any **MUST** or **MUST NOT** requirement — in particular, placing path-resolution utilities in `mcp_validation.go`, placing data-coercion helpers in `validation_helpers.go`, or placing exported validation functions in `strings.go` — constitutes non-conformance.

---

*This is a DRAFT ADR generated by the [Design Decision Gate](https://github.com/github/gh-aw/actions/runs/25177070075) workflow. The PR author must review, complete, and finalize this document before the PR can merge.*
