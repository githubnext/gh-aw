# ADR-0001: Semantic Function Clustering for Go Package File Organization

**Date**: 2026-04-10
**Status**: Draft
**Deciders**: [TODO: verify — pelikhan and Copilot per PR #25638]

---

## Part 1 — Narrative (Human-Friendly)

### Context

The `pkg/workflow` and `pkg/cli` packages in this repository accumulated two opposing structural problems over time: some concerns were spread too thinly (single-function files of ~11 lines each), while at least one file (`gateway_logs.go`, 1,332 lines) had grown into a monolith containing types, parsing, metrics computation, and rendering all in a single file. An automated semantic clustering analysis of the packages surfaced both patterns as maintainability risks — small files create noise and obscure shared infrastructure, while the large file creates high cognitive load and merge-conflict surface area. The goal was to apply a consistent file-organization principle across both cases without changing any logic.

### Decision

We will organize Go source files within a package by **semantic responsibility cluster**: files that share a single cohesive concern (a data layer, a processing stage, a set of parallel operations) are co-located, while files that have grown beyond a single concern are split at clean semantic boundaries. Concretely, this means merging files whose only content is a forwarding call to shared infrastructure (e.g., `missing_tool.go`, `missing_data.go`, `report_incomplete.go` → `missing_issue_reporting.go`), consolidating structurally parallel files that always change together (e.g., `add_labels.go` + `remove_labels.go` → `labels.go`), and decomposing monolithic files along their natural layer boundaries (types / parsing / metrics / rendering). WASM no-op stubs are annotated with explicit cross-reference comments pointing at the canonical non-WASM implementation to prevent signature drift.

### Alternatives Considered

#### Alternative 1: One File per Exported Symbol (Strict Single-Responsibility)

Each exported function or type lives in its own file, named after the symbol. This is common in some Go codebases and makes it trivial to find a specific function. It was rejected because it exacerbates the exact problem the clustering analysis found in `pkg/workflow`: a proliferation of tiny files with no shared context, making package-level navigation harder rather than easier. It also makes it impossible to see relationships between closely related functions at a glance.

#### Alternative 2: Single File per Package (Maximum Consolidation)

The entire package lives in one file. This is practical only for very small packages and was not seriously considered for packages already above ~500 lines. A single file would recreate the `gateway_logs.go` monolith problem at the package level, with even more severe merge-conflict and navigation costs.

#### Alternative 3: Maintain the Status Quo (No Reorganization)

Leave files as-is and tolerate both the tiny-file and monolith problems until they cause a concrete bug or blocked review. This was rejected because the semantic clustering analysis provided an objective, reproducible signal that the existing organization was sub-optimal, and the refactoring cost was low (zero logic changes required). Deferring incurs ongoing maintenance friction with no offsetting benefit.

### Consequences

#### Positive
- Maintainers navigating `pkg/cli` now find types, parsing logic, metrics, and rendering in separate files — each file has a clear, single purpose.
- Related parallel operations (add/remove labels; missing-tool/missing-data/report-incomplete reporting) are co-located, making it easier to keep them consistent.
- WASM stub files carry explicit cross-reference comments, reducing the risk of stub signatures drifting from their canonical counterparts.
- Smaller, focused files reduce per-file merge-conflict surface area.

#### Negative
- The split of `gateway_logs.go` into four files means cross-cutting concerns (e.g., a type used in both parsing and metrics) must be placed in `gateway_logs_types.go` even if the type is only incidentally shared; over time this file may accumulate types that no longer share a clear affinity.
- The file-naming convention (`gateway_logs_types.go`, `gateway_logs_parser.go`, etc.) relies on consistent prefix naming; violations of this convention are not enforced by the Go toolchain.
- Reviewers unfamiliar with the prior organization must read multiple files to trace the full gateway logs flow; the original monolith was self-contained.

#### Neutral
- This ADR establishes an implicit convention for future decomposition of other large files in the repository. Teams should treat it as a reference pattern rather than a strict rule requiring ADRs for every subsequent file reorganization of similar scale.
- No build system, import, or API surface changes are required; this is a pure file-layout reorganization.

---

## Part 2 — Normative Specification (RFC 2119)

> The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this section are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### File Granularity

1. A Go source file **MUST NOT** contain functions or types from more than one semantic responsibility cluster (e.g., a file **MUST NOT** mix data-type definitions with rendering logic).
2. A Go source file dedicated to a single forwarding call or delegating wrapper (where the file contains only a logger variable and one function that calls shared infrastructure) **SHOULD** be merged into the file that contains the shared infrastructure it wraps.
3. Structurally parallel files that always change together and share a common abstraction **SHOULD** be consolidated into a single file named after their shared concern.

### Large File Decomposition

1. A Go source file that exceeds 500 lines **SHOULD** be reviewed for decomposition along semantic layer boundaries.
2. When decomposing a large file, the resulting files **MUST** be named with a consistent shared prefix followed by a suffix that identifies the layer (e.g., `<feature>_types.go`, `<feature>_parser.go`, `<feature>_metrics.go`, `<feature>_render.go`).
3. All type definitions shared across the decomposed files **MUST** be placed in the `_types.go` file for that feature.
4. Decomposition **MUST NOT** change any function signatures, exported types, or observable behavior — it **MUST** be a pure reorganization.

### WASM Stub Maintenance

1. Every `*_wasm.go` stub file **MUST** contain a header comment that identifies the canonical (non-WASM) implementation file by name.
2. The header comment **SHOULD** include an explicit maintenance note stating that if function signatures change in the canonical file, the stub file must be updated to match.
3. WASM stub function signatures **MUST** exactly match the corresponding signatures in the canonical implementation.

### Conformance

An implementation is considered conformant with this ADR if it satisfies all **MUST** and **MUST NOT** requirements above. Specifically: files do not mix semantic clusters; decomposed file sets use consistent prefixed naming; type definitions are centralized in `_types.go` files; WASM stubs carry cross-reference comments and have matching signatures; and all reorganizations are pure (no logic changes). Failure to meet any **MUST** or **MUST NOT** requirement constitutes non-conformance.

---

*This is a DRAFT ADR generated by the [Design Decision Gate](https://github.com/github/gh-aw/actions/runs/24248236762) workflow. The PR author must review, complete, and finalize this document before the PR can merge.*
