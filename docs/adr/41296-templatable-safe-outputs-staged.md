# ADR-41296: Make `safe-outputs.staged` a Templatable Boolean

**Date**: 2026-06-24
**Status**: Draft
**Deciders**: pelikhan, copilot-swe-agent [TODO: verify human deciders]

---

### Context

The `safe-outputs.staged` flag toggles preview mode, where safe-output handlers emit step-summary messages instead of making real GitHub API calls. It was modeled as a plain `boolean` in the JSON schema and as a Go `bool` in the workflow types. This prevented expression-based configuration: a workflow author could not write `staged: ${{ inputs.staged }}` or `staged: ${{ github.event_name != 'push' }}`, because the schema rejected non-literal strings and the compiler had no way to carry an unresolved expression through to the runtime. As a result, dynamic, context-driven preview toggling was impossible without duplicating workflow definitions.

### Decision

We will make `safe-outputs.staged` a **templatable boolean** end to end: the schema accepts `#/$defs/templatable_boolean` (literal booleans or GitHub expression strings), config parsing accepts both forms at the top level and per handler, the compiler carries the value as `*TemplatableBool` instead of `bool`, and the JavaScript handlers treat a resolved expression value (`"true"`) the same as a literal `true` via a shared `isTemplatableTrue` helper. Centralizing resolution keeps literal `true`, literal `false`, and expressions handled consistently across all handlers.

### Alternatives Considered

#### Alternative 1: Keep `staged` boolean-only

Leave the flag as a literal boolean and require authors who want conditional preview mode to maintain separate workflow files or rely on the global `GH_AW_SAFE_OUTPUTS_STAGED` environment variable. Rejected because it forces workflow duplication and cannot express per-handler conditional staging driven by inputs or event context.

#### Alternative 2: Add a separate `staged-expression` field

Introduce a parallel field (e.g., `staged-when`) accepting only expressions, leaving `staged` as a literal boolean. Rejected because it splits one concept across two fields, complicates precedence rules, and produces a less intuitive schema than a single templatable field that accepts either form.

#### Alternative 3: Resolve expressions only in JS, no type changes

Keep the Go type as `bool` and push all expression handling into the JavaScript runtime. Rejected because the compiler would have nowhere to store an unresolved expression, schema validation would still reject expression strings, and the value could not flow cleanly from frontmatter through config-JSON generation to env vars.

### Consequences

#### Positive
- Authors can drive preview mode dynamically with GitHub expressions (`${{ inputs.staged }}`, event-based conditions) at both global and per-handler scope.
- A single shared `isTemplatableTrue` helper gives all handlers consistent truthiness semantics, replacing ad hoc `config.staged === true` checks.

#### Negative
- Converting workflow types from `bool` to `*TemplatableBool` introduces pointer/nil and string-resolution handling that callers must respect, increasing surface for nil-dereference or mis-typed comparisons.
- Unresolved expression strings (e.g., `"${{ inputs.staged }}"` that never resolves) evaluate to `false`, so a misconfigured expression silently runs in live mode rather than preview — a fail-open behavior for staging that authors must be aware of.

#### Neutral
- The schema change touches many `staged` definitions (every safe-output type) but is mechanical, swapping `"type": "boolean"` for `"$ref": "#/$defs/templatable_boolean"`.
- Existing literal-boolean configurations remain valid and behave identically; this is a backward-compatible widening of accepted values.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
