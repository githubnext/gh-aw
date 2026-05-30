# ADR-35694: Expose `authHeader` in `sandbox.agent.targets` Frontmatter

**Date**: 2026-05-29
**Status**: Draft
**Deciders**: Unknown

---

## Part 1 — Narrative (Human-Friendly)

### Context

The AWF firewall sidecar (PR #3998) introduced `--openai-api-auth-header` and `--anthropic-api-auth-header` flags, plus matching `apiProxy.targets.{openai,anthropic}.authHeader` fields in the AWF JSON config. These exist to support gateways such as Azure OpenAI, which require API keys to be sent as `api-key: <rawkey>` rather than the provider default (`Authorization: Bearer <key>` for OpenAI, `x-api-key: <key>` for Anthropic). The runtime capability exists in the sidecar, but `gh-aw` workflow authors had no declarative way to set it from their frontmatter — they would have had to hand-edit the generated workflow YAML, which is regenerated on every compile. The authentication-header override is independent of host overrides: a workflow may need a custom header against the standard public provider host, or against a custom host already configured via `OPENAI_BASE_URL` / `ANTHROPIC_BASE_URL`.

### Decision

We will expose `authHeader` as a frontmatter field at `sandbox.agent.targets.<provider>.authHeader` for `provider ∈ {openai, anthropic}`. The new field is read by a dedicated helper `extractAPITargetAuthHeader` (in `pkg/workflow/engine_api_targets.go`) and applied inside `BuildAWFConfigJSON` (in `pkg/workflow/awf_config.go`) by mutating the existing `AWFAPITargetConfig` entry when one is already present, or creating a header-only entry when no host override exists. The field is emitted with `omitempty` so the generated AWF JSON stays clean when it is not configured. The frontmatter path mirrors the AWF JSON config structure 1:1, preserving the drift-tracking guarantee documented in `specs/awf-config-sources-spec.md`.

### Alternatives Considered

#### Alternative 1: Top-level engine field (e.g. `engine.auth-header`)

We could attach the override to the engine config itself, similar to how `engine.api-target` works for Copilot. Rejected because the auth header is a per-target proxy concern, not an engine concern. It would not scale cleanly to per-provider configuration when a future workflow declares multiple engines, and it would diverge from the AWF JSON config layout that the rest of `apiProxy.targets.*` already follows.

#### Alternative 2: Reuse `engine.env` with new `OPENAI_API_AUTH_HEADER` / `ANTHROPIC_API_AUTH_HEADER` env vars

The compiler already reads `OPENAI_BASE_URL` and `ANTHROPIC_BASE_URL` out of `engine.env` to derive host overrides, so the same channel could carry a header name. Rejected because it conflates HTTP-header-level proxy configuration with engine runtime environment variables. The AWF JSON config groups all per-target proxy settings under `apiProxy.targets.<provider>`, and adding env-var-only overrides would break the 1:1 mapping that the drift-tracking spec relies on.

#### Alternative 3: Auto-detect Azure-style hosts and default `authHeader` to `api-key`

The compiler could inspect the host of `OPENAI_BASE_URL` and, when it matches an Azure pattern (e.g. `*.openai.azure.com`), automatically emit `authHeader: api-key`. Rejected because it would require maintaining a brittle host-pattern allowlist, would not cover non-Azure gateways with similar requirements, and would silently override user intent in edge cases. Explicit configuration is more predictable.

### Consequences

#### Positive
- Workflow authors can target Azure OpenAI (and other gateways requiring custom headers) without forking or hand-editing the generated workflow YAML.
- The schema validates `authHeader` as a string and rejects unknown providers via `additionalProperties: false`, so typos fail at compile time rather than at runtime in the AWF sidecar.
- The frontmatter path mirrors the runtime AWF JSON config 1:1, extending the drift-tracking design in `specs/awf-config-sources-spec.md` rather than introducing a new mapping convention.
- `authHeader` is independent of host overrides, so workflows that need a custom header against the public provider host (or against an already-configured host) can express that without redundant configuration.

#### Negative
- The workflow frontmatter schema grows by a new nested block (`sandbox.agent.targets.{openai,anthropic}.authHeader`); the regenerated `main_workflow_schema.json` adds ~1.9k net lines, increasing the surface that schema-based tooling must scan.
- Two truths must be kept in sync: a workflow can configure `authHeader: api-key` without setting a custom host, which is valid but easy to misuse if the public provider rejects the non-standard header.
- The new helper `extractAPITargetAuthHeader` is a per-provider lookup that traverses the same frontmatter path that future per-target fields (e.g. timeouts, retries) would also walk; we accept this duplication for now rather than building a generic per-target extractor.

#### Neutral
- `AWFAPITargetConfig` now carries `AuthHeader string` alongside `Host string`; entries created solely for the auth-header override (no host) still serialize cleanly because `Host` and `AuthHeader` both use `omitempty`.
- A `specs/awf-config-sources-spec.md` table gains two rows (`apiProxy.targets.openai.authHeader`, `apiProxy.targets.anthropic.authHeader`) to record the new frontmatter ↔ CLI-flag mapping.
- The decision deliberately scopes the feature to `openai` and `anthropic` for now; adding more providers (e.g. `copilot`, `gemini`) requires only extending the `for _, provider := range []string{"openai", "anthropic"}` slice and the schema enum.

---

## Part 2 — Normative Specification (RFC 2119)

> The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this section are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### Frontmatter Schema

1. The workflow frontmatter **MUST** accept an optional `sandbox.agent.targets.<provider>.authHeader` path that holds the custom authentication header name.
2. The set of recognized providers under `sandbox.agent.targets` **MUST** be restricted via `additionalProperties: false`; unknown provider keys **MUST** be rejected at schema-validation time.
3. The value of `authHeader` **MUST** be a string; non-string values (including numbers, booleans, arrays, and objects) **MUST** be rejected at schema-validation time.
4. Workflows **MAY** set `authHeader` without configuring a custom host for the same provider; the two settings **MUST** be independent.

### Compiler Behavior

1. `BuildAWFConfigJSON` **MUST** read `sandbox.agent.targets.<provider>.authHeader` from `WorkflowData.SandboxConfig` for each supported provider and apply it to the emitted AWF JSON config.
2. When a target entry already exists for a provider (e.g. because a custom host was configured), the compiler **MUST** mutate the existing entry in place rather than overwriting it; the host and `authHeader` fields **MUST** coexist.
3. When no target entry exists for a provider, the compiler **MUST** create a header-only entry containing only `authHeader`, leaving `host` unset.
4. The compiler **MUST NOT** emit an `authHeader` field in the AWF JSON output when the frontmatter value is absent, empty, or non-string.
5. The frontmatter-extraction helper **MUST** return an empty string (and the compiler **MUST** treat that as "not configured") when any of the following hold: `WorkflowData` is `nil`; `SandboxConfig` or `Agent` is `nil`; `Targets` is `nil`; the provider key is absent; or `authHeader` is empty.

### Drift Tracking

1. `specs/awf-config-sources-spec.md` **MUST** list every frontmatter path that maps to an AWF JSON config field or AWF CLI flag, including `sandbox.agent.targets.openai.authHeader` and `sandbox.agent.targets.anthropic.authHeader`.
2. Any future addition of a per-target proxy field to the AWF JSON config **SHOULD** be exposed via the parallel `sandbox.agent.targets.<provider>.<field>` frontmatter path and **MUST** be recorded in the drift-tracking table.

### Conformance

An implementation is considered conformant with this ADR if it satisfies all **MUST** and **MUST NOT** requirements above. Failure to meet any **MUST** or **MUST NOT** requirement constitutes non-conformance.

---

*This is a DRAFT ADR generated by the [Design Decision Gate](https://github.com/github/gh-aw/actions/runs/26638539097) workflow. The PR author must review, complete, and finalize this document before the PR can merge.*
