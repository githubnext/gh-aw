# ADR-29793: Best-Effort Inline Sub-Agent Frontmatter Field Validation

**Date**: 2026-05-02
**Status**: Draft
**Deciders**: pelikhan (PR author), gh-aw maintainers

---

## Part 1 — Narrative (Human-Friendly)

### Context

Inline sub-agents (introduced in ADR-29668) let workflow authors embed agent definitions directly inside a workflow file using `## agent: \`name\`` heading markers, each optionally followed by a YAML frontmatter block. Only two fields are semantically meaningful in sub-agent frontmatter: `description` (a human-readable summary) and `model` (the LLM to use). Fields such as `engine`, `on`, `permissions`, or any workflow-level key are silently ignored when placed inside a sub-agent block, because sub-agents inherit those settings from the parent workflow. Without validation, authors can inadvertently copy workflow frontmatter into a sub-agent block or typo a field name and receive no feedback, creating silent misconfigurations that are difficult to debug.

### Decision

We will perform best-effort validation of inline sub-agent frontmatter fields during two existing processing phases: the static `imports:` BFS traversal (via `importAccumulator`) and `{{#runtime-import}}` resolution (via `validateRuntimeImportFiles`). Any field key other than `description` or `model` produces an advisory warning written to stderr via `console.FormatWarningMessage`. Validation failures never abort compilation; they are purely informational, accumulating in `ImportsResult.Warnings` for propagation to the compiler's warning counter.

### Alternatives Considered

#### Alternative 1: Fatal validation errors

Unknown sub-agent frontmatter fields could be treated as hard compilation errors that abort the build. This gives the strongest signal to authors and prevents misconfigurations from silently shipping. It was not chosen because existing workflow files may already contain unknown fields that have been harmlessly ignored; making validation fatal would be a breaking change requiring a migration period.

#### Alternative 2: Deferred runtime-only validation

Validation could be deferred entirely to the moment a sub-agent is actually invoked at runtime (e.g., when the Copilot CLI reads the extracted `.agent.md` file). This avoids any compile-time overhead. It was not chosen because it gives authors feedback much later in the development cycle — after a workflow run has started — compared to surfacing warnings during the import phase, which is traversed before the workflow even executes.

#### Alternative 3: No validation (status quo)

Unknown fields in sub-agent frontmatter are already silently discarded by the YAML parser. Maintaining this behaviour requires zero code and avoids any risk of false positives. It was not chosen because silent discarding makes authoring errors invisible; a developer who writes `engine: copilot` in a sub-agent block will never learn that the field has no effect.

### Consequences

#### Positive
- Authors receive immediate, actionable feedback when sub-agent frontmatter contains unsupported fields, reducing silent misconfiguration.
- The implementation is strictly additive and backwards-compatible: no existing workflow fails to compile because of these warnings.
- Warning messages include the offending file path, agent name, unknown field names, and the complete list of valid fields, giving authors enough context to fix the issue without consulting documentation.

#### Negative
- The valid field set (`description`, `model`) is hardcoded in `validSubAgentFrontmatterFields`. Adding a new supported frontmatter field requires a source-code change and a deployment; there is no runtime extensibility.
- The validation function must be called from two separate integration points (`import_field_extractor.go` and `runtime_import_validation.go`). Future changes to the set of processing phases that load workflow files must remember to invoke it, or the coverage will silently regress.

#### Neutral
- Warnings are emitted to stderr and counted by the compiler's warning counter; they are not surfaced as GitHub check annotations or pull-request comments by this PR.
- Top-level workflow frontmatter is explicitly excluded from sub-agent validation: only content after the `## agent:` heading is inspected.

---

## Part 2 — Normative Specification (RFC 2119)

> The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this section are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### Valid Fields

1. Inline sub-agent frontmatter blocks **MUST NOT** contain any key other than `description` or `model`.
2. Implementations **MUST** treat any other key as an unknown field and emit an advisory warning.
3. Implementations **MUST NOT** fail compilation solely because an unknown field is present in sub-agent frontmatter.

### Warning Format and Propagation

1. Each warning **MUST** identify the source file path, the sub-agent name, the unknown field(s), and the complete list of valid fields.
2. Warnings **MUST** be emitted to stderr using the project's standard `console.FormatWarningMessage` formatter.
3. Warnings **MUST** increment the compiler's warning counter so they are reflected in build summaries.
4. Implementations **SHOULD** accumulate all warnings from a file and return them together rather than aborting on the first unknown field, so authors can fix all issues in one edit.

### Integration Points

1. Sub-agent frontmatter validation **MUST** be invoked for every file processed during the static `imports:` BFS traversal.
2. Sub-agent frontmatter validation **MUST** be invoked for every file resolved via a `{{#runtime-import}}` directive.
3. Validation **MUST** strip the file's top-level frontmatter before scanning for `## agent:` sections, so workflow-level fields are not misidentified as sub-agent content.
4. Implementations **MAY** skip validation if the file contains no `## agent:` headings (as a performance optimisation); the observable result **MUST** be identical.

### Conformance

An implementation is considered conformant with this ADR if it satisfies all **MUST** and **MUST NOT** requirements above. Failure to meet any **MUST** or **MUST NOT** requirement constitutes non-conformance.

---

*This is a DRAFT ADR generated by the [Design Decision Gate](https://github.com/github/gh-aw/actions/runs/25258037565) workflow. The PR author must review, complete, and finalize this document before the PR can merge.*
