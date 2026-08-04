# ADR-50108: Ambient Folders — Frontmatter-Declared Folder Bundling in Activation Artifact

**Date**: 2026-08-03
**Status**: Draft
**Deciders**: Unknown

---

### Context

Shared workflows that run activation steps (e.g., Squad CLI initialization) produce workspace folders (`.squad/`, `.github/agents/`) that must be present inside the agent container at runtime. Before this change, the Squad shared workflow uploaded those folders as a separate `squad-state` artifact and each consuming workflow's agent job downloaded it independently. This meant every workflow author who imported Squad had to manage an extra download step, the pattern was not composable, and any workflow that also performed a custom `actions/checkout` would clobber the restored files because the restore happened before the checkout.

### Decision

We will introduce a top-level `ambient-folders` frontmatter field in gh-aw workflow markdown files. Shared workflows declare the workspace-relative folder paths they produce; the compiler merges those declarations across all imports (deduplicating), adds the declared folders to the activation sparse-checkout, and adds them directly to the activation artifact path list. The artifact is extracted at its root in downstream jobs, preserving both the generated `/tmp/gh-aw` files and the declared workspace folders. Workflows with no trigger event remain shared components.

### Alternatives Considered

#### Alternative 1: Per-workflow custom artifact upload/download steps (status quo)

Each shared workflow that produces agent-facing folders manages its own named artifact (e.g., `squad-state`). Consuming workflows add a `download-artifact` step explicitly.

Not chosen because: the pattern is not composable — every workflow author must remember to add the download step, the step must appear after any custom checkout to avoid being wiped, and each shared workflow occupies a separate artifact slot. Generalizing this to multiple shared workflows produces combinatorial download steps.

#### Alternative 2: Automatically include a fixed set of well-known folders

Hard-code a list of folders (`.squad`, `.github/agents`, etc.) that are always included in the activation artifact regardless of frontmatter.

Not chosen because: it couples the platform to specific tooling choices, does not scale to third-party shared workflows producing different folder structures, and adds unnecessary artifact size for workflows that do not use those tools.

### Consequences

#### Positive
- Shared workflows can declare their folder dependencies once in frontmatter; the compiler packages them with the standard activation artifact, removing the need for per-consumer download steps.
- The merge strategy (union/deduplicated) means multiple shared workflows each declaring overlapping folders still produce a single coherent restore.
- The field is validated by JSON Schema with path-traversal protections (no `..`, no absolute paths), keeping the attack surface small.

#### Negative
- Adds a new frontmatter field that must be validated, documented, kept in sync across the parser, compiler, schema, and runtime; any future rename or removal is a breaking change for consuming workflows.
- The shared-workflow classification logic now depends on an `on:` field value inspection (`IsImportSafeSharedWorkflowOn`), which increases coupling between the parser and compiler orchestration.

#### Neutral
- Workflows with `ambient-folders: [...]` and no trigger event are classified as shared components.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
