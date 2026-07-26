# ADR-48185: Google Vertex AI Workload Identity Federation Auth for Gemini Engine

**Date**: 2026-07-26
**Status**: Accepted
**Deciders**: Gemini engine maintainers

---

### Context

Enterprise Gemini workloads on Google Vertex AI and the Gemini Enterprise Agent Platform require keyless
authentication rather than a static `GEMINI_API_KEY` long-lived secret. Storing long-lived API keys in
GitHub repository secrets creates a credential-rotation burden and a higher blast radius on compromise.
The `engine.auth` mechanism already supports a `type: github-oidc` + `provider: <name>` pattern for
Anthropic WIF (Claude engine), establishing a precedent for exchanging short-lived GitHub OIDC tokens
for provider-specific credentials. The goal is to give Gemini the same keyless alternative without
disrupting existing `GEMINI_API_KEY`-based workflows.

### Decision

We add a `provider: gcp` discriminator to `engine.auth` that enables Google Cloud Workload
Identity Federation (WIF) for the Gemini engine. When `type: github-oidc` and `provider: gcp` are
set with the three required fields (`workload-identity-provider`, `service-account`, `project`), the
compiler skips the `GEMINI_API_KEY` secret requirement, suppresses static-key validation, switches
the Gemini CLI to the Vertex AI backend (`GOOGLE_GENAI_USE_VERTEXAI=true`), and emits
`AWF_AUTH_GCP_WORKLOAD_IDENTITY_PROVIDER` and `AWF_AUTH_GCP_SERVICE_ACCOUNT` environment variables
for the AWF api-proxy sidecar. The `GOOGLE_CLOUD_PROJECT` and `GOOGLE_CLOUD_LOCATION` vars are set
directly for the Gemini CLI (defaulting `location` to `us-central1`). The Vertex AI env vars are
applied after any `engine.env` merge so they cannot be overridden by users. The feature is
implemented by extending `EngineAuthConfig` with four new fields (`GoogleWorkloadIdentityProvider`,
`GoogleServiceAccount`, `GoogleProject`, `GoogleLocation`), parsing them in `engine_config_parser.go`,
adding `validateGCPWIFEngineAuth()` for compile-time required-field validation, and branching on
`isGeminiVertexWIF()` in the Gemini engine compilation path. Existing `GEMINI_API_KEY` flows are
unchanged. The provider name `gcp` was chosen to match the AWF firewall's existing OIDC provider
contract (`AWF_AUTH_PROVIDER=gcp`).

### Alternatives Considered

#### Alternative 1: Keep `GEMINI_API_KEY` as the Only Auth Path

The Gemini engine continues to require a static API key, with no keyless option. This avoids any
change to the auth pipeline and the schema. It was not chosen because it blocks enterprise Vertex AI
workloads that prohibit long-lived secrets and cannot adopt the tool without a keyless path. The
Anthropic WIF precedent already established that a keyless alternative is the right direction for
enterprise engine adoption.

#### Alternative 2: Generic Provider Plugin / Extension Point

Introduce a fully generic WIF provider registry so any engine can declare its own WIF fields without
touching core auth structs. This would be more extensible but requires a significant refactor of the
auth pipeline, parser, and schema — far more complexity than warranted for a single new provider.
The simpler flat-field extension approach (consistent with how Anthropic and Azure WIF fields are
structured today) achieves the same goal with minimal risk and easier auditability.

### Consequences

#### Positive
- Enables enterprise Vertex AI / Gemini Enterprise workloads that require zero long-lived secrets in
  the repository.
- Consistent with the existing `type: github-oidc` + `provider:` discriminator pattern for Anthropic
  WIF, reducing conceptual surface area for users already familiar with Claude engine keyless auth.
- Short-lived OIDC tokens exchanged via Google Cloud WIF reduce blast radius vs. a static API key
  stored indefinitely in repo secrets.

#### Negative
- Users must pre-configure a Google Cloud Workload Identity Pool, Provider, and service account with
  appropriate Vertex AI permissions — substantially more setup than providing a single `GEMINI_API_KEY`.
- Four new fields added to `EngineAuthConfig` struct and the `engine.auth` JSON schema; the parser and
  compiler now have an additional branch (`isGeminiVertexWIF`) to maintain and test for each future
  Gemini engine change.
- The `service-account` and `project` keys in `engine.auth` are shared key names without a `gcp-`
  prefix, which could cause ambiguity if a future provider also uses these field names for different
  semantics. This is accepted as a known trade-off consistent with the existing `engine.auth` pattern.

#### Neutral
- The `location` field is optional and defaults to `us-central1` when omitted. The compile-time
  validation (`validateGCPWIFEngineAuth()`) only checks for the presence of the required fields
  (`workload-identity-provider`, `service-account`, `project`); it does not validate that the region
  string is valid, so an invalid region will fail at runtime rather than compile time.
- The AWF api-proxy sidecar handles the actual OIDC token exchange with Google Cloud; this ADR covers
  only the compiler-side changes that emit the required `AWF_AUTH_GCP_*` environment variables.

---

*ADR accepted. Provider name changed from `google` to `gcp` to match the AWF firewall OIDC contract.*
