# ADR-29236: github_ref Constraint for Compile-Time Action Pinning in Import Schemas

**Date**: 2026-04-30
**Status**: Draft
**Deciders**: pelikhan, copilot-swe-agent

---

## Part 1 — Narrative (Human-Friendly)

### Context

The workflow compiler processes `import-schema` inputs that callers pass when importing shared workflow files. Some of those inputs are GitHub repository references (`owner/repo` or `owner/repo/path`) that are later used to fetch or reference external action packages (e.g., APM packages). Without a declared type constraint, the compiler treats these values as opaque strings: it does not validate their format, and it does not pin mutable tags or branches to immutable SHAs. This creates a supply-chain risk — a mutable ref such as `owner/repo@main` could silently resolve to a different commit at runtime without any indication in the compiled output.

### Decision

We will add an opt-in `github_ref: true` boolean property to `import-schema` input definitions (for both scalar `string` and `string[]` array items). When `github_ref: true` is set, the compiler validates that the caller-supplied value matches the `owner/repo[@ref]` or `owner/repo/path[@ref]` format, and it resolves and pins those references to `repo@sha # version` strings via the existing `actionpins` pin manager — before expression substitution — so pinned values flow naturally through the template engine. The pinner is injected into the import cache via a new `GitHubRefPinner` interface, keeping the parser package independent of the concrete pin-resolution logic in the workflow compiler.

### Alternatives Considered

#### Alternative 1: Runtime Pinning (at Workflow Execution Time)

Resolve and pin GitHub references when the workflow actually executes rather than at compile time. This would avoid coupling the compiler to the pin manager and would allow the workflow runner to resolve the latest SHA at the point of execution.

This was rejected because the existing action-pin strategy in the codebase (for `uses:` directives) pins at compile time to produce deterministic, auditable compiled output. Moving pinning to runtime would mean compiled workflow files contain mutable refs, breaking the supply-chain guarantee that a compiled workflow is a reproducible artifact. Compile-time pinning also enables CI checks to fail on unresolvable references before deployment.

#### Alternative 2: Dedicated Top-Level `github-packages:` Field

Introduce a new top-level key (e.g., `github-packages:`) in the workflow frontmatter specifically for listing action package references that need pinning, separate from the general-purpose `import-schema` mechanism.

This was rejected because it would fragment the import-schema contract: callers already declare typed inputs via `import-schema`, and a separate key for "pinnable inputs" would require authors to declare the same concept in two places. Extending `import-schema` with a first-class `github_ref` constraint reuses the existing validation and defaulting infrastructure and keeps the schema self-describing.

### Consequences

#### Positive
- Supply-chain security for external package references is enforced at compile time, producing pinned SHAs in compiled output.
- The feature is opt-in (`github_ref: false` by default), so existing import-schema definitions are completely unaffected.
- Graceful degradation: when a SHA cannot be resolved (not in the embedded pins database), the original value is left unchanged and a warning is logged — compilation always succeeds.
- The `GitHubRefPinner` interface decouples the parser from the workflow compiler, making the parser independently testable with mock pinners.

#### Negative
- The `ImportCache` struct now carries an optional `Pinner` field that callers must set if pinning is desired; forgetting to set it silently skips pinning rather than erroring.
- The `nil`-pinner branch adds a code path that is exercised only in parser-only contexts (e.g., LSP, schema validation), requiring care when adding future callers that need pinning.
- The `github_ref` format is GitHub-specific, making this constraint non-portable if the import system is ever extended to non-GitHub sources.

#### Neutral
- Pinning happens before expression substitution (`resolveGitHubRefInputs` is called after defaults are applied but before `substituteImportInputsInContent`), which means `${{ github.aw.import-inputs.package }}` expressions will see the already-pinned SHA value.
- The JSON schema (`main_workflow_schema.json`) is updated to expose `github_ref` as a recognized property, which improves editor autocompletion and schema validation for workflow authors.

---

## Part 2 — Normative Specification (RFC 2119)

> The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this section are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### Input Validation

1. An `import-schema` parameter definition **MAY** include a `github_ref` boolean property; when absent or `false`, the parameter **MUST** be treated as a plain string with no format constraint.
2. When `github_ref: true` is set on a `string`-typed parameter, the compiler **MUST** validate that the caller-supplied value matches the pattern `owner/repo`, `owner/repo/path`, `owner/repo@ref`, or `owner/repo/path@ref`; non-conforming values **MUST** cause a compilation error.
3. When `github_ref: true` is set on an `array`-typed parameter whose `items` schema has `github_ref: true`, the compiler **MUST** validate every element of the array against the same format rule; a single non-conforming element **MUST** cause a compilation error.
4. The `github_ref` property **MUST NOT** be applied to parameter types other than `string` scalars or `string[]` arrays.

### Compile-Time Pinning

1. After input defaults are applied and before expression substitution, implementations **MUST** invoke the `GitHubRefPinner` (if one is configured) on every value belonging to a `github_ref: true` parameter.
2. If the pinner returns an empty string or an error, the implementation **MUST** leave the original value unchanged and **SHOULD** emit a warning to the compile log; compilation **MUST NOT** fail solely because a pin could not be resolved.
3. When a `GitHubRefPinner` is not configured (nil pinner), implementations **MUST** skip pin resolution entirely and pass the validated but unpinned value through unchanged.
4. The `GitHubRefPinner` interface **MUST** be defined in the `parser` package and **MUST NOT** import concrete pin-resolution packages, preserving parser-package independence.

### Schema Registration

1. The `github_ref` boolean property **MUST** be declared in the canonical JSON schema (`main_workflow_schema.json`) for both the scalar input definition object and the `items` sub-schema of array-typed inputs.
2. The `github_ref` property **MUST** default to `false` in the JSON schema.

### Conformance

An implementation is considered conformant with this ADR if it satisfies all **MUST** and **MUST NOT** requirements above. Failure to meet any **MUST** or **MUST NOT** requirement constitutes non-conformance.

---

*This is a DRAFT ADR generated by the [Design Decision Gate](https://github.com/github/gh-aw/actions/runs/25144879707) workflow. The PR author must review, complete, and finalize this document before the PR can merge.*
