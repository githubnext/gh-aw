# ADR-26113: `env` Field Support in Shared Workflow Imports with Conflict-Error Semantics

**Date**: 2026-04-14
**Status**: Draft
**Deciders**: pelikhan, Copilot

---

## Part 1 — Narrative (Human-Friendly)

### Context

The GitHub Agentic Workflows (gh-aw) compiler supports shared workflow files that are imported by main workflows. These shared files allow teams to extract reusable steps, tools, permissions, and other configuration into composable fragments. Prior to this change, the `env` field was explicitly listed in `SharedWorkflowForbiddenFields`, meaning any `env:` block in a shared import was silently dropped with a warning. This prevented shared workflows from declaring workflow-level environment variables (e.g., `TARGET_REPOSITORY`, `SHARED_CONFIG`), forcing workflow authors to duplicate those declarations in every consuming main workflow — violating the DRY principle and making shared workflows less self-contained.

### Decision

We will lift the `env` restriction from shared workflow imports and implement a three-tier precedence model: (1) the main workflow's `env` vars always win over any imported value, (2) import-import conflicts on the same variable name are a hard compilation error with an actionable message, and (3) when no conflict exists, imported env vars are merged into the compiled lock file. Additionally, the lock file header will list all env vars with source attribution (`(main workflow)` or the import file path) to aid auditability. This model makes the main workflow the single authoritative override point while enforcing explicit conflict resolution between imports.

### Alternatives Considered

#### Alternative 1: Last-Write-Wins Between Imports

Import order could determine precedence when two shared files define the same env var (breadth-first topological order of the import graph). This would avoid surfacing an error but would make behavior silently dependent on the import declaration order, creating subtle, hard-to-debug bugs when shared files are reordered or when a new shared import is added that happens to define the same variable.

#### Alternative 2: First-Write-Wins Between Imports

Similar to last-write-wins but more predictable in practice (the first declared import "owns" the variable). Still suffers from the same silent ordering dependency; teams would have no visible signal that two imports conflict, leading to unexpected behavior when the import list is edited.

#### Alternative 3: Keep `env` Forbidden in Shared Imports (Status Quo)

Maintaining the current restriction is simple and avoids the complexity of conflict resolution entirely. However, it forces every consumer of a shared workflow to manually redeclare env vars that logically belong to the shared concern, making shared workflows less reusable and increasing the risk of drift between copies.

#### Alternative 4: Allow All Overrides Without Source Attribution

Merging could be done without tracking which import contributed which variable. This simplifies the implementation but sacrifices transparency: when a compiled workflow contains an unexpected env var value, there is no way to determine where it came from without re-reading every imported file.

### Consequences

#### Positive
- Shared workflow files are more self-contained; env vars that logically belong to a reusable concern can be co-located with the rest of the shared configuration.
- Import-import conflicts fail loudly at compile time with a clear, actionable error message instead of silently producing incorrect behavior.
- Lock file headers now list all env vars with source attribution, improving auditability of compiled workflows.
- The main workflow retains unconditional override authority, preserving the "main workflow is the source of truth" invariant already established for other merged fields.

#### Negative
- Workflow authors who accidentally duplicated the same env var across two shared imports will now get a compilation error they must resolve before their workflow compiles.
- The `importAccumulator` struct gains two new fields (`envBuilder`, `envSources`) and the `ImportsResult` and `WorkflowData` types each gain new fields, increasing structural complexity.
- The lock file header grows by the number of env vars, which may slightly increase generated file size.

#### Neutral
- The env merging approach (newline-separated JSON objects) follows the same internal serialisation convention already used for other merged fields (e.g., `MergedJobs`).
- Existing workflows without `env:` in their shared imports are entirely unaffected; no migration is required.
- The `include_processor.go` suppression-list update (adding `"env"` to valid non-workflow frontmatter fields) removes a spurious warning that users would have seen if they had `env:` in an included file.

---

## Part 2 — Normative Specification (RFC 2119)

> The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this section are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### Env Field Allowance

1. Shared workflow imports **MUST** be permitted to declare an `env:` field; the compiler **MUST NOT** treat `env` as a forbidden field in shared workflow files.
2. The `env` key **MUST NOT** appear in `SharedWorkflowForbiddenFields`.
3. The `env` field **MUST** be listed as a valid non-workflow frontmatter field in the include processor to suppress spurious unknown-field warnings.

### Env Merge Precedence

1. Env vars declared in the main workflow **MUST** take precedence over any env var from an imported file with the same key; the compiled output **MUST** use the main workflow's value.
2. When two different imported files both declare the same env var key, the compiler **MUST** return a compilation error before producing any output.
3. The compilation error for import-import conflicts **MUST** identify the conflicting variable name and both import file paths, and **SHOULD** include guidance on how to resolve the conflict (e.g., move the variable to the main workflow or remove it from one import).
4. Imported env vars that do not conflict with each other and are not overridden by the main workflow **MUST** be merged into the compiled workflow's `env:` block.

### Source Attribution

1. The compiled lock file header **MUST** include an `# Env variables:` section listing every env var that is present in the merged env block.
2. Each entry in the env variables section **MUST** be annotated with its source: `(main workflow)` if the variable originates from the main workflow file, or the import file path (relative to the repo root) if it originates from a shared import.
3. Keys in the env variables header section **MUST** be emitted in sorted (lexicographic ascending) order for deterministic output.

### Data Model

1. `ImportsResult` **MUST** expose a `MergedEnv string` field containing the accumulated env var JSON from all imports, and a `MergedEnvSources map[string]string` field mapping each env var key to its originating import path.
2. `WorkflowData` **MUST** expose an `EnvSources map[string]string` field mapping each env var key to its final source label (`(main workflow)` or import path) for use in lock file header generation.
3. Implementations **MUST NOT** store the merged env blob in any format other than newline-separated JSON objects, consistent with the existing convention for other multi-import merged fields.

### Conformance

An implementation is considered conformant with this ADR if it satisfies all **MUST** and **MUST NOT** requirements above. Specifically: the `env` field is accepted in shared imports without warning, import-import conflicts cause a hard compilation error with an informative message, main-workflow env vars always override imported values, and the compiled lock file header lists all env vars with source attribution in sorted order. Failure to meet any **MUST** or **MUST NOT** requirement constitutes non-conformance.

---

*This is a DRAFT ADR generated by the [Design Decision Gate](https://github.com/github/gh-aw/actions/runs/24374798953) workflow. The PR author must review, complete, and finalize this document before the PR can merge.*
