# ADR-53659: Add Missing Safe-Output Config Fields to JSON Schema

**Date**: 2026-08-18
**Status**: Draft
**Deciders**: Unknown

---

### Context

The workflow JSON schema at `pkg/parser/schemas/main_workflow_schema.json` defines `safe-outputs.*` sub-schemas that control what configuration fields workflow authors may set. The Go implementation in `pkg/workflow/` had already grown 13 YAML-tagged config fields across six safe-output types (`add-comment`, `assign-milestone`, `comment-memory`, `create-issue`, `create-pull-request`, `push-to-pull-request-branch`, `threat-detection`) that were absent from the schema. The daily conformance checker (IMP-004 / `check_safe_output_config_schema_coverage`) surfaces this as a spec violation: because the schema does not set `additionalProperties: false` for these sections, there is no hard validation failure at parse time, but editor autocomplete, type checking, and docs-generation tooling are blind to the missing fields. Three of the 13 flagged fields (`call-workflow.workflow_files`, `dispatch-workflow.workflow_files`, `dispatch-workflow.aw_context_workflows`) are compiler-populated internals and must be excluded from the user-facing schema rather than documented in it.

### Decision

We will add all 10 user-facing safe-output configuration fields to `main_workflow_schema.json` under their respective `safe-outputs.*` sub-schemas, matching the Go types and semantics. We will also add the `comment-memory` top-level key (previously absent entirely from `safe-outputs.properties`). For the 3 compiler-populated fields, we will add an explicit `compiler_populated_fields` allowlist to `scripts/check-safe-outputs-conformance.sh` so IMP-004 skips them rather than reporting them as missing. The primary driver is IMP-004 conformance and closing the DX gap for workflow authors.

### Alternatives Considered

#### Alternative 1: Auto-generate the JSON schema from Go struct `yaml` tags

Generate `main_workflow_schema.json` entries automatically from Go struct reflection or `go generate` tooling, eliminating manual drift. This would make schema and Go implementation structurally impossible to diverge. It was not chosen because it requires building a code-generation pipeline and the schema also contains human-authored descriptions, constraints, and `anyOf`/`oneOf` wrappers that go beyond what struct tags can express automatically; the investment was not justified for this incremental fix.

#### Alternative 2: Annotate compiler-populated fields with a skip marker on Go structs

Add a `schema:"-"` tag (or equivalent) to the three compiler-populated Go struct fields and update the conformance checker to honour the annotation, rather than maintaining a hardcoded allowlist in the shell script. This would be more self-documenting at the field level. It was not chosen because it requires changing the Go struct definitions and establishing a new annotation convention, while the hardcoded allowlist in the script is simpler for the immediate fix; the three fields are stable and unlikely to change.

### Consequences

#### Positive
- Workflow authors gain full editor autocomplete, type checking, and schema-based documentation for all 10 previously undocumented user-facing safe-output config fields.
- IMP-004 conformance check now passes cleanly; the explicit `compiler_populated_fields` allowlist distinguishes internal-only fields from user-authored ones, preventing false positives on future runs.

#### Negative
- The JSON schema must continue to be manually maintained in sync with Go struct changes; any future new YAML-tagged field in a Go safe-output config struct requires a coordinated schema update to avoid regressing IMP-004.
- The `compiler_populated_fields` set in `check-safe-outputs-conformance.sh` is a separate maintenance artifact: if compiler-populated fields are renamed or added in Go, the allowlist must be updated in lockstep.

#### Neutral
- No change to the Go parsing or validation logic; existing workflow files that already use these fields continue to compile and run identically.
- The conformance checker script is now slightly more complex (a set lookup before the `missing` append), but the logic remains easy to follow.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
