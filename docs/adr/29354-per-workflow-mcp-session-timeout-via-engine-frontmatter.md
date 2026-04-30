# ADR-29354: Per-Workflow MCP Session Timeout via `engine.mcp` Frontmatter

**Date**: 2026-04-30
**Status**: Draft
**Deciders**: lpcox, Copilot SWE Agent

---

## Part 1 — Narrative (Human-Friendly)

### Context

All workflows sharing a single MCP gateway instance inherit the same session lifetime, controlled globally by the `MCP_GATEWAY_SESSION_TIMEOUT` environment variable set on the gateway container. There is no mechanism for individual workflow authors to declare the session lifetime their workflow needs. This is a problem because workflows have wildly different runtime profiles: a short code-review workflow may finish in minutes while a large-scale migration may run for several hours and require a long-lived MCP session to avoid mid-task reconnection failures.

### Decision

We will add an optional `engine.mcp.session-timeout` frontmatter field that workflow authors can use to declare the MCP gateway session lifetime required by their workflow. The compiler will parse the value, validate it is a Go duration string in the range 5m–12h, and propagate it as `sessionTimeout` into the gateway's stdin configuration JSON. The field coexists with the existing `MCP_GATEWAY_SESSION_TIMEOUT` env-var mechanism: the per-workflow value in the stdin config takes precedence over the env var, which takes precedence over the gateway's built-in default of 6h.

### Alternatives Considered

#### Alternative 1: `MCP_GATEWAY_SESSION_TIMEOUT` env-var-only (status quo)

The existing approach relies on a single environment variable that applies uniformly to all workflows on the gateway instance. It was considered acceptable because infrastructure operators can tune it. It was not chosen as the long-term solution because it requires infrastructure-level changes (redeploy) to accommodate per-workflow requirements and cannot express that two workflows running on the same gateway need different lifetimes.

#### Alternative 2: Place the field under `sandbox.mcp` instead of `engine.mcp`

The `sandbox.mcp` sub-object already hosts gateway-level settings like `trusted-bots` and `keepalive-interval`, which are operational concerns about how the gateway behaves. Session timeout was considered for placement there too. However, `engine.mcp` was chosen because session lifetime is an engine-level concern: it describes how long the engine's connection to the MCP gateway must remain alive, not a property of the sandbox environment itself. Keeping engine-specific knobs under `engine.*` maintains a cleaner frontmatter taxonomy.

#### Alternative 3: Integer seconds field instead of a Go duration string

Using an integer (seconds) would be simpler to parse and validate. A Go duration string was chosen instead because the rest of the codebase uses `time.Duration`-compatible strings for timeout configuration, it is more readable in YAML (`4h` versus `14400`), and it aligns with the format already used by `MCP_GATEWAY_SESSION_TIMEOUT` on the gateway side.

### Consequences

#### Positive
- Workflow authors can express session lifetime requirements declaratively alongside the workflow definition, without requiring infrastructure changes.
- Long-running workflows (multi-hour migrations, large batch operations) get appropriately long sessions; short workflows can use shorter values, freeing gateway resources sooner.
- Compile-time validation (5m–12h range, valid Go duration format) surfaces misconfiguration early with actionable error messages.

#### Negative
- Adds another frontmatter field and a new `engine.mcp` sub-object, increasing frontmatter surface area.
- The three-level precedence rule (stdin config > env var > gateway default) is non-obvious and must be documented; operators may be surprised that a per-workflow value overrides their env-var setting.
- Any workflow that sets `session-timeout` to a very long value could hold gateway resources for extended periods if a run hangs or is abandoned.

#### Neutral
- Constants `MCPSessionTimeoutMin` (5m) is added to `pkg/constants`, establishing a named home for the MCP session minimum bound that future policies can reference.
- Only the kebab-case (`session-timeout`) key spelling is accepted during extraction; the JSON Schema uses `additionalProperties: false` to reject any other keys under `engine.mcp`.
- The JSON Schema for `engine_config` gains a new `mcp` object definition with `additionalProperties: false`, which prevents unrecognized MCP sub-keys from silently passing validation.

---

## Part 2 — Normative Specification (RFC 2119)

> The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this section are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### Frontmatter Field

1. The `engine.mcp.session-timeout` frontmatter field is **OPTIONAL**.
2. When present and non-empty, the value **MUST** be a valid Go duration string parseable by `time.ParseDuration`.
3. When present, the value **MUST** be greater than or equal to `5m` (as defined by `constants.MCPSessionTimeoutMin`); there is no upper bound.
4. Implementations **MUST** accept only kebab-case (`session-timeout`) for this key; camelCase is not supported.
5. Implementations **MUST NOT** accept any other spellings or sibling keys under `engine.mcp`; the `mcp` sub-object **MUST** be declared with `additionalProperties: false` in the JSON Schema.

### Validation

1. The compiler **MUST** validate `engine.mcp.session-timeout` during `ParseWorkflowFile`, before the workflow is further processed.
2. A validation error **MUST** produce a human-readable message that includes the offending value, the minimum bound, and a corrected example.
3. Workflows that omit `engine.mcp.session-timeout` or set it to an empty string **MUST** be treated as if the field were absent; no default value **SHALL** be injected by the compiler.

### Propagation

1. When `engine.mcp.session-timeout` is set to a valid non-empty value, the compiler **MUST** propagate it into `MCPGatewayRuntimeConfig.SessionTimeout`.
2. The renderer **MUST** emit `"sessionTimeout": "<value>"` inside the `gateway` JSON block when `MCPGatewayRuntimeConfig.SessionTimeout` is non-empty.
3. The renderer **MUST NOT** emit `"sessionTimeout"` when `MCPGatewayRuntimeConfig.SessionTimeout` is empty.
4. The emitted value **MUST** take precedence over the `MCP_GATEWAY_SESSION_TIMEOUT` environment variable at the gateway runtime.

### Conformance

An implementation is considered conformant with this ADR if it satisfies all **MUST** and **MUST NOT** requirements above. Failure to meet any **MUST** or **MUST NOT** requirement constitutes non-conformance.

---

*This is a DRAFT ADR generated by the [Design Decision Gate](https://github.com/github/gh-aw/actions/runs/25178770914) workflow. The PR author must review, complete, and finalize this document before the PR can merge.*
