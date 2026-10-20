# ADR-70616: Reject Unsafe Action Pins with Explicit Errors Instead of Escaping

**Date**: 2026-10-20
**Status**: Draft
**Deciders**: pelikhan (PR author); [TODO: verify additional deciders]

---

### Context

The compiler emits `uses:` lines into generated YAML from action pins. Those pins can come from the built-in pin cache, a SHA resolver, or user-configured action mappings and prefixes. Single-quoted references were already escaped for apostrophes (`escapeYAMLSingleQuoted`), so code scanning alerts 1030/1031 were dismissed. However, `quoteYAMLActionPin` had a `string`-only return type, so it could not signal failure. It could therefore still emit a resolved pin or version comment that contained a newline, a carriage return, a Unicode line or paragraph separator (U+0085, U+2028, U+2029), or a `${{ ... }}` expression. Any of these could break out of the `uses:` scalar or comment and inject YAML keys such as `run:`. The maintenance-workflow label jobs (`label_disable_agentic_workflow`, `label_apply_safe_outputs`) and quoted action-pin mapping (`mapPinnedUsesLine`) both used this path. Issue #69882 asked for defense in depth for these alerts.

### Decision

We will validate action pins and their version comments before quoting them, and reject unsafe values with an explicit error instead of sanitizing or escaping them. The new `validateYAMLActionPin` rejects any Unicode control character except tab, U+2028/U+2029, and any `${{` sequence. `quoteYAMLActionPin` now returns `(string, error)`. The error propagates through `buildMaintenanceLabelDisableJob`, `buildMaintenanceLabelApplySafeOutputsJob`, `buildMaintenanceLabelTriggeredJobs`, `buildMaintenanceWorkflowYAML`, and `mapPinnedUsesLine`, so compilation fails rather than writing unsafe YAML. We chose fail-closed rejection because legitimate pins never need these characters, and silently changing a pin would hide a supply-chain or configuration problem.

### Alternatives Considered

#### Alternative 1: Escape or strip unsafe characters and keep the `string` return type

We could strip or escape control characters and expressions inside `quoteYAMLActionPin`, as is already done for apostrophes. That would avoid changing function signatures across the maintenance builders. We rejected it because a silently rewritten pin may no longer match the action the author intended. It would also hide a bad resolver result or mapping instead of surfacing it. [TODO: verify whether this was explicitly considered]

#### Alternative 2: Rely on the existing dismissals and emit double-quoted YAML scalars

We could keep the dismissed alerts as-is, or switch to double-quoted scalars with full YAML escaping. Double-quoted scalars would contain newlines inside the reference. They would not protect the unquoted ` # version` comment, and they would not block `${{ }}` expression evaluation by GitHub Actions. They would also change the generated output format across many lock files.

#### Alternative 3: Validate pins once at their source (resolver/mapping configuration)

We could validate pins only where they enter the system, such as resolver responses or mapping configuration parsing. This approach would be cleaner, but pins reach the emitter from several sources, including prefixes, mappings, the cache, and resolvers. Validating at the YAML emission point covers all of these sources in one place.

### Consequences

#### Positive
- YAML injection through action pins or version comments is blocked at emission time, whatever the source of the pin.
- Failures are explicit and include context, such as `label_apply_safe_outputs: invalid action pin: ...` and `invalid action pin comment for repo@version`, so misconfiguration is easy to diagnose.
- Valid output is unchanged: apostrophe escaping, tab-separated mapping comments, and existing lock files stay byte-identical. Of the 345 workflows recompiled with no drift, none produced a validation error.

#### Negative
- The maintenance job builders now return `(string, error)`, which adds error-plumbing boilerplate and changes their call sites, including tests.
- Error handling is now inconsistent across maintenance builders: other job builders (for example, `buildMaintenanceCloseIssuesJob`) still return a bare `string`.
- A legitimate but unusual pin that contains a forbidden character would now fail compilation and could not be used. [TODO: verify that no such pins exist]

#### Neutral
- The validation rule (control characters except tab, Unicode line separators, `${{`) is now a shared policy that future YAML emitters of action references should reuse.
- The PR also adds test-only hardening (pip option-like package names, Pi subagent stdin-only task passing) that does not change runtime behavior.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
