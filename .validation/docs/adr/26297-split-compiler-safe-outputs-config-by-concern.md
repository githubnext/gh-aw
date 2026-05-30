# ADR-26297: Split compiler_safe_outputs_config.go by Concern

**Date**: 2026-04-14
**Status**: Draft
**Deciders**: pelikhan, Copilot

---

## Part 1 — Narrative (Human-Friendly)

### Context

`pkg/workflow/compiler_safe_outputs_config.go` had grown to 1,035 lines mixing three distinct responsibilities into a single file: (1) the fluent `handlerConfigBuilder` infrastructure and footer helper functions, (2) the `handlerRegistry` map containing 30+ individual handler configuration generator functions, and (3) the top-level orchestrator (`addHandlerManagerConfigEnvVar`) that drives the `GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG` env var generation. This made the file difficult to navigate—a contributor modifying one handler had to scroll through unrelated builder code and orchestration logic to find it. The file was a classic "God file" within an otherwise well-structured package.

### Decision

We will split `compiler_safe_outputs_config.go` into three files within the same `workflow` package, each owning a single concern: `compiler_safe_outputs_builder.go` holds the `handlerConfigBuilder` struct with its fluent methods, the `handlerBuilder` type alias, and footer helper functions; `compiler_safe_outputs_handlers.go` holds the `handlerRegistry` map with all individual handler config generator functions; and `compiler_safe_outputs_config.go` is reduced to 136 lines containing only the top-level orchestrator, workflow-call relay helpers, and engine agent file info. All files remain in `package workflow`, so no import paths or public APIs change. No logic is modified; this is a pure structural reorganization to improve file-level navigability.

### Alternatives Considered

#### Alternative 1: Keep Everything in a Single File

The file could remain as-is. This is the simplest option—no merge conflicts, no navigability changes. It was rejected because 1,035 lines with three independent concerns (builder infrastructure, handler registry, orchestration) makes the file genuinely hard to navigate; finding any single handler requires scrolling through or searching within hundreds of lines of unrelated code.

#### Alternative 2: Extract into a Separate Sub-Package

The handler registry and builder could have been moved into a dedicated sub-package (e.g., `pkg/workflow/safeoutputsconfig/`). This would provide stronger compile-time boundaries and make the separation visible at the import level. It was not chosen because the handler functions reference unexported types in the `workflow` package, and moving them would require either exporting those types or significantly restructuring package boundaries—a change well beyond the scope of the navigability problem being solved.

#### Alternative 3: Split by Type vs. Function Rather Than by Concern

An alternative structure would group all struct/type definitions in one file and all functions in another, regardless of concern. This was rejected because it does not improve navigability for the primary use case: understanding or modifying all logic related to one concern (e.g., the handler registry). Concern-based grouping collocates related types and functions.

### Consequences

#### Positive
- Each new file is ≤800 lines and scoped to a single concern, improving navigability for contributors editing builder infrastructure, handler configs, or orchestration independently.
- The `handlerRegistry` in `compiler_safe_outputs_handlers.go` is now the unambiguous single source of truth for all handler names and field contracts, making it easy to audit or extend handlers without encountering unrelated code.
- `compiler_safe_outputs_config.go` is reduced from 1,035 to 136 lines; the orchestration intent is now immediately visible without builder or handler noise.
- No API surface changes—callers outside the package are unaffected.

#### Negative
- The codebase now has more files, which adds overhead when doing a first-time sweep of the package.
- Cross-concern relationships (e.g., the orchestrator in `compiler_safe_outputs_config.go` iterating over `handlerRegistry` from `compiler_safe_outputs_handlers.go`) are less immediately visible than when everything was co-located.

#### Neutral
- Go's intra-package visibility means all unexported identifiers remain accessible across the split files; no visibility changes are needed.
- IDE tooling and `go build` are unaffected by intra-package file splits.
- The `handlerRegistry` map (~783 lines) remains the largest single unit and is a candidate for further decomposition if the number of handlers grows substantially.

---

## Part 2 — Normative Specification (RFC 2119)

> The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this section are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### File Organization

1. Implementations **MUST** keep all `compiler_safe_outputs_config*.go` files in the same Go package (`package workflow`).
2. Implementations **MUST NOT** introduce a new sub-package solely to house the split files; the existing `pkg/workflow` package boundary **SHALL** be maintained.
3. Implementations **SHOULD** keep each split file focused on a single concern; a file **SHOULD NOT** contain logic belonging to another concern's designated file.

### Builder Infrastructure

1. The `handlerConfigBuilder` struct, all its fluent methods (`AddIfPositive`, `AddIfNotEmpty`, `AddStringSlice`, `AddBoolPtr`, `AddBoolPtrOrDefault`, `AddStringPtr`, `AddDefault`, `AddIfTrue`, `Build`, and any future additions), the `handlerBuilder` type alias, and the `getEffectiveFooter*` helper functions **MUST** reside in `compiler_safe_outputs_builder.go`.
2. Implementations **MUST NOT** duplicate the `handlerConfigBuilder` definition in another file; it **SHALL** be defined in exactly one location.

### Handler Registry

1. The `handlerRegistry` map and all individual handler config generator functions **MUST** reside in `compiler_safe_outputs_handlers.go`.
2. The `handlerRegistry` **MUST** be the single source of truth for handler names and field contracts; handler name strings **MUST NOT** be defined outside this map.
3. New handler implementations **MUST** be added as entries in `handlerRegistry` in `compiler_safe_outputs_handlers.go` and **MUST NOT** be inlined into the orchestrator.

### Orchestration

1. The top-level orchestrator function `addHandlerManagerConfigEnvVar`, the workflow-call relay helpers (`safeOutputsWithDispatchTargetRepo`, `safeOutputsWithDispatchTargetRef`), and the engine file info helper (`getEngineAgentFileInfo`) **MUST** remain in `compiler_safe_outputs_config.go`.
2. The orchestrator **MUST NOT** contain inline handler config construction logic; it **SHALL** delegate to `handlerRegistry` for all per-handler configuration.

### Conformance

An implementation is considered conformant with this ADR if it satisfies all **MUST** and **MUST NOT** requirements above. Failure to meet any **MUST** or **MUST NOT** requirement constitutes non-conformance. The primary conformance checks are: (a) all split files are in `package workflow`, (b) no new sub-package is created, (c) builder infrastructure is collocated in `compiler_safe_outputs_builder.go`, (d) the handler registry and all handler functions are collocated in `compiler_safe_outputs_handlers.go`, and (e) orchestration logic remains in `compiler_safe_outputs_config.go`.

---

*This is a DRAFT ADR generated by the [Design Decision Gate](https://github.com/github/gh-aw/actions/runs/24424691653) workflow. The PR author must review, complete, and finalize this document before the PR can merge.*
