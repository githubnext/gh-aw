# ADR-33491: Migrate High-Impact Validation Paths to Structured `NewValidationError`

**Date**: 2026-05-20
**Status**: Draft
**Deciders**: pelikhan

---

## Part 1 — Narrative (Human-Friendly)

### Context

Workflow validation in `pkg/workflow` historically returned errors via `fmt.Errorf(...)` or `errors.New(...)` with freeform message strings that mixed the offending field, the failure reason, and remediation guidance into a single sentence. The package already has a `WorkflowValidationError` type with discrete `Field`, `Value`, `Reason`, and `Suggestion` fields (defined in `pkg/workflow/workflow_errors.go`) and a `NewValidationError(field, value, reason, suggestion)` constructor, but many of the highest-traffic validation failures had not been migrated to it. As a result, users hitting compile-time errors for things like wildcard `network.allowed`, conflicting `branches`/`branches-ignore`, missing MCP `command`/`container`, or invalid `workflow_dispatch` inputs got generic prose instead of a field-anchored message with a YAML example.

### Decision

We will migrate the highest-frequency workflow validation failures to use `workflow.NewValidationError(field, value, reason, suggestion)` so each error carries a structured field path (e.g., `on.<event>.<filter>`, `mcp-servers.<name>.type`, `network.allowed`, `tools.cache-memory.scope`, `on.workflow_dispatch.inputs`), the offending value, a "expected …" reason, and a YAML-shaped suggestion that the user can copy-paste. The scope of this migration is the validation paths in `pkg/workflow/compiler_filters_validation.go`, `pkg/workflow/strict_mode_network_validation.go`, `pkg/workflow/mcp_property_validation.go`, `pkg/workflow/pip_validation.go`, and `pkg/cli/run_workflow_validation.go`. Tests in each touched package assert `errors.As(err, &*WorkflowValidationError)` and that `Suggestion` contains the expected YAML keys.

### Alternatives Considered

#### Alternative 1: Keep `fmt.Errorf` with richer freeform strings

We could continue using `fmt.Errorf` but standardize on a longer message template (field name, reason, example) inside the format string. This keeps the diff small and avoids touching the type system, but it leaves the message unparseable by downstream consumers (test assertions, audit tooling, IDE diagnostics) and offers no compile-time guarantee that the error includes a suggestion. It was rejected because the structured type already exists and adopting it consistently is a one-time cost.

#### Alternative 2: Migrate all validators in one sweep

We could migrate every `fmt.Errorf`/`errors.New` call in `pkg/workflow` to `NewValidationError` in a single PR. This was rejected because some errors are not validation failures (e.g., I/O, parsing, internal invariants) and would be miscategorized; doing it incrementally lets us prioritize the highest-frequency user-facing paths first while leaving non-validation error sites untouched.

#### Alternative 3: Introduce a new error builder/DSL

A fluent builder (`Validation().Field(…).Value(…).Expect(…).Example(…)`) could enforce required fields and reduce repetition. This was rejected as premature: the existing four-argument constructor is already concise, and a builder adds an abstraction layer for marginal benefit at the call sites being migrated.

### Consequences

#### Positive
- Users hitting these validation paths now get a copy-pasteable YAML example in the error output, which materially shortens the edit-compile-debug loop during workflow authoring.
- Downstream consumers (tests, audit, future IDE/LSP integrations) can type-assert `*WorkflowValidationError` and read the discrete fields instead of regex-parsing freeform strings.
- Error severity and category are now classified automatically via `classifyValidationSeverity` for every migrated site.

#### Negative
- The validation layer is now non-uniform: some sites use `NewValidationError`, others still use `fmt.Errorf`. Readers of the code must check both patterns and contributors must know which to reach for in new code (ADR-32507 carves out a similar non-uniformity for mount validation).
- Each migration site is more verbose at the call-site (a four-argument struct constructor vs. a single `fmt.Errorf` format string), which slightly increases line counts in already-dense validation files.
- The YAML examples in `Suggestion` are static strings; if the documented YAML schema changes (e.g., a key is renamed), every suggestion must be updated independently, with no compile-time link to the schema source of truth.

#### Neutral
- Existing error messages change shape (now include `Field`, `Value`, `Reason`, `Suggestion` headers from `WorkflowValidationError.Error()`); any external tooling that pattern-matched the previous message text would need to update its matchers.
- The PR also tightens the suggestion text in `pip_validation.go` to lead with a concrete `network.allowed` example, even though that site already used `NewValidationError`.

---

## Part 2 — Normative Specification (RFC 2119)

> The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this section are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### Validation Error Construction

1. New user-facing validation failures in `pkg/workflow` and `pkg/cli` that fault a specific frontmatter or configuration field **MUST** be returned via `workflow.NewValidationError(field, value, reason, suggestion)`.
2. The `field` argument **MUST** be a dotted YAML path rooted at the frontmatter top level (e.g., `on.push.branches`, `mcp-servers.my-server.type`, `tools.cache-memory.scope`, `network.allowed`).
3. The `value` argument **SHOULD** carry the offending value as observed (stringified) so it can be echoed back to the user, and **MAY** be empty when the failure is a missing-field condition.
4. The `reason` argument **MUST** be a short clause describing what was expected versus what was found (e.g., `"expected exactly one of 'branches' or 'branches-ignore'"`).
5. The `suggestion` argument **MUST** contain at least one YAML snippet under a top-level frontmatter key (e.g., starts with `on:`, `tools:`, `network:`, or `mcp-servers:`) that, if pasted, would resolve the failure.
6. New validation sites **MUST NOT** use `fmt.Errorf` or `errors.New` for failures that fault an identifiable frontmatter or configuration field.

### Scope and Exceptions

1. Non-validation errors (I/O, network, parsing, internal invariants) **MUST NOT** be wrapped in `NewValidationError`; they **SHOULD** use `fmt.Errorf` with `%w` or a domain-specific error type (`OperationError`, `ConfigurationError`).
2. MCP mount validation (`mcp_*_mount_*`) **MUST** continue to return errors via `fmt.Errorf` per the carve-out documented in [ADR-32507](32507-consolidate-config-parser-proxies-and-share-mount-classification.md); this ADR does not override that exception.
3. Pre-existing `fmt.Errorf`/`errors.New` sites outside the migration scope listed in the Decision section **MAY** remain unchanged until they are touched for an unrelated reason.

### Testing

1. Each migrated validation site **MUST** have at least one unit test that asserts `errors.As(err, &*WorkflowValidationError)` succeeds and that `Suggestion` is non-empty.
2. The suggestion assertion **SHOULD** verify the presence of the expected top-level YAML key (e.g., `on:`, `tools:`, `network:`) to guard against accidental loss of the example block.

### Conformance

An implementation is considered conformant with this ADR if it satisfies all **MUST** and **MUST NOT** requirements above. Failure to meet any **MUST** or **MUST NOT** requirement constitutes non-conformance.

---

*This is a DRAFT ADR generated by the [Design Decision Gate](https://github.com/github/gh-aw/actions/runs/26162336064) workflow. The PR author must review, complete, and finalize this document before the PR can merge.*
