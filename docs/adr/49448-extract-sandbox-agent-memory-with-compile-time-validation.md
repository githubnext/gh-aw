# ADR-49448: Extract sandbox.agent.memory from Frontmatter with Compile-Time Format Validation

**Date**: 2026-08-01
**Status**: Draft
**Deciders**: Unknown

---

### Context

The `sandbox.agent.memory` field was declared in the AWF workflow schema and consumed by the AWF arg builder to produce `--memory-limit`. However, `extractAgentSandboxConfig` never assigned the field, so `--memory-limit` was silently dropped from every compiled lock file. Users who set `sandbox.agent.memory` saw no error, no warning, and no effect — the field appeared functional in documentation but was effectively broken. Without memory overrides, large-memory build tools (MSBuild, JVM processes) hit the default container memory cap and were killed with exit code 137.

### Decision

We will complete the extraction of `sandbox.agent.memory` inside `extractAgentSandboxConfig`, mirroring the existing `mounts`/`runtime` extraction pattern, and add a dedicated `validateAgentMemoryLimit` function that reuses the existing `memoryLimitPattern` regex to reject malformed values (e.g. `48gb`, `48`) at compile time rather than at execution time.

### Alternatives Considered

#### Alternative 1: Leave the silent drop in place (status quo)

The field remains schema-documented but has no effect. Users who configure memory limits see no error and receive no benefit. This is the current accidental state — it is not a deliberate choice and provides no value; keeping it active would be actively misleading.

#### Alternative 2: Extract the field but defer validation to AWF at execution time

The extraction step is added but no compile-time validation is performed; AWF is responsible for rejecting invalid format strings when it starts the container. This delays error detection to workflow execution, producing a runtime failure rather than a compile-time failure, and makes the error message less actionable since it surfaces far from the authoring step.

#### Alternative 3: Extract and validate at compile time (chosen)

Extraction is added alongside a compile-time validator that matches the pattern already used for bounded-query memory limits. Invalid formats are rejected at `gh aw compile` time with a descriptive error pointing to the docs. This is consistent with how other `sandbox` sub-fields are validated and provides the best developer experience.

### Consequences

#### Positive
- `sandbox.agent.memory` now propagates correctly to AWF as `--memory-limit`, making the feature functional.
- Malformed memory values are caught at compile time with a clear, actionable error message, consistent with the existing validation philosophy for sandbox fields.
- The fix reuses the existing `memoryLimitPattern`, so no new regex is introduced.

#### Negative
- `validateAgentMemoryLimit` is a thin wrapper around `memoryLimitPattern`; if the pattern's semantics change, callers must be updated consistently.
- Any workflow that previously silently had `sandbox.agent.memory` set to an invalid string will now fail at compile time — a breaking change for malformed configs, though this surfaces a pre-existing latent error.

#### Neutral
- Test coverage is added for the extraction path (valid string, absent, non-string type) and for the validation function (valid units, invalid suffixes, leading zeros, zero value).
- Documentation in `docs/reference/sandbox.md` now includes valid format examples and an exit-137 note.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
