# ADR-51090: Add `gh aw import` Command for Workflow Frontmatter Injection

**Date**: 2026-08-07
**Status**: Draft
**Deciders**: pelikhan

---

### Context

Agentic workflow `.md` files use a YAML frontmatter `imports:` list to reference shared workflow fragments. Until now, adding an import required manually editing the YAML frontmatter, which is error-prone, non-automatable, and duplicates effort when the same import must be added across many workflow files. The gh-aw CLI already has a parser pipeline (`ExtractFrontmatterFromContent` + `reconstructWorkflowFileFromMap`) that can round-trip frontmatter safely. A dedicated `import` subcommand can expose idempotent, scriptable frontmatter mutation without building a general-purpose YAML editor.

### Decision

We will add a new `gh aw import <workflow> <import-path>` CLI subcommand registered under the `setup` command group. The command reads the target workflow file, delegates frontmatter manipulation to the existing `parser.ExtractFrontmatterFromContent` + `reconstructWorkflowFileFromMap` round-trip pipeline, appends the requested import path if not already present, and writes the result back in place while preserving the original file permissions and trailing-newline behaviour. The operation is idempotent: a duplicate import is detected and silently skipped.

### Alternatives Considered

#### Alternative 1: Direct YAML library manipulation

Mutate `imports:` by parsing the file with a standalone YAML library (e.g., `gopkg.in/yaml.v3`) rather than reusing the existing frontmatter round-trip helpers. This would avoid introducing a dependency on internal parser internals. However, `yaml.v3` serialisation may reorder keys, strip comments, and normalise whitespace in ways that differ from the existing pipeline, producing noisy diffs for users. Reusing the established round-trip path keeps output consistent with every other command that writes frontmatter.

#### Alternative 2: Add `--import` flag to the existing `gh aw add` command

Extend the existing `add` subcommand with an `--import` flag rather than introducing a new top-level command. This reduces the surface area of the CLI. However, `add` is semantically about creating new workflows, not mutating existing ones; mixing creation and mutation in one command would blur the command's purpose and make the `--import` flag unavailable when a workflow already exists. A dedicated `import` subcommand follows the single-responsibility principle already established by `gh aw add`, `gh aw update`, `gh aw remove`, etc.

### Consequences

#### Positive
- Users and automation can inject shared import references into workflow files without hand-editing YAML, reducing error surface and enabling scripted bulk updates.
- The implementation reuses the existing frontmatter round-trip pipeline, ensuring output format consistency with all other commands that write workflow files.
- The idempotent design makes the command safe to call unconditionally in CI scripts.
- Original file permissions are preserved on write, preventing accidental permission downgrades.

#### Negative
- The `imports:` field must be a YAML list; workflows where the field holds a scalar or map value will receive an error rather than a graceful coercion, requiring the user to fix the file manually.
- Frontmatter key ordering after the round-trip is determined by the parser's internal serialisation logic; if that logic does not guarantee stable key order, repeated invocations may produce unnecessary diff noise for non-`imports:` keys.
- Adding a new top-level `setup` command grows the CLI's command surface; discoverability depends on documentation and help text.

#### Neutral
- The command accepts both a workflow name (resolved via `resolveWorkflowFile`) and a direct file path, following the same resolution convention used by other commands in the CLI.
- Tests cover the five key scenarios (append to existing list, create from scratch, duplicate skip, no-frontmatter error, file-level idempotency), consistent with the test style of sibling commands.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
