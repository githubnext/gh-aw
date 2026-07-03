# ADR-43071: Split compiler_types.go Grab-Bag into Semantically-Cohesive Files

**Date**: 2026-07-03
**Status**: Draft
**Deciders**: Unknown

---

### Context

`pkg/workflow/compiler_types.go` had grown to 835 lines and mixed three unrelated concerns in a single file: the `Compiler` struct and its constructors, the `WorkflowData` domain struct (the central data bag threaded through compilation), and five safe-output configuration types (`BaseSafeOutputConfig`, `SafeOutputsConfig`, `SafeOutputMessagesConfig`, `MentionsConfig`, `SecretMaskingConfig`) that logically belong alongside the parsing logic that already lived in `safe_outputs_config.go`. A grab-bag file of this size makes it hard to locate any individual concept and creates a misleading association between the `Compiler` type and unrelated domain structs.

### Decision

We will split `compiler_types.go` into three semantically-focused files within the same `pkg/workflow` package: `workflow_data.go` (owns `WorkflowData`, its `PinContext()` method, and the three `SkipIf*Config` types it references directly), `safe_outputs_config.go` (prepended with `BaseSafeOutputConfig`, `SafeOutputsConfig`, `SafeOutputMessagesConfig`, `MentionsConfig`, and `SecretMaskingConfig` to co-locate them with their extraction logic), and a reduced `compiler_types.go` (retains only `Compiler`, `CompilerOption`/`With*` constructors, `FileCreationTracker`, `NewCompiler`, and accessor methods). This is a pure mechanical split with no logic changes.

### Alternatives Considered

#### Alternative 1: Add Section Headers and Comments Within compiler_types.go

Add `// === WorkflowData ===`, `// === SafeOutputsConfig ===`, and similar comment banners to demarcate sections inside the existing file without moving any code. This avoids file proliferation and keeps all types in one place for grep-ability.

Rejected because comment banners do not enforce cohesion — the file would continue to grow freely across all sections, and IDE navigation by file name would remain misleading. The file name `compiler_types.go` gives no signal that it contains the domain's primary data struct.

#### Alternative 2: Extract Types into a Dedicated sub-package (e.g., pkg/workflow/types)

Move `WorkflowData` and the config types into a separate `types` sub-package, breaking the circular dependency risk and making the boundary explicit via package-level import.

Rejected for this PR because it would require updating every import site across the compiler, introduce a new package boundary decision, and risk circular import issues given how `WorkflowData` and `Compiler` reference shared types. The mechanical split within the same package achieves the readability goal at near-zero risk and can always be followed by a deeper extraction later.

### Consequences

#### Positive
- `workflow_data.go` now has a clear single-concept identity: it is the home of `WorkflowData`, the central domain struct, making it easy to locate for anyone unfamiliar with the codebase.
- Safe-output config types live in `safe_outputs_config.go` directly above the parsing/extraction logic that consumes them, improving colocation and reducing cross-file jumps during development.
- `compiler_types.go` is reduced to only `Compiler` and its constructors, so its name now accurately describes its contents.

#### Negative
- Reviewers and code-search users must now look across three files instead of one to understand the full type landscape of `pkg/workflow`; the split increases file count without reducing overall line count.
- The `actionpins` import responsibility moves from `compiler_types.go` to `workflow_data.go`; contributors who had memorized the old dependency graph need to update their mental model.

#### Neutral
- No logic changes are introduced — the split carries zero behavioral risk but also delivers no functional improvement; all value is in maintainability.
- Any future extraction of `WorkflowData` into its own sub-package is now easier because it already lives in an isolated file with its direct dependencies co-located.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
