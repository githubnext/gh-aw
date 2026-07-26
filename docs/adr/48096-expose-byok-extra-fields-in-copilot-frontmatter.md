# ADR-48096: Expose BYOK extraHeaders, extraBodyFields, and sessionId in Copilot Frontmatter

**Date**: 2026-07-26
**Status**: Draft
**Deciders**: pelikhan, copilot-swe-agent

---

### Context

The AWF proxy sidecar already accepts three Copilot BYOK-specific injection parameters — `AWF_BYOK_EXTRA_HEADERS`, `AWF_BYOK_EXTRA_BODY_FIELDS`, and `AWF_PROVIDER_SESSION_ID` — that let operators append custom HTTP headers, JSON body fields, and a session identifier to upstream Copilot BYOK requests. gh-aw provided no frontmatter surface for these parameters, so workflow authors who needed them (e.g., to route through OpenRouter with required `x-openrouter-title` or `http-referer` headers) had to hand-edit the generated AWF JSON config. This hand-edited config was silently overwritten on every `gh aw compile`, making the workaround fragile and unsupported.

### Decision

We will expose `extraHeaders`, `extraBodyFields`, and `sessionId` as explicit fields in the `sandbox.agent.targets.copilot` frontmatter section (`AgentAPIProxyTargetConfig`), and wire them through `BuildAWFConfigJSON` into the `apiProxy.targets.copilot` section of the generated AWF config. A dedicated `extractCopilotTargetConfig` helper (mirroring `extractAPITargetAuthHeader`) reads these fields from the parsed frontmatter and merges them into an existing or newly created copilot target entry. `sessionId` is opt-in only and is never auto-derived from `GITHUB_RUN_ID`, because strict OpenAI-compatible upstreams (e.g., Azure OpenAI) reject the unknown `session_id` body field with HTTP 400.

### Alternatives Considered

#### Alternative 1: Expose fields as top-level engine config rather than under `sandbox.agent.targets.copilot`

Placing `extraHeaders` and `extraBodyFields` directly on the `engine` or `sandbox.agent` frontmatter block would have been a shorter path. This was rejected because it would be inconsistent with the established convention: per-provider API proxy overrides already live under `sandbox.agent.targets.<provider>`, and `authHeader` already follows this pattern for OpenAI and Anthropic targets. Deviating would have fragmented the per-provider config surface.

#### Alternative 2: Auto-derive `sessionId` from `GITHUB_RUN_ID` at compile time

Automatically populating `sessionId` from the `GITHUB_RUN_ID` environment variable would reduce boilerplate for tracking purposes. This was rejected because strict OpenAI-compatible servers (including Azure OpenAI, a primary BYOK target) reject requests containing an unknown `session_id` body field with HTTP 400. Auto-derivation would silently break BYOK configurations that work today. Opt-in is the only safe default.

### Consequences

#### Positive
- Workflow authors targeting OpenRouter-style BYOK providers can configure required routing headers (`x-openrouter-title`, `http-referer`) and custom body fields declaratively in frontmatter, without post-compile JSON edits that would be overwritten.
- The implementation follows existing patterns (`extractAPITargetAuthHeader` → `extractCopilotTargetConfig`), keeping the codebase consistent and the approach auditable against prior art.

#### Negative
- `ExtraHeaders`, `ExtraBodyFields`, and `SessionId` are added to the generic `AWFAPITargetConfig` struct, but are semantically valid only for the `copilot` provider target. Code comments call this out, but misapplication to other providers (e.g., `openai`, `anthropic`) would be silently ignored rather than rejected.
- `sessionId` requires explicit opt-in with the full expression (e.g., `${{ github.run_id }}`). Authors wanting automatic session correlation must add it manually; there is no convenience default.

#### Neutral
- The sidecar env vars (`AWF_BYOK_EXTRA_HEADERS`, `AWF_BYOK_EXTRA_BODY_FIELDS`, `AWF_PROVIDER_SESSION_ID`) already existed in the AWF proxy sidecar; this PR is a pure frontmatter-to-AWF-config wiring change with no sidecar modifications.
- The spec drift table (`specs/awf-config-sources-spec.md`) now tracks the three new mappings, extending the existing pattern for `apiProxy.targets.*.authHeader`.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
