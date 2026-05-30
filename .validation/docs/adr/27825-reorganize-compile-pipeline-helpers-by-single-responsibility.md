# ADR-27825: Reorganize Compile Pipeline Helpers by Single-Responsibility Concern

**Date**: 2026-04-22
**Status**: Draft
**Deciders**: pelikhan

---

## Part 1 — Narrative (Human-Friendly)

### Context

The `pkg/cli/compile_*.go` package grew organically as new compilation features were added. By the time of this decision, three structural problems had accumulated: (1) `CompilationStats` and `WorkflowFailure` types were defined in `compile_config.go` while their operations (`trackWorkflowFailure`, `printCompilationSummary`) lived in `compile_file_operations.go`, splitting a cohesive abstraction across two files; (2) `compile_post_processing.go` bundled manifest/workflow generation wrappers alongside infrastructure side-effects such as updating `.gitattributes` and persisting the action cache; (3) each directory-scanning tool (poutine, runner-guard) had its own private pass-through wrapper that duplicated the same strict/non-strict error-handling pattern verbatim.

### Decision

We will reorganize the compile pipeline helpers so that each file owns exactly one concern: types and operations for a concept live together, infrastructure side-effects are isolated from workflow-generation logic, and repeated patterns are unified into a single shared helper. Concretely: `compile_stats.go` owns all stats types and operations; `compile_infrastructure.go` owns `.gitattributes` and action-cache side-effects; `compile_file_operations.go` gains the path helper `getAbsoluteWorkflowDir`; and `runBatchDirectoryTool` replaces four near-identical pass-through wrappers.

### Alternatives Considered

#### Alternative 1: Keep the existing file layout and add documentation

Rather than moving code, add package-level comments and godoc cross-references to explain where things live. This is low-risk and requires no diff. It was rejected because documentation describes structure but does not fix it — the root cause is that related code is in different files, which forces developers to open multiple files to understand or modify a single abstraction. Documentation decays as code changes; co-location is self-enforcing.

#### Alternative 2: Extract compile infrastructure into a separate internal package

Move the infrastructure helpers (`updateGitAttributes`, `saveActionCache`) into a new `pkg/cli/compileinfra` sub-package. This maximises isolation and makes the dependency graph explicit. It was not chosen because the helpers are small, tightly coupled to the compile pipeline's types, and a new package would add import indirection without meaningful boundary enforcement — the sub-package and its caller would still be in the same binary and changed together. A file boundary is sufficient for this level of separation.

### Consequences

#### Positive
- Types and their operations now live in the same file, making navigation predictable (open `compile_stats.go` to understand compilation statistics end-to-end).
- The batch wrapper duplication is eliminated: four near-identical functions collapse into one parameterised helper, reducing the surface for drift bugs.
- `compile_post_processing.go` becomes focused on manifest and workflow generation, making its purpose clear from its name.
- The strict/non-strict error-handling policy for directory tools is now tested in one place (`compile_batch_operations_test.go`).

#### Negative
- The refactor is a non-trivial file churn (eight files changed) that creates a large, hard-to-review diff with no functional change. Reviewers must verify that no logic was lost during the moves.
- Any external tools or scripts that relied on the unexported symbol locations (e.g., `grep`-based tooling or generated stubs) may need updating.

#### Neutral
- `compile_pipeline.go` call sites are updated to reference the unified helpers directly; callers are functionally equivalent but read slightly differently.
- The directory structure of `pkg/cli/` grows by two files (`compile_infrastructure.go`, `compile_stats.go`), which is consistent with the existing naming convention.

---

## Part 2 — Normative Specification (RFC 2119)

> The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this section are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### File Ownership

1. The `compile_stats.go` file **MUST** contain the `CompilationStats` and `WorkflowFailure` type definitions and all functions that operate on those types (`trackWorkflowFailure`, `printCompilationSummary`, `collectWorkflowStatisticsWrapper`).
2. The `compile_infrastructure.go` file **MUST** contain infrastructure side-effect helpers (`updateGitAttributes`, `saveActionCache`) and **MUST NOT** contain workflow-generation or manifest logic.
3. The `compile_post_processing.go` file **MUST NOT** contain infrastructure side-effect helpers; it **MUST** be limited to manifest and workflow generation wrappers.
4. Path and file system helpers (functions that compute or manipulate file paths) **SHOULD** reside in `compile_file_operations.go`.

### Batch Tool Helpers

1. New directory-scanning batch tool wrappers **MUST** delegate to `runBatchDirectoryTool` rather than duplicating the strict/non-strict error-handling pattern inline.
2. New lock-file batch tool wrappers **MUST** delegate to `runBatchLockFileTool` rather than duplicating the empty-list guard and logging pattern inline.
3. Pass-through wrapper functions that add no logic beyond calling a single public function **MUST NOT** be introduced.

### Conformance

An implementation is considered conformant with this ADR if it satisfies all **MUST** and **MUST NOT** requirements above. Failure to meet any **MUST** or **MUST NOT** requirement constitutes non-conformance.

---

*This is a DRAFT ADR generated by the [Design Decision Gate](https://github.com/github/gh-aw/actions/runs/24780779719) workflow. The PR author must review, complete, and finalize this document before the PR can merge.*
