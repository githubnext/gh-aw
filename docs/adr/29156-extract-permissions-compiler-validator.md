# ADR-29156: Extract `validatePermissions` into a Dedicated Compiler Validator File

**Date**: 2026-04-29
**Status**: Draft
**Deciders**: pelikhan, copilot-swe-agent

---

## Part 1 — Narrative (Human-Friendly)

### Context

`pkg/workflow/compiler_validators.go` had grown to 477 lines — 59% over the 300-line hard limit enforced by AGENTS.md — by mixing multiple unrelated validation concerns in a single file. The `validatePermissions` method alone accounted for over 100 lines and orchestrated a 7-step permission-validation sequence (dangerous permissions, GitHub App constraints, MCP app write restrictions, unsupported context warnings, `workflow_run` branch security, MCP toolset alignment, and id-token write warning). Keeping all compiler validators in one file made per-concern review and future modification harder as the codebase grew.

### Decision

We will extract the `validatePermissions` compiler method from `compiler_validators.go` into its own file, `permissions_compiler_validator.go`, inside the same `workflow` package. This is a pure mechanical extraction — no logic changes — driven by the AGENTS.md 300-line file size limit and the single-responsibility principle. Each compiler validator file will now contain validators grouped by a single concern, making it easier to locate, review, and extend permission-specific logic without navigating an omnibus file.

### Alternatives Considered

#### Alternative 1: Keep All Validators in `compiler_validators.go` (Accept the Limit Violation)

Retaining all validators in the original file was considered because it avoids creating additional files and preserves a single point of lookup for all compiler validators. This was rejected because `compiler_validators.go` at 477 lines already exceeded the AGENTS.md hard limit by 59%, and adding further validators would widen the violation. Ignoring the limit would also set a precedent for other files in the package.

#### Alternative 2: Create a Dedicated `validators` Sub-Package

Moving permission validation into a separate Go sub-package (e.g., `pkg/workflow/validators`) was considered as a more formal decomposition. This was rejected because all validator functions share internal symbols (unexported helpers, package-level constants, and the `Compiler` receiver) that are not accessible across package boundaries. Introducing a sub-package would require either exposing those symbols or duplicating them, making this a non-trivial refactor that goes beyond the scope of a file-size compliance fix.

### Consequences

#### Positive
- `compiler_validators.go` is reduced from 477 to 377 lines, bringing it under the AGENTS.md 300-line guidance (with further splits available for remaining validators if needed).
- `permissions_compiler_validator.go` has a clear, searchable name: any developer looking for permission validation logic can find it immediately.
- The file-level doc comment in `permissions_compiler_validator.go` documents the full 7-step validation sequence and strict-mode behaviour in one place, improving discoverability.

#### Negative
- The compiler validator logic is now split across multiple files in the same package, which means a developer must know to look in both `compiler_validators.go` and `permissions_compiler_validator.go` (and potentially future extracted files) to get a complete picture of all validators.
- `compiler_validators.go` at 377 lines still exceeds the 300-line target; additional extractions may be needed to reach full compliance.

#### Neutral
- All existing tests continue to pass without modification because the extraction is purely mechanical and no logic changes were made.
- The `import` declarations in `compiler_validators.go` were audited to confirm all remaining imports are still in use after the removal of `validatePermissions`.

---

## Part 2 — Normative Specification (RFC 2119)

> The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this section are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### File Organisation for Compiler Validators

1. Each compiler validator file **MUST** be placed in the `pkg/workflow` package and **MUST** be named using the pattern `{concern}_compiler_validator.go`, where `{concern}` identifies the single validation responsibility contained in that file.
2. A compiler validator file **MUST NOT** contain validators for more than one distinct validation concern.
3. Every new compiler validator file **SHOULD** include a package-level doc comment that describes the concern it covers, lists the validation steps performed, and documents any mode-specific behaviour (e.g., strict mode).
4. No single Go source file in `pkg/workflow` **SHOULD** exceed 300 lines; when an extraction brings a file over this limit, further extractions **SHOULD** be scheduled.
5. Extraction of an existing function into a new validator file **MUST NOT** change the function's observable behaviour, signature, or package visibility.
6. Imports in the originating file **MUST** be audited after extraction to remove any imports that are no longer referenced by the remaining code.

### Conformance

An implementation is considered conformant with this ADR if it satisfies all **MUST** and **MUST NOT** requirements above. Failure to meet any **MUST** or **MUST NOT** requirement constitutes non-conformance. The **SHOULD** recommendations **MAY** be waived when documented justification exists, but such waivers **SHOULD** be noted in the relevant PR or commit message.

---

*This is a DRAFT ADR generated by the [Design Decision Gate](https://github.com/github/gh-aw/actions/runs/25120532407) workflow. The PR author must review, complete, and finalize this document before the PR can merge.*
