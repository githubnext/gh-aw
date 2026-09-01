# ADR-55475: Schema-Validated Workflow Frontmatter Edit Command

**Date**: 2026-08-24
**Status**: Draft
**Deciders**: pelikhan, copilot-swe-agent

---

### Context

Workflow definitions in this project are Markdown files with YAML frontmatter that controls execution parameters (e.g., `max-turns`, `model`, `on.schedule`, `imports`). To change any of these parameters today, users must manually edit the raw Markdown file and then separately run `gh aw compile` to regenerate the corresponding `.lock.yml` file. Manual editing bypasses schema validation entirely, meaning invalid frontmatter values are only detected at compile time—after the file has already been written to disk. The need for an explicit, validated, and atomic path for mutating workflow frontmatter is the driving problem this PR addresses.

### Decision

We will add a new `gh aw edit` CLI command (`pkg/cli/edit_command.go`) that provides schema-validated, programmatic mutation of workflow frontmatter with automatic recompilation. The command accepts typed flag-based mutations (`--set`, `--unset`, `--add`, `--remove`, `--schedule`, `--add-import`, `--add-skill`) and a positional `path: value` shorthand. It validates the resulting frontmatter against the workflow schema before writing and immediately recompiles the `.lock.yml`; on compilation failure it rolls back both files to their prior state.

### Alternatives Considered

#### Alternative 1: Direct file editing + manual compile

Users continue to edit YAML frontmatter by hand and run `gh aw compile` separately. This requires no new code but provides no schema validation at edit time, allows invalid frontmatter to be committed before compile, and leaves the lock file in an inconsistent state when a user forgets to recompile. It was rejected because it does not address the safety problem.

#### Alternative 2: Extend `gh aw update` with frontmatter mutation flags

Add mutation flags to the existing `update` command. The `update` command is semantically about syncing a workflow from a `source:` declaration. Mixing configuration mutation into the same command would create a confusing API where the same command both fetches external content and edits local state. It was rejected to preserve the clarity of the existing command model.

### Consequences

#### Positive
- Schema validation runs before any bytes are written to disk, preventing invalid frontmatter values from ever reaching the repository.
- Automatic recompilation keeps `.lock.yml` atomically in sync with the edited workflow file.
- Transactional rollback: if recompilation fails, both the workflow source file and the lock file are restored to their pre-edit state.
- Fuzzy schedule expressions (`daily`, `every 6h`, `daily on weekdays`, etc.) are validated with the shared schedule parser at edit time, providing a user-friendly schedule API.
- Source-managed workflows can be edited locally while preserving their managed `source:` provenance; by default, later `gh aw update` runs merge those local edits with upstream changes.

#### Negative
- The managed top-level `source:` field is not user-editable on source-managed workflows; changing provenance still belongs to `gh aw update` or the upstream source.
- `gh aw update --no-merge` intentionally overwrites local edits instead of preserving them.
- YAML re-serialization via `go-yaml` may alter key ordering, indentation, comments, or whitespace in the frontmatter beyond the intended change, potentially producing noisy diffs. Edits that change nothing are detected and skip writing entirely, so no-op edits never rewrite a workflow.

#### Neutral
- The command is labelled "Experimental" in its `Short` and `Long` help text, signalling that its interface may change before stabilization.
- The command is registered in the `setup` group alongside `add`, `update`, and `remove`, maintaining consistency with the existing command taxonomy.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
