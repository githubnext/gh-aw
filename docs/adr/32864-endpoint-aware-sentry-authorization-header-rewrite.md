# ADR-32864: Endpoint-Aware Sentry Authorization Header Rewrite

**Date**: 2026-05-17
**Status**: Draft
**Deciders**: Unknown

---

## Part 1 — Narrative (Human-Friendly)

### Context

Workflows fan out OTLP telemetry to multiple backends — at minimum Sentry and Grafana Cloud — using the same `shared/otlp.md` import. Sentry's OTLP ingest endpoint authenticates with a non-standard header name (`x-sentry-auth`) rather than the standard `Authorization` header used by Grafana and most other OTLP backends. Previously, a separate `shared/otel.md` import existed to paper over this difference, but it duplicated configuration surface and forced consumers to provision two parallel sets of header secrets (`*_HEADERS`) per backend. We want a single, uniform secret convention — `Authorization` — that still produces vendor-correct outbound headers.

### Decision

We will normalize OTLP headers per endpoint: when an endpoint URL is recognized as a Sentry endpoint, the compiler rewrites any `Authorization` header key to `x-sentry-auth` while leaving the header value unchanged; for all other endpoints, header names pass through unmodified. Sentry recognition is endpoint-URL-based — the hostname (or, for unresolvable `${{ secrets.* }}` expressions, the raw string) is checked for the substring `sentry`. This collapses the previous `shared/otel.md` shim into `shared/otlp.md` and lets workflows author one `*_AUTHORIZATION` secret per backend with consistent semantics.

### Alternatives Considered

#### Alternative 1: Require workflow authors to set `x-sentry-auth` explicitly

Authors could write the correct header name themselves on Sentry endpoints (e.g., `headers: {x-sentry-auth: "${{ secrets.SENTRY_AUTH }}"}`). This is the most explicit option and avoids any heuristic detection. It was rejected because it pushes a vendor-specific quirk onto every consumer of the shared import, breaks the principle that `shared/otlp.md` should be vendor-neutral, and makes mistakes (silent auth failures) easy.

#### Alternative 2: Keep `shared/otel.md` as a thin Sentry-specific wrapper

The pre-existing `shared/otel.md` imported `shared/otlp.md` and supplied vendor-specific header handling. Retaining it would preserve backward compatibility for any caller still importing it. Rejected because it duplicates the secret surface (`*_HEADERS` vs `*_AUTHORIZATION`), forces every consumer to know which import to pick, and leaves the underlying Authorization-vs-x-sentry-auth quirk unfixed for direct callers of `shared/otlp.md`.

#### Alternative 3: Per-endpoint `vendor:` discriminator field

Add an explicit `vendor: sentry` field to each endpoint object so the compiler dispatches on a declared tag rather than substring-matching the URL. Cleaner and avoids the heuristic, but requires every workflow author to learn and remember the field, adds a new schema, and provides no upgrade path for existing endpoint URLs that are already self-identifying via hostname.

### Consequences

#### Positive
- Workflow authors author a single `Authorization`-shaped secret per backend (`GH_AW_OTEL_SENTRY_AUTHORIZATION`, `GH_AW_OTEL_GRAFANA_AUTHORIZATION`), with consistent semantics across vendors.
- Removes `shared/otel.md`, collapsing the OTLP setup to one shared import.
- Sentry-vs-Grafana behavior is unit-tested in `observability_otlp_test.go`, covering both raw URLs and `${{ secrets.* }}` expressions.

#### Negative
- Substring-based detection is heuristic: a non-Sentry hostname containing the literal token `sentry` would be silently rewritten.
- For `${{ secrets.X }}` endpoints whose value cannot be resolved at compile time, detection relies on the secret name containing `sentry`. Renaming the secret to something without `sentry` in it would silently disable the rewrite — a foot-gun the user only finds out about when telemetry stops ingesting.
- Adds vendor-specific branching inside an otherwise vendor-neutral OTLP layer (`normalizeOTLPHeadersForEndpoint`, `shouldRewriteAuthorizationForSentry`).

#### Neutral
- `normalizeOTLPHeaders` is preserved as a thin wrapper over `normalizeOTLPHeadersForEndpoint(raw, "")`, so existing callers that don't have an endpoint context still compile.
- All call sites that build endpoint entries (`collectAllOTLPEndpoints`, `injectOTLPConfig`) now thread the endpoint URL through to header normalization.

---

## Part 2 — Normative Specification (RFC 2119)

> The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this section are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### Header Normalization

1. When normalizing OTLP headers, the compiler **MUST** consider the endpoint URL associated with those headers.
2. When the endpoint is detected as a Sentry endpoint, the compiler **MUST** rewrite header keys matching `Authorization` (case-insensitive) to `x-sentry-auth`.
3. When the endpoint is detected as a Sentry endpoint, the compiler **MUST NOT** modify the header value associated with the `Authorization` key.
4. When the endpoint is not detected as a Sentry endpoint, the compiler **MUST NOT** rewrite any header key.
5. The compiler **MUST NOT** modify non-`Authorization` header keys regardless of endpoint detection.

### Sentry Endpoint Detection

1. An endpoint **MUST** be classified as a Sentry endpoint if its URL parses successfully and the lowercase hostname contains the substring `sentry`.
2. An endpoint **MUST** be classified as a Sentry endpoint if its URL does not parse to a resolvable hostname but its trimmed, lowercased string form contains the substring `sentry` (covering `${{ secrets.* }}` expressions).
3. An empty or whitespace-only endpoint string **MUST NOT** be classified as a Sentry endpoint.

### Shared Import Surface

1. The `shared/otlp.md` import **MUST** be the sole shared import for OTLP backend configuration in this repository.
2. Consumers of `shared/otlp.md` **MUST** supply `GH_AW_OTEL_SENTRY_AUTHORIZATION` and `GH_AW_OTEL_GRAFANA_AUTHORIZATION` as the `Authorization` header value for the respective backend.
3. New workflows **MUST NOT** reintroduce a `shared/otel.md` import or any equivalent Sentry-only header shim.

### Conformance

An implementation is considered conformant with this ADR if it satisfies all **MUST** and **MUST NOT** requirements above. Failure to meet any **MUST** or **MUST NOT** requirement constitutes non-conformance.

---

*This is a DRAFT ADR generated by the [Design Decision Gate](https://github.com/github/gh-aw/actions/runs/25998194024) workflow. The PR author must review, complete, and finalize this document before the PR can merge.*
