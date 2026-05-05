# ADR-29952: Semantic Function Clustering in pkg/cli

**Date**: 2026-05-03
**Status**: Draft
**Deciders**: Unknown (automated refactor by copilot-swe-agent)

---

## Part 1 — Narrative (Human-Friendly)

### Context

The `pkg/cli` package in this repository had grown organically, with utility functions placed near their first point of use rather than alongside semantically related functions. This produced four concrete cohesion findings: format helpers living in `audit_diff.go` instead of `audit_math_helpers.go`, external tool runners (actionlint, zizmor, poutine, runner-guard) mixed into a general-purpose `compile_batch_operations.go`, file-discovery functions in a parsing-focused `logs_parsing_core.go`, and path utilities in the compilation-specific `compile_file_operations.go`. The result was files with weak cohesion, long import lists, and poor discoverability for new contributors.

### Decision

We will reorganize `pkg/cli` by moving each function to the file whose existing responsibility most closely matches that function's domain, a practice known as semantic function clustering. Concretely: format helpers move to `audit_math_helpers.go`; external tool runners move to a new `compile_external_tools.go`; post-processing cleanup and warning display move to `compile_post_processing.go`; and path utilities move to `helpers.go`. The old `compile_batch_operations.go` becomes a stub with redirect comments pointing to the new locations.

### Alternatives Considered

#### Alternative 1: Leave Functions at Their Point of First Use

The status quo — functions remain in the file where they were first written, close to the call site that introduced them. This is simple and requires no migration work, but it produces the cohesion violations observed: unrelated functions accumulate in files, making each file harder to reason about and extending import graphs unnecessarily (e.g., `logs_parsing_core.go` importing `fileutil` solely to support a discovery function).

#### Alternative 2: Consolidate All Utilities into a Single `utils.go`

A common Go pattern is to create a single `utils.go` (or `helpers.go`) catch-all. This avoids cohesion violations by brute force but introduces a different anti-pattern: a file with no coherent theme that grows without bound. It also does not express domain intent; `compile_external_tools.go` signals "these are external tool integrations" in a way that `utils.go` never could.

### Consequences

#### Positive
- Each file in `pkg/cli` now has a narrower, better-defined responsibility, making it easier to locate the right file when adding or debugging a function.
- Import graphs are cleaner: `logs_parsing_core.go` drops `strings`, `constants`, and `fileutil` imports that were only needed for the moved discovery functions.
- New contributors can infer where to put a function from file names alone (e.g., a new external tool runner clearly belongs in `compile_external_tools.go`).

#### Negative
- The number of files in `pkg/cli` increases, which can feel heavyweight for a single package.
- `compile_batch_operations.go` becomes a comment-only stub rather than being deleted, preserving a redirect that may confuse readers until a follow-up PR removes it.
- Function moves break `git blame` lineage; the original authorship and rationale are only recoverable by looking at the commit that introduced the function in its old location.

#### Neutral
- No public API surface changes: all exported functions (`RunActionlintOnFiles`, `RunZizmorOnFiles`, `RunPoutineOnDirectory`, `RunRunnerGuardOnDirectory`) are re-exported from the new file, preserving callers.
- Tests are unaffected because Go's same-package compilation treats all `.go` files in `pkg/cli` as a single compilation unit.

---

## Part 2 — Normative Specification (RFC 2119)

> The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this section are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### File Organization

1. Functions in `pkg/cli` **MUST** be placed in the file whose stated responsibility most closely matches the function's domain, not in the file where the function was first needed.
2. A function **MUST NOT** be placed in a file solely because it is called from that file; co-location with semantically related functions takes precedence.
3. Format and math helpers for audit reporting **MUST** reside in `audit_math_helpers.go`.
4. External tool runner functions (actionlint, zizmor, poutine, runner-guard) **MUST** reside in `compile_external_tools.go`.
5. Post-compilation cleanup, warning display, and action cache pruning **MUST** reside in `compile_post_processing.go`.
6. General path and repository utilities **MUST** reside in `helpers.go`.
7. Log file discovery functions **MUST** reside in `logs_utils.go`, not in parsing-specific files.

### Cohesion Maintenance

1. When a new utility function is added to `pkg/cli`, the author **MUST** identify the file whose existing responsibilities most closely match before creating a new file.
2. A new file in `pkg/cli` **SHOULD** be created only when no existing file's responsibility covers the new function's domain and the function is unlikely to be a one-off.
3. Import lists in a file **SHOULD NOT** contain packages that are only needed by a function whose domain belongs elsewhere; such imports are a signal that the function **SHOULD** be moved.

### Conformance

An implementation is considered conformant with this ADR if it satisfies all **MUST** and **MUST NOT** requirements above. Failure to meet any **MUST** or **MUST NOT** requirement constitutes non-conformance.

---

*This is a DRAFT ADR generated by the [Design Decision Gate](https://github.com/github/gh-aw/actions/runs/25283930938) workflow. The PR author must review, complete, and finalize this document before the PR can merge.*
