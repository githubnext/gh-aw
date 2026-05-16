# ADR-32489: Canonical Descriptor Table for Safe-Output Dispatch

**Date**: 2026-05-15
**Status**: Draft
**Deciders**: pelikhan, Copilot

---

## Part 1 — Narrative (Human-Friendly)

### Context

Safe-output handler behavior in `pkg/workflow` was spread across several parallel switch/if tables: `hasSafeOutputType` performed key-to-field checks, `SafeOutputsConfigFromKeys` constructed concrete handler config structs via a large switch, `mergeSafeOutputConfig` enumerated per-handler merge logic, and `ComputePermissionsForSafeOutputs` repeated per-handler permission derivation branches. Every new handler required a coordinated multi-file edit, and the parallel tables could drift independently — most notably in alias handling, permission derivation, and merge semantics — without any compile-time check catching the divergence. The result was high cognitive load when adding or auditing a handler, and a real risk of subtle bugs when one of the tables fell out of sync.

### Decision

We will introduce a single canonical `safeOutputHandlers` descriptor list in `pkg/workflow/safe_output_handlers.go`, where each entry carries the handler's `Key`, optional `Aliases`, `StructField` name on `SafeOutputsConfig`, `ToolName`, `Builtin` flag, a `NewConfig` constructor, and a `PermissionBuilder`. The dispatch sites (`hasSafeOutputType`, `SafeOutputsConfigFromKeys`, `mergeSafeOutputConfig`, `ComputePermissionsForSafeOutputs`, and the `safeOutputFieldMapping`) are routed through this descriptor table using shared lookup and reflection helpers, so adding or modifying a handler becomes a single-entry change in the descriptor list. Special-case semantics that don't fit the generic shape (e.g., the protected-files set-merge, auto-default override behavior, and the nil-check fast paths on `hasAnySafeOutputEnabled` / `hasNonBuiltinSafeOutputsEnabled`) are preserved explicitly rather than forced into the generic path.

### Alternatives Considered

#### Alternative 1: Keep the Parallel Switch/If Tables

The simplest option is to leave the existing parallel dispatch tables in place and rely on code review to keep them in sync. This was rejected because the number of handlers and the number of dispatch concerns (existence check, construction, merge, permission derivation, tool name mapping, alias resolution) make manual synchronization fragile, and the parallel tables provide no compile-time guarantee that all dispatch concerns have been considered for a new handler.

#### Alternative 2: Code-Generate the Dispatch Tables from a Schema

The descriptor information could have been expressed as a YAML/JSON schema and code-generated into Go at build time, eliminating reflection entirely. This was rejected as overkill for the current handler count: it introduces a build-time generator, a schema-to-Go pipeline, and review overhead for generated files, in exchange for marginal runtime gains versus a runtime descriptor list with localized reflection.

#### Alternative 3: Force All Handlers Through a Single Generic Path

The descriptor could have been pushed further so that *every* dispatch concern — including special cases like protected-files set merging and auto-default overrides — flowed through a single generic mechanism via extra descriptor fields or strategy hooks. This was rejected because the special-case semantics are genuinely irregular; encoding them as descriptor extensions would have moved the complexity into the descriptor type rather than removing it, and would have obscured the intent of those special cases.

### Consequences

#### Positive
- Adding a new safe-output handler becomes a single-entry change in `safeOutputHandlers` plus the corresponding `SafeOutputsConfig` field, instead of edits to four or five parallel tables.
- Alias resolution, permission derivation, and tool-name mapping are guaranteed consistent across dispatch sites because they read from the same descriptor.
- `safeOutputFieldMapping` is derived from the descriptor table rather than maintained as a separate constant, removing one drift surface.

#### Negative
- Generic dispatch paths now use reflection (`reflect`) to read and write `SafeOutputsConfig` fields, introducing a small runtime cost and a class of errors (typo'd `StructField`) that would previously have been caught at compile time.
- The split between "generic descriptor-driven path" and "preserved special-case semantics" creates a second mental model — contributors must know which handlers go through the descriptor and which retain bespoke logic in `mergeSafeOutputConfig`.
- The descriptor list is now a centralized hotspot: any change to the descriptor type ripples across all handlers and all dispatch sites.

#### Neutral
- Hot-path predicates (`hasAnySafeOutputEnabled`, `hasNonBuiltinSafeOutputsEnabled`) intentionally remain as direct nil-check cascades rather than reflection loops, so compile-time performance is unchanged on the most frequently executed checks.
- `TestHasSafeOutputTypeNewKeys` is updated to validate legacy key parity against the descriptor table and to assert constructor/field type compatibility, providing a guardrail against future drift.
- The descriptor approach is internal to `pkg/workflow`; no public API surface changes.

---

## Part 2 — Normative Specification (RFC 2119)

> The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this section are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### Descriptor Source of Truth

1. The `safeOutputHandlers` list in `pkg/workflow/safe_output_handlers.go` **MUST** be the single source of truth for the per-handler tuple of `Key`, `Aliases`, `StructField`, `ToolName`, `Builtin`, `NewConfig`, and `PermissionBuilder`.
2. New safe-output handlers **MUST** be registered by adding an entry to `safeOutputHandlers` and **MUST NOT** be introduced solely by editing dispatch-site switches.
3. Handler key strings and aliases **MUST NOT** be hardcoded in dispatch sites in a way that bypasses descriptor lookup; dispatch sites **SHALL** resolve keys via the shared descriptor lookup helper.

### Dispatch-Site Routing

1. `hasSafeOutputType` **MUST** resolve handler existence by descriptor key (including aliases) and **MUST** verify the corresponding `StructField` is set via the shared reflection helper.
2. `SafeOutputsConfigFromKeys` **MUST** construct handler config values via the descriptor's `NewConfig` constructor and **MUST NOT** reintroduce a parallel per-handler switch for construction.
3. `mergeSafeOutputConfig` **MUST** perform descriptor-driven field merge for standard handlers, and **MAY** retain explicit special-case branches for handlers whose merge semantics are not generic (e.g., protected-files set merging, auto-default override behavior).
4. `ComputePermissionsForSafeOutputs` **MUST** iterate descriptor `PermissionBuilder` functions to derive handler-managed permissions, and **MUST NOT** duplicate handler-specific permission logic outside the descriptor table.
5. `safeOutputFieldMapping` **MUST** be derived from descriptor metadata rather than maintained as an independent constant.

### Hot-Path Preservation

1. `hasAnySafeOutputEnabled` and `hasNonBuiltinSafeOutputsEnabled` **MUST** remain direct nil-check cascades over `SafeOutputsConfig` fields and **MUST NOT** be rewritten to perform reflection or descriptor iteration on the compile hot path.
2. Implementations **MAY** generate or maintain these nil-check cascades from the descriptor list at build time, provided the runtime cost is equivalent to the original direct checks.

### Parity Guardrails

1. The descriptor table **MUST** be covered by a parity test (currently `TestHasSafeOutputTypeNewKeys`) that asserts: every legacy handler key resolves via the descriptor table, every descriptor `StructField` exists on `SafeOutputsConfig`, and every `NewConfig` constructor returns a type assignable to the corresponding `StructField`.
2. Removing or weakening the parity test **MUST NOT** be done without an accompanying ADR update.

### Conformance

An implementation is considered conformant with this ADR if it satisfies all **MUST** and **MUST NOT** requirements above. Failure to meet any **MUST** or **MUST NOT** requirement constitutes non-conformance. The primary conformance checks are: (a) `safeOutputHandlers` is the sole place where the descriptor tuple is declared, (b) the four named dispatch sites route through the descriptor table as specified, (c) `hasAnySafeOutputEnabled` / `hasNonBuiltinSafeOutputsEnabled` retain direct nil-check cascades, and (d) the parity test continues to validate descriptor/field/constructor coherence.

---

*This is a DRAFT ADR generated by the [Design Decision Gate](https://github.com/github/gh-aw/actions/runs/25944925610) workflow. The PR author must review, complete, and finalize this document before the PR can merge.*
