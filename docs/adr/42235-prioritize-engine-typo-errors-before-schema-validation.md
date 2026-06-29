# ADR-42235: Prioritize Engine Typo Errors Before Schema Validation

**Date**: 2026-06-29
**Status**: Draft
**Deciders**: pelikhan, copilot-swe-agent

---

### Context

The workflow compiler validates frontmatter in two sequential stages: JSON schema validation (which checks field names and value types) followed by engine-specific validation. When a workflow file contains both an engine typo (e.g., `engine: copiilot`) and an unrelated schema violation (e.g., an unknown field), the schema validator runs first and its generic error shadows the more actionable "invalid engine" error with the "Did you mean: copilot?" suggestion. The user receives a confusing, misleading error that points to an unrelated problem instead of the root cause. Additionally, when `engine` holds a non-string value (e.g., `engine: 123`), the schema error reported a duplicate `(line N, col N)` location in the message body, producing redundant noise in the output.

### Decision

We will pre-validate the `engine` field as a string value before the main JSON schema validation and import resolution steps. If the engine value is a non-empty string that is not a recognized engine name, the compiler will immediately return an error pointing to the `engine` field's source location with the existing suggestion text (e.g., "Did you mean: copilot?"), short-circuiting all further validation. This is implemented as `validateStringEngineBeforeSchema`, called from both `parseFrontmatterSection` and `ParseWorkflowString`, ensuring consistent behaviour across file-based and in-memory compilation paths.

### Alternatives Considered

#### Alternative 1: Post-process and reorder all collected errors

Collect all validation errors from schema validation, import resolution, and engine validation, then sort or filter them so engine errors are promoted to first position before returning to the caller.

This was not chosen because it requires a full validation pass before any error can be surfaced, increases complexity by adding an error-sorting/ranking layer, and risks silently dropping or interleaving errors that depend on ordering guarantees elsewhere in the pipeline.

#### Alternative 2: Encode engine validation inside the JSON schema as an enum constraint

Add the list of valid engine names as an `enum` in the frontmatter JSON schema so that schema validation itself rejects invalid engine values, producing a single unified validation pass.

This was not chosen because it would lose the rich, tailored error message ("Did you mean: copilot?" with fuzzy matching) that the existing `getAgenticEngine` error path provides, and would require the schema to be regenerated every time a new engine is added, creating a maintenance coupling between the schema artifact and the engine registry.

### Consequences

#### Positive
- Engine typos always surface at the correct source line and column with the "Did you mean: X?" suggestion, regardless of what other errors are present in the file.
- Engine type errors (non-string values) now produce a single, clean error location without duplicate `(line N, col N)` fragments in the message body.
- The fix applies consistently across both compilation entry points (`CompileWorkflow` and `ParseWorkflowString`).

#### Negative
- Every compilation of a workflow with a non-empty string `engine` field now incurs an extra `getAgenticEngine` lookup before schema validation — a minor performance overhead for valid workflows.
- The `engine` field now has a dual validation pathway: the pre-validation step handles string-typed typos, while the JSON schema still handles non-string types and missing values. These two paths must be kept in sync if the engine field contract changes.

#### Neutral
- Two new regression tests are added to the compiler test suite covering engine-typo-before-schema-errors and engine-typo-before-import-errors scenarios; existing tests for the import-precedence case remain unchanged.
- The `validateStringEngineBeforeSchema` helper is defined on `*Compiler` and shares the existing `formatCompilerErrorWithContext` and `readSourceContextLines` infrastructure.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
