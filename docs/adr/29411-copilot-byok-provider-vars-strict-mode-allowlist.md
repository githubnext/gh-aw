# ADR-29411: Allowlist COPILOT_PROVIDER_* Credentials Unconditionally in Copilot Engine Strict-Mode Validation

**Date**: 2026-05-01
**Status**: Draft
**Deciders**: pelikhan, copilot-swe-agent

---

## Part 1 — Narrative (Human-Friendly)

### Context

The Copilot engine's strict-mode validation rejects any `${{ secrets.* }}` reference placed in `engine.env` unless the variable name appears in the engine's `GetRequiredSecretNames` allowlist. Before this change, the three COPILOT_PROVIDER_* variables that carry external-LLM credentials (`COPILOT_PROVIDER_BASE_URL`, `COPILOT_PROVIDER_API_KEY`, `COPILOT_PROVIDER_BEARER_TOKEN`) were absent from that allowlist. This made it impossible for users to configure Copilot Bring Your Own Key (BYOK) mode — routing requests to an external LLM provider such as OpenAI, Anthropic, or a local Ollama/vLLM instance — without triggering strict-mode errors or leaking credentials. The feature was also entirely undocumented.

### Decision

We will add `COPILOT_PROVIDER_BASE_URL`, `COPILOT_PROVIDER_API_KEY`, and `COPILOT_PROVIDER_BEARER_TOKEN` unconditionally to `GetRequiredSecretNames` in `CopilotEngine`, regardless of whether those variables appear in the current workflow's `EngineConfig.Env`. This allows strict-mode validation and `FilterEnvForSecrets` to pass their values through to the execution step. The three variables are named with Go constants in `engine_constants.go` to avoid string duplication. We chose unconditional registration because the security boundary is the AWF API proxy sidecar, not the allowlist: these credentials never reach the agent container regardless, so requiring presence in the workflow config to add them would add complexity without a corresponding security gain.

### Alternatives Considered

#### Alternative 1: Conditional Registration (Check Workflow Config First)

Only add the BYOK variables to `GetRequiredSecretNames` when they are actually present in `EngineConfig.Env` for the current workflow. This was the most obvious alternative and was explicitly considered in the code comment. It was rejected because it adds parsing/lookup complexity while providing no security benefit: the AWF proxy sidecar isolates credentials from the agent container regardless of whether the variables are pre-registered. Requiring the presence check also couples the allowlist logic to the data model unnecessarily.

#### Alternative 2: Prefix-Based Allowlist (Allow All `COPILOT_PROVIDER_*`)

Instead of enumerating the three credential-bearing variables by name, use a prefix match to allowlist any `COPILOT_PROVIDER_*` variable. This would future-proof the allowlist against new credential variable additions. It was not chosen because it would silently permit future `COPILOT_PROVIDER_*` variables — including non-credential ones like `COPILOT_PROVIDER_TYPE` or `COPILOT_PROVIDER_WIRE_API` — to carry `${{ secrets.* }}` references without review, expanding the permitted secret surface in a non-obvious way.

### Consequences

#### Positive
- BYOK mode is now fully functional under strict-mode validation; users can pass `${{ secrets.* }}` references for provider credentials in `engine.env` without errors or warnings.
- Three named Go constants (`CopilotProviderBaseURL`, `CopilotProviderAPIKey`, `CopilotProviderBearerToken`) serve as the canonical, single-source-of-truth for these variable names across the codebase.
- Comprehensive test coverage (five new cases) verifies each BYOK credential variable individually, together, and in combination with unrelated secrets.
- User-facing documentation (`engines.md`) now describes BYOK mode, the full variable reference table, and usage examples for OpenAI-compatible and Anthropic providers.

#### Negative
- The three BYOK variable names are unconditionally included in the required secrets list even when the workflow does not use BYOK mode, which marginally over-expands the allowed secret surface for non-BYOK workflows.
- Adding a new COPILOT_PROVIDER_* credential variable in the future requires an explicit code change to `GetRequiredSecretNames` — the design is not self-extending.

#### Neutral
- `GetRequiredSecretNames` now has an inline comment block explaining the unconditional registration rationale, establishing a precedent for documenting allowlist entries at the declaration site.
- Consumers of `FilterEnvForSecrets` downstream of `GetRequiredSecretNames` will transparently pass BYOK credentials through to the execution step without changes.

---

## Part 2 — Normative Specification (RFC 2119)

> The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this section are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### Secret Allowlist Registration

1. The `CopilotEngine.GetRequiredSecretNames` implementation **MUST** include `COPILOT_PROVIDER_BASE_URL`, `COPILOT_PROVIDER_API_KEY`, and `COPILOT_PROVIDER_BEARER_TOKEN` in its returned slice unconditionally — regardless of whether the current workflow's `EngineConfig.Env` contains those variables.
2. These three variable names **MUST** be referenced via the Go constants `CopilotProviderBaseURL`, `CopilotProviderAPIKey`, and `CopilotProviderBearerToken` defined in `pkg/constants/engine_constants.go`; implementations **MUST NOT** inline the string literals directly.
3. Future COPILOT_PROVIDER_* variables that carry secrets **MUST** be added to the allowlist by name as additional named constants; prefix-based matching **MUST NOT** be used.

### Strict-Mode Validation

1. When strict-mode is active, `${{ secrets.* }}` references in `engine.env` for `COPILOT_PROVIDER_BASE_URL`, `COPILOT_PROVIDER_API_KEY`, or `COPILOT_PROVIDER_BEARER_TOKEN` **MUST** pass validation without error.
2. When strict-mode is active, `${{ secrets.* }}` references for variable names not present in `GetRequiredSecretNames` **MUST** produce a validation error, even when BYOK variables are simultaneously present in the same `engine.env` block.
3. Non-credential `COPILOT_PROVIDER_*` variables (e.g., `COPILOT_PROVIDER_TYPE`, `COPILOT_PROVIDER_WIRE_API`) **SHOULD** be set as plain string values; they **MAY** carry `${{ secrets.* }}` references only if the workflow author explicitly requires confidentiality, but this is not required.

### Secret Isolation

1. BYOK provider credentials passed via `COPILOT_PROVIDER_*` variables **MUST NOT** be forwarded to the agent container process; they **MUST** remain isolated in the AWF API proxy sidecar.
2. The real provider credential **MUST NOT** be accessible to code running inside the agent container; only the dummy API key that activates the AWF BYOK detection path **MAY** be visible to the agent process.

### Conformance

An implementation is considered conformant with this ADR if it satisfies all **MUST** and **MUST NOT** requirements above. Failure to meet any **MUST** or **MUST NOT** requirement constitutes non-conformance.

---

*This is a DRAFT ADR generated by the [Design Decision Gate](https://github.com/github/gh-aw/actions/runs/25196318154) workflow. The PR author must review, complete, and finalize this document before the PR can merge.*
