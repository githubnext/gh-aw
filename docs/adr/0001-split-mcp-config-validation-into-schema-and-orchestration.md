# ADR-0001: Split MCP Config Validation into Schema and Orchestration Files

**Date**: 2026-04-10
**Status**: Draft
**Deciders**: pelikhan, Copilot

---

## Part 1 — Narrative (Human-Friendly)

### Context

The `pkg/workflow/mcp_config_validation.go` file grew to 462 lines while containing two
logically distinct validation domains: (1) the public API and orchestration layer for MCP
configuration validation, and (2) type-specific requirements and JSON schema validation for
stdio vs http MCP types. The repository's AGENTS.md enforces a 300-line hard limit per file
to maintain readability and focused responsibility. With 462 lines, the file exceeded this
limit by 162 lines, and the two domains map cleanly enough to separate files that a split is
the natural remedy.

### Decision

We will split `mcp_config_validation.go` into two files within the same `package workflow`:
`mcp_config_validation.go` retains the public API and orchestration functions
(`ValidateMCPConfigs()`, `ValidateToolsSection()`, `getRawMCPConfig()`, `inferMCPType()`),
while `mcp_schema_validation.go` houses the type-specific validation and schema-construction
functions (`validateStringProperty()`, `validateMCPRequirements()`, `mcpSchemaTopLevelFields`,
`buildSchemaMCPConfig()`, `validateMCPMountsSyntax()`). This is a purely structural split with
no logic changes; all existing call sites remain unaffected because both files share the same
package namespace.

### Alternatives Considered

#### Alternative 1: Leave the File as a Single 462-Line File

Keeping all validation logic in one file and requesting an exception to the 300-line rule in
AGENTS.md was the simplest option. This was rejected because the line limit exists to enforce
focused responsibility, and the two validation domains (orchestration vs. type-specific
requirements) represent genuinely distinct concerns that benefit from separation regardless of
the limit.

#### Alternative 2: Extract Schema Validation into a Separate Package

Schema validation logic could be moved to a new `pkg/workflow/mcp` sub-package, creating a
harder boundary between the orchestration and schema layers. This was not chosen because the
two domains share internal helpers (e.g., `inferMCPType`) and a separate package would add
import overhead without proportional benefit — the domains are closely related and the
additional indirection would complicate future maintenance.

#### Alternative 3: Merge with the Existing `validation.go` File

The schema validation functions could be merged into the general `validation.go` file. This
was rejected because MCP-specific validation logic would pollute the general validation file,
making it harder to understand what belongs to MCP configuration vs. general workflow
validation, and likely pushing `validation.go` over the 300-line limit as well.

### Consequences

#### Positive
- Both files comply with the AGENTS.md 300-line hard limit (253 lines each after the split).
- Each file has a single, well-defined responsibility: orchestration vs. type-specific
  validation, which aids future contributors in locating the right code quickly.
- File-level comments in each file cross-reference the other, providing discoverability hints.

#### Negative
- `validateMCPRequirements()` in `mcp_schema_validation.go` calls `inferMCPType()` defined in
  `mcp_config_validation.go`, creating a cross-file dependency within the same package that is
  not immediately obvious from file names alone.
- The split is a naming convention, not a package boundary, so there is no compiler-enforced
  separation; future additions could inadvertently blur the division of responsibility.

#### Neutral
- Both files remain in `package workflow`, so no callers are affected and no import paths
  change.
- The total line count of the combined validation surface is unchanged; only the organizational
  boundary shifts.
- The `mcpSchemaTopLevelFields` map, which must be kept in sync with
  `pkg/parser/schemas/mcp_config_schema.json`, moves to `mcp_schema_validation.go` — a more
  semantically appropriate home.

---

## Part 2 — Normative Specification (RFC 2119)

> The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**,
> **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this section are to be
> interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### File Organization

1. Implementations **MUST** keep `mcp_config_validation.go` focused on the public API and
   orchestration layer: `ValidateMCPConfigs()`, `ValidateToolsSection()`, `getRawMCPConfig()`,
   and `inferMCPType()`.
2. Implementations **MUST** keep `mcp_schema_validation.go` focused on type-specific
   validation and schema construction: `validateStringProperty()`,
   `validateMCPRequirements()`, `mcpSchemaTopLevelFields`, `buildSchemaMCPConfig()`, and
   `validateMCPMountsSyntax()`.
3. Both files **MUST** remain in `package workflow` to preserve the existing package API and
   avoid breaking any callers.
4. Each file **MUST NOT** exceed 300 lines, per the AGENTS.md file size policy.

### Cross-File Dependencies

1. Functions in `mcp_schema_validation.go` **MAY** call functions defined in
   `mcp_config_validation.go` (e.g., `inferMCPType`) since both files share the same package
   namespace.
2. New helper functions that serve only one validation domain **SHOULD** be placed in the
   file for that domain rather than split arbitrarily across both files.
3. If a helper function grows to serve both domains, it **SHOULD** be documented with a
   comment noting its shared use and placed in whichever file represents its primary consumer.

### Schema Field Maintenance

1. The `mcpSchemaTopLevelFields` map in `mcp_schema_validation.go` **MUST** be kept in sync
   with the properties declared in `pkg/parser/schemas/mcp_config_schema.json`.
2. When a property is added to or removed from `mcp_config_schema.json`, the corresponding
   entry in `mcpSchemaTopLevelFields` **MUST** be updated in the same pull request.

### Conformance

An implementation is considered conformant with this ADR if it satisfies all **MUST** and
**MUST NOT** requirements above. Specifically: the public API and orchestration functions live
in `mcp_config_validation.go`, the type-specific schema validation functions live in
`mcp_schema_validation.go`, both files remain in `package workflow`, neither file exceeds 300
lines, and `mcpSchemaTopLevelFields` is kept in sync with the JSON schema. Failure to meet any
**MUST** or **MUST NOT** requirement constitutes non-conformance.

---

*This is a DRAFT ADR generated by the [Design Decision Gate](https://github.com/github/gh-aw/actions/runs/24252303485) workflow. The PR author must review, complete, and finalize this document before the PR can merge.*
