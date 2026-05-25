# ADR-34450: Inject `OTEL_RESOURCE_ATTRIBUTES` for Child OTel SDK Correlation in gh-aw Workflows

**Date**: 2026-05-24
**Status**: Draft
**Deciders**: pelikhan (PR author: app/copilot-swe-agent)

---

## Part 1 — Narrative (Human-Friendly)

### Context

gh-aw workflows already export OTLP endpoint, service name, and headers to child processes (Copilot CLI, MCP gateway) via the compiled workflow `env:` block, so those processes correctly emit spans into the same trace. However, the workflow compiler did not export `OTEL_RESOURCE_ATTRIBUTES`, the standard OTel SDK env var that downstream SDKs read to populate their `Resource` block. As a result, child-process spans joined the parent trace but were missing stable gh-aw correlation keys (workflow name, repository, run id, engine id), making cross-surface correlation in the trace backend incomplete and forcing manual joins against GitHub run metadata.

### Decision

We will inject `OTEL_RESOURCE_ATTRIBUTES` into the compiled workflow `env:` block from `Compiler.injectOTLPConfig`, alongside `OTEL_EXPORTER_OTLP_ENDPOINT` and `OTEL_SERVICE_NAME`. The value is a comma-separated list of `key=value` pairs containing `gh-aw.workflow.name`, `gh-aw.repository`, `gh-aw.run.id`, `github.run_id`, and (conditionally) `gh-aw.engine.id`. Values are escaped with a centralized `strings.NewReplacer` that handles the three characters reserved by the OpenTelemetry env-var resource attribute grammar (`\`, `,`, `=`). The engine id attribute is omitted when neither `EngineConfig.ID` nor `workflowData.AI` is set, so workflows without an explicit engine do not get an empty `gh-aw.engine.id=` entry.

### Alternatives Considered

#### Alternative 1: Configure Resource Attributes Programmatically in Each Child SDK

Have each downstream emitter (Copilot CLI harness, MCP gateway, span-emitting scripts) hard-code the gh-aw resource keys in its OTel SDK initialization, reading them from individual GitHub Actions env vars. Rejected because it duplicates the same five-attribute resource block across every engine and language runtime in the project, drifts over time as each surface evolves independently, and forces every new OTel-emitting subprocess to re-implement the conventions. Centralizing the assembly in the workflow compiler means one source of truth for the gh-aw resource schema.

#### Alternative 2: Per-Step Env Injection at Each Engine Step

Inject `OTEL_RESOURCE_ATTRIBUTES` only at the steps that invoke OTel-emitting binaries (e.g. the Copilot CLI step), rather than at the workflow level. Rejected because it requires the compiler to enumerate every present and future OTel-emitting step and keep their env blocks in sync; the workflow-level `env:` block is automatically inherited by every step and matches the existing pattern used for `OTEL_EXPORTER_OTLP_ENDPOINT` and `OTEL_SERVICE_NAME`, which are already emitted at the workflow level by the same function.

#### Alternative 3: Append to Any Existing `OTEL_RESOURCE_ATTRIBUTES` Set by the User

Read any user-defined `OTEL_RESOURCE_ATTRIBUTES` from the workflow frontmatter and append gh-aw keys to it (or merge). Rejected for this initial change because gh-aw workflows do not currently expose a frontmatter-level OTel resource attribute field, and adding a merge path before there is a real user-facing input invents a contract that nothing consumes. Custom span attributes already have a dedicated path via `GH_AW_OTLP_ATTRIBUTES`; a future ADR can extend resource-level merging once a frontmatter surface for it exists.

### Consequences

#### Positive
- Every OTel SDK in a child process inherits a consistent gh-aw resource block automatically (via the standard `OTEL_RESOURCE_ATTRIBUTES` discovery path) without each subprocess needing its own initialization code.
- Trace backends can group, filter, and join spans by `gh-aw.workflow.name`, `gh-aw.repository`, `gh-aw.run.id`, and `gh-aw.engine.id` without post-hoc enrichment from GitHub Actions metadata.
- Centralized escaping in `escapeOTELResourceAttributeValue` makes the OpenTelemetry env-var grammar an enforced invariant of the compiler, not something each call site has to remember.

#### Negative
- Every compiled workflow with an OTLP endpoint now carries an extra `OTEL_RESOURCE_ATTRIBUTES` line in its `env:` block. This is a small but real surface increase in the generated YAML and shows up in any golden-file or snapshot tests that pin exact env contents.
- The attribute set is now part of the compiler's contract: renaming or removing a key (`gh-aw.workflow.name`, `gh-aw.run.id`, etc.) is a backwards-incompatible change for any downstream dashboard or alert that joins on those keys.
- Workflow names and engine ids with `\`, `,`, or `=` now flow through an extra escaping step that did not previously exist; bugs in that escaping could subtly corrupt resource attributes in the trace backend.

#### Neutral
- `gh-aw.run.id` and `github.run_id` carry the same value (`${{ github.run_id }}`); both are emitted because some dashboards key off the namespaced form and others off the unprefixed GitHub-standard form.
- `${{ github.repository }}` and `${{ github.run_id }}` are intentionally emitted as literal GitHub Actions expressions, not pre-resolved at compile time, so each run substitutes the live values.
- When neither `EngineConfig.ID` nor `workflowData.AI` is set, the resulting `OTEL_RESOURCE_ATTRIBUTES` simply omits `gh-aw.engine.id`; downstream SDKs see a four-attribute resource block rather than an attribute with an empty value.

---

## Part 2 — Normative Specification (RFC 2119)

> The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this section are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### Workflow-Level Env Injection

1. When `Compiler.injectOTLPConfig` runs against a workflow whose first OTLP endpoint is non-empty, the compiled workflow `env:` block **MUST** include an `OTEL_RESOURCE_ATTRIBUTES` entry.
2. The `OTEL_RESOURCE_ATTRIBUTES` value **MUST** be emitted at the workflow `env:` level, not solely at the step level, so that every step in the job inherits it without per-step duplication.
3. When no OTLP endpoint is configured for the workflow, the compiled `env:` block **MUST NOT** include an `OTEL_RESOURCE_ATTRIBUTES` entry sourced from `injectOTLPConfig`.

### Attribute Composition

1. The injected `OTEL_RESOURCE_ATTRIBUTES` value **MUST** include the following keys in this order: `gh-aw.workflow.name`, `gh-aw.repository`, `gh-aw.run.id`, `github.run_id`.
2. When `workflowData.Name` (after `strings.TrimSpace`) is non-empty, `gh-aw.workflow.name` **MUST** be set to the trimmed name passed through `escapeOTELResourceAttributeValue`; otherwise it **MUST** be set to the literal string `unknown`.
3. `gh-aw.repository` **MUST** be set to the literal GitHub Actions expression `${{ github.repository }}`.
4. `gh-aw.run.id` and `github.run_id` **MUST** each be set to the literal GitHub Actions expression `${{ github.run_id }}`.
5. When `resolveWorkflowEngineID(workflowData)` returns a non-empty string, an additional `gh-aw.engine.id` attribute **MUST** be appended after the four required keys, with its value passed through `escapeOTELResourceAttributeValue`.
6. When `resolveWorkflowEngineID(workflowData)` returns an empty string, the `OTEL_RESOURCE_ATTRIBUTES` value **MUST NOT** contain a `gh-aw.engine.id=` substring.
7. `resolveWorkflowEngineID` **MUST** return `workflowData.EngineConfig.ID` when it is non-empty, and **MUST** otherwise fall back to `workflowData.AI`.

### Value Escaping

1. Every dynamic attribute value (workflow name, engine id) **MUST** be passed through `escapeOTELResourceAttributeValue` before being concatenated into the `OTEL_RESOURCE_ATTRIBUTES` string.
2. `escapeOTELResourceAttributeValue` **MUST** replace `\` with `\\`, `,` with `\,`, and `=` with `\=`, in that grammar order.
3. Static attribute values that are literal GitHub Actions expressions (`${{ github.repository }}`, `${{ github.run_id }}`) **MUST NOT** be passed through `escapeOTELResourceAttributeValue`, because they are emitted verbatim for runtime substitution.
4. Implementations **MUST NOT** introduce a second, divergent escaper for OTel resource attribute values; any new call site **MUST** route through `escapeOTELResourceAttributeValue`.

### Test Coverage

1. A unit test **MUST** assert that the compiled workflow `env:` contains `OTEL_RESOURCE_ATTRIBUTES: gh-aw.workflow.name=unknown,gh-aw.repository=${{ github.repository }},gh-aw.run.id=${{ github.run_id }},github.run_id=${{ github.run_id }},gh-aw.engine.id=copilot` when `workflowData.AI == "copilot"` and no name is set.
2. A unit test **MUST** assert that, when neither `EngineConfig.ID` nor `workflowData.AI` is set, the compiled `env:` does not contain the substring `gh-aw.engine.id=`.
3. A unit test **MUST** assert that an engine id containing `\`, `,`, and `=` is rendered as `gh-aw.engine.id=copilot\,eq\=uals\\slash`.
4. A unit test **MUST** assert that a workflow name containing `\`, `,`, and `=` is rendered as `gh-aw.workflow.name=triage\,weekly\=run\\v2`.

### Conformance

An implementation is considered conformant with this ADR if it satisfies all **MUST** and **MUST NOT** requirements above. Failure to meet any **MUST** or **MUST NOT** requirement constitutes non-conformance.

---

*This is a DRAFT ADR generated by the [Design Decision Gate](https://github.com/github/gh-aw/actions/runs/26363472263) workflow. The PR author must review, complete, and finalize this document before the PR can merge.*
