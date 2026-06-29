# ADR-42361: Add Dry-Run Mode to update and upgrade Commands

**Date**: 2026-06-29
**Status**: Draft
**Deciders**: Unknown (PR author: copilot-swe-agent / pelikhan)

---

### Context

The `gh aw update` and `gh aw upgrade` commands modify files on disk (workflow Markdown, actions-lock.json, compiled lock files) with no way for the user to preview what changes would be made before applying them. This creates operational risk: users running maintenance commands in automated pipelines or unfamiliar repositories cannot safely verify the scope of changes first. The MCP tools exposed to AI agents share this limitation, meaning an AI assistant invoking `update` or `upgrade` via MCP would immediately apply changes. A preview mode is a standard CLI safety convention and is especially important when tools are invoked non-interactively.

### Decision

We will add a `--dry-run` flag to both `gh aw update` and `gh aw upgrade`. In dry-run mode, the commands skip all file-write operations and instead print `(dry run) Would …` messages describing each step that would have run. The MCP tools for `update` and `upgrade` will expose a `dry_run` field that defaults to `true` via `AddSchemaDefault`, so AI-initiated calls are safe by default. The dashboard Maintenance tab will expose a "Dry run" checkbox (checked by default) that appends `--dry-run` to the commands it builds.

### Alternatives Considered

#### Alternative 1: Interactive Confirmation Prompt

Display a diff or summary and require the user to type `y` before applying changes. This is the UX pattern used by tools like `terraform apply`. It was not chosen because it is not automation-friendly (breaks CI and AI-agent workflows unless a `--yes` flag is also added) and because this project already uses `--yes` for other operations, making a flag-based approach more consistent.

#### Alternative 2: Dedicated Preview Subcommands (`update preview`, `upgrade preview`)

Introduce separate subcommands whose only purpose is to simulate what the main commands would do. This is more discoverable via `--help` but would double the surface area of the CLI, require duplicating command parsing and option logic, and would be less familiar than the `--dry-run` convention widely used by tools such as `rsync`, `apt`, and `helm`.

### Consequences

#### Positive
- Users and AI agents can safely preview the scope of `update` and `upgrade` before any files are touched, reducing the risk of unintended changes.
- MCP tools default to `dry_run: true`, providing a safe-by-default experience for AI-driven maintenance operations.
- The dashboard "Dry run" checkbox (enabled by default) gives visual confirmation that commands are in preview mode via a `Label--attention` badge.

#### Negative
- The dry-run output for `upgrade` is a static list of "Would do X" lines rather than an actual simulation; it does not query remote sources to determine whether a newer extension version or dispatcher skill actually exists, so the preview may not accurately reflect what a real run would do.
- The dry-run logic in `update_command.go` calls `UpdateWorkflows` before the early-exit guard, meaning workflow-resolution network requests still execute even in dry-run mode; only the final file writes are skipped.

#### Neutral
- Both commands now return an error from `registerUpdateTool` / `registerUpgradeTool` (previously `void`), so the MCP server start-up path now propagates schema-generation failures explicitly.
- The new `upgrade` MCP tool surface mirrors the CLI flag set (`--dry-run`, `--no-fix`, `--no-compile`) but omits `--no-actions` and `--pre-releases`; callers needing those flags must use the CLI directly.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
