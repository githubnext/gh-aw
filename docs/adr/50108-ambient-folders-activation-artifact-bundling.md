# ADR-50108: Ambient Folders — Frontmatter-Declared Folder Bundling in Activation Artifact

**Date**: 2026-08-03
**Status**: Draft
**Deciders**: Unknown

---

### Context

Shared workflows that run activation steps (e.g., Squad CLI initialization) produce workspace folders (`.squad/`, `.github/agents/`) that must be present inside the agent container at runtime. Before this change, the Squad shared workflow uploaded those folders as a separate `squad-state` artifact and each consuming workflow's agent job downloaded it independently. This meant every workflow author who imported Squad had to manage an extra download step, the pattern was not composable, and any workflow that also performed a custom `actions/checkout` would clobber the restored files because the restore happened before the checkout.

### Decision

We will introduce a new `on.ambient-folders` frontmatter field in gh-aw workflow markdown files. Shared workflows declare the workspace-relative folder paths they produce; the compiler merges those declarations across all imports (deduplicating), adds the declared folders to the activation sparse-checkout, stages them into `/tmp/gh-aw/ambient-folders/` immediately before the activation artifact is uploaded, includes that staging directory in the artifact path list, and emits a restore step in the agent job after the last custom checkout (so multi-checkout workflows do not lose ambient content). Workflows whose `on:` block contains only import-safe keys (including `ambient-folders`) are classified as shared components rather than standalone workflows.

### Alternatives Considered

#### Alternative 1: Per-workflow custom artifact upload/download steps (status quo)

Each shared workflow that produces agent-facing folders manages its own named artifact (e.g., `squad-state`). Consuming workflows add a `download-artifact` step explicitly.

Not chosen because: the pattern is not composable — every workflow author must remember to add the download step, the step must appear after any custom checkout to avoid being wiped, and each shared workflow occupies a separate artifact slot. Generalizing this to multiple shared workflows produces combinatorial download steps.

#### Alternative 2: Automatically include a fixed set of well-known folders

Hard-code a list of folders (`.squad`, `.github/agents`, etc.) that are always included in the activation artifact regardless of frontmatter.

Not chosen because: it couples the platform to specific tooling choices, does not scale to third-party shared workflows producing different folder structures, and adds unnecessary artifact size for workflows that do not use those tools.

### Consequences

#### Positive
- Shared workflows can declare their folder dependencies once in frontmatter; the compiler handles staging and restore automatically, removing the need for per-consumer download steps.
- The restore step is injected after the last custom checkout, which prevents multi-checkout workflows from clobbering ambient content.
- The merge strategy (union/deduplicated) means multiple shared workflows each declaring overlapping folders still produce a single coherent restore.
- The field is validated by JSON Schema with path-traversal protections (no `..`, no absolute paths), keeping the attack surface small.

#### Negative
- Adds a new frontmatter field that must be validated, documented, kept in sync across the parser, compiler, schema, and runtime; any future rename or removal is a breaking change for consuming workflows.
- The shared-workflow classification logic now depends on an `on:` field value inspection (`IsImportSafeSharedWorkflowOn`), which increases coupling between the parser and compiler orchestration.

#### Neutral
- Workflows with `on: ambient-folders: [...]` and no trigger event are now classified as shared components, consistent with the existing behaviour for other import-safe `on:` keys (`skip-if-match`, `github-token`, etc.).
- The staging script uses `cp -a` (archive copy) and silently skips missing source folders, so workflows that conditionally produce folders do not fail the activation job.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
