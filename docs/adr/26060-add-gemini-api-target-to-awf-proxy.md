# ADR-26060: Add Gemini API Target Routing to AWF Proxy

**Date**: 2026-04-13
**Status**: Draft
**Deciders**: pelikhan, Copilot

---

## Part 1 — Narrative (Human-Friendly)

### Context

The AWF proxy sidecar provides an LLM gateway that routes API requests from agent containers to external AI providers. For OpenAI (codex), Anthropic (claude), and Copilot engines, the proxy has built-in default routing targets. Gemini was integrated as an engine but never received a corresponding proxy routing target: when a workflow runs with `engine: gemini` and the network firewall enabled, `GEMINI_API_BASE_URL` points at the proxy on port 10003, but the proxy cannot forward the request and returns `API_KEY_INVALID`. The fix must follow the existing pattern for other engines to stay consistent and maintainable.

### Decision

We will add `GetGeminiAPITarget()` to the AWF helpers layer and wire it into `BuildAWFArgs()` so that the `--gemini-api-target` flag is emitted whenever the engine is Gemini. The default target is `generativelanguage.googleapis.com`; when `GEMINI_API_BASE_URL` is set in `engine.env`, the hostname extracted from that URL takes precedence. When the custom URL includes a path component, `--gemini-api-base-path` is also emitted. This mirrors the existing pattern used for `--openai-api-target`, `--anthropic-api-target`, and `--copilot-api-target`, keeping the engine routing model uniform.

### Alternatives Considered

#### Alternative 1: Hard-code the Gemini default target inside the AWF sidecar binary

The AWF sidecar could be patched to know about Gemini's default endpoint without requiring the caller to pass `--gemini-api-target`. This would eliminate the need for the go-layer change. However, it couples the sidecar to a specific vendor endpoint, making it harder to test independently and requiring a sidecar release for every new engine. The current pattern—caller-supplied targets—keeps the sidecar generic.

#### Alternative 2: Require users to always set `GEMINI_API_BASE_URL` explicitly

Without a default target, users who want to use the public Gemini endpoint would need to add `GEMINI_API_BASE_URL: "https://generativelanguage.googleapis.com"` to every workflow. This adds boilerplate and differs from every other engine, which all route to a sensible default without extra configuration. The experience asymmetry is a significant usability cost.

#### Alternative 3: Use `engine.api-target` YAML field instead of an environment variable

The Copilot engine already has an `engine.api-target` field in the workflow YAML that overrides `GITHUB_COPILOT_BASE_URL`. We could introduce a similar `engine.api-target` for Gemini. However, no other engine besides Copilot uses this field, and adding it only for Gemini would create inconsistency. Using `GEMINI_API_BASE_URL` in `engine.env` aligns Gemini with the codex and claude pattern.

### Consequences

#### Positive
- Gemini engine workflows now work correctly when the network firewall is enabled — the proxy can forward requests to the correct upstream.
- Users get custom endpoint support (`GEMINI_API_BASE_URL`) consistent with the codex and claude engines.
- The implementation follows the established engine-routing pattern; new engines in the future can be added the same way.
- `GH_AW_ALLOWED_DOMAINS` is kept in sync with `--allow-domains` via the existing `computeAllowedDomainsForSanitization` hook.

#### Negative
- `BuildAWFArgs` grows slightly larger; the engine-specific target logic is co-located in one function rather than being dispatched polymorphically.
- A hard-coded constant (`DefaultGeminiAPITarget`) must be updated if Google changes the Gemini API hostname, though this is an unlikely scenario.

#### Neutral
- The smoke-test lock file (`.github/workflows/smoke-gemini.lock.yml`) must be recompiled to include `--gemini-api-target generativelanguage.googleapis.com` in generated `awf` invocations.
- Documentation for custom API endpoints in `docs/src/content/docs/reference/engines.md` gains a Gemini example section, extending an existing pattern rather than introducing new concepts.

---

## Part 2 — Normative Specification (RFC 2119)

> The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this section are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### Gemini Proxy Target Resolution

1. When the active engine is `gemini` and `GEMINI_API_BASE_URL` is not set in `engine.env`, implementations **MUST** emit `--gemini-api-target generativelanguage.googleapis.com` in the `awf` command arguments.
2. When `GEMINI_API_BASE_URL` is set in `engine.env`, implementations **MUST** extract the hostname from that URL and emit `--gemini-api-target <hostname>` instead of the default.
3. When `GEMINI_API_BASE_URL` contains a non-empty path component (e.g. `/v1/beta`), implementations **MUST** also emit `--gemini-api-base-path <path>`.
4. Implementations **MUST NOT** emit `--gemini-api-target` when the engine is not `gemini` and `GEMINI_API_BASE_URL` is not configured.
5. The `DefaultGeminiAPITarget` constant **SHOULD** be the single source of truth for the default Gemini hostname; it **MUST NOT** be duplicated as a string literal elsewhere in the codebase.

### Domain Allowlist Synchronization

1. The effective Gemini API target hostname **MUST** be included in the domain set computed by `computeAllowedDomainsForSanitization()` so that `GH_AW_ALLOWED_DOMAINS` and `--allow-domains` remain consistent.
2. Implementations **MUST** call `GetGeminiAPITarget()` with the same `engineID` used for the proxy flag, ensuring both paths resolve identically.

### Custom Endpoint Pattern

1. New engine API-target integrations **SHOULD** follow the same three-part pattern established here: (a) a `Get<Engine>APITarget()` helper that reads `<ENGINE>_API_BASE_URL` with a default fallback, (b) a call in `BuildAWFArgs()` to emit the `--<engine>-api-target` flag, and (c) inclusion in `computeAllowedDomainsForSanitization()`.
2. Engine-specific environment variables for custom endpoints **MUST** follow the naming convention `<ENGINE_UPPERCASE>_API_BASE_URL` (e.g. `GEMINI_API_BASE_URL`, `OPENAI_BASE_URL`).

### Conformance

An implementation is considered conformant with this ADR if it satisfies all **MUST** and **MUST NOT** requirements above. Failure to meet any **MUST** or **MUST NOT** requirement constitutes non-conformance.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
