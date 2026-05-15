# ADR-32280: Pass `OTEL_EXPORTER_OTLP_HEADERS` as a Container Env Var Instead of Embedding in Gateway JSON Config

**Date**: 2026-05-15
**Status**: Draft
**Deciders**: lpcox

---

## Part 1 — Narrative (Human-Friendly)

### Context

The MCP gateway (`mcpg`) container is configured by writing a JSON document to its stdin pipe at container start. Previously, OTLP authentication headers were materialized into that JSON document as a `"headers": "${OTEL_EXPORTER_OTLP_HEADERS}"` field inside the `opentelemetry` config block. This placed live credentials inside the JSON config pipe — a surface where they could be captured by shell tracing, container logs, or future diagnostics dumps — and the variable was not forwarded into the container's environment, so `mcpg` itself never saw it. The downstream effect was that OTLP collectors requiring authentication (Sentry, Datadog, Grafana Cloud) returned `401 Unauthorized` while operators saw the header expression rendered into the config and assumed it had been applied.

### Decision

We will forward `OTEL_EXPORTER_OTLP_HEADERS` to the `mcpg` container as a `docker run -e OTEL_EXPORTER_OTLP_HEADERS` flag whenever `OTLPEndpoint != ""`, following the same pattern already used for `GITHUB_AW_OTEL_TRACE_ID` and `GITHUB_AW_OTEL_PARENT_SPAN_ID`. The `"headers"` field is removed from the rendered `opentelemetry` JSON config block and from both copies of `mcp-gateway-config.schema.json`. `mcpg` reads `OTEL_EXPORTER_OTLP_HEADERS` directly as the standard OTel SDK mechanism for OTLP authentication, so credentials no longer transit the JSON config pipe.

### Alternatives Considered

#### Alternative 1: Keep Headers in the JSON Config Pipe (Status Quo)

Retain the `"headers": "${OTEL_EXPORTER_OTLP_HEADERS}"` line in the rendered opentelemetry block and fix the 401 by teaching the gateway to honor that JSON field. Rejected because it keeps credentials flowing through the stdin JSON pipe — a surface that is harder to audit than container env vars and that diverges from the OTel SDK's standard `OTEL_EXPORTER_OTLP_HEADERS` discovery path. It would also require new bespoke parsing logic in `mcpg` that duplicates what the OTel SDK already does for free.

#### Alternative 2: Mount Headers via a Secret File

Write the headers to a temporary file and mount it into the container (e.g., `--mount type=bind,source=/tmp/otlp-headers,target=/run/secrets/otlp-headers`), then point `mcpg` at the path. Rejected because it introduces a new filesystem artifact that must be created, secured, and cleaned up, while a `-e` flag achieves the same isolation from the JSON config pipe with no additional moving parts. It would also not align with the existing pattern used for sibling OTel variables (`GITHUB_AW_OTEL_TRACE_ID`, `GITHUB_AW_OTEL_PARENT_SPAN_ID`).

#### Alternative 3: Keep Both JSON Field and Env Var (Belt-and-Suspenders)

Pass the headers via both the JSON config and the container env var. Rejected because it doubles the credential surface, leaves the original leak path in place, and creates ambiguity about which source `mcpg` should treat as authoritative.

### Consequences

#### Positive
- Credentials no longer transit the stdin JSON config pipe, reducing the surface where they could be captured by logs, shell tracing, or future diagnostics tooling.
- OTLP authentication actually works end-to-end against collectors like Sentry, Datadog, and Grafana Cloud — `mcpg` now receives the env var through the standard OTel SDK discovery mechanism.
- The forwarding pattern is consistent with sibling variables (`GITHUB_AW_OTEL_TRACE_ID`, `GITHUB_AW_OTEL_PARENT_SPAN_ID`), reducing cognitive load for future maintainers.

#### Negative
- `headers` is removed from the published `mcp-gateway-config.schema.json` — downstream tooling that referenced or validated against that property will need to drop the field.
- Operators who previously relied on inspecting the rendered JSON config to confirm header values can no longer do so; verification must move to `docker inspect` / container env inspection.
- The change is observable in the rendered gateway command (extra `-e` flag), so any golden-file or snapshot tests outside this repo that pin the exact docker invocation will need updates.

#### Neutral
- The `OTLPHeaders` workflow field is unchanged at the author level — the only difference is how the value reaches `mcpg`.
- Empty / unset `OTEL_EXPORTER_OTLP_HEADERS` is forwarded as an empty env var, which `mcpg` and the OTel SDK already treat as "no auth headers" — no special case is needed in the generator.
- `OTEL_EXPORTER_OTLP_HEADERS` is registered in `buildAddedGatewayEnvVarSet` so that the HTTP MCP env deduplication path does not emit a duplicate `-e` flag.

---

## Part 2 — Normative Specification (RFC 2119)

> The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this section are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### Gateway Container Environment Forwarding

1. When `workflowData.OTLPEndpoint != ""`, the compiled `docker run` invocation for the `mcpg` container **MUST** include a `-e OTEL_EXPORTER_OTLP_HEADERS` flag.
2. When `workflowData.OTLPEndpoint == ""`, the compiled `docker run` invocation **MUST NOT** include a `-e OTEL_EXPORTER_OTLP_HEADERS` flag.
3. The forwarded env var **MUST** be emitted as a name-only `-e` flag (delegating value resolution to the host shell), consistent with the treatment of `GITHUB_AW_OTEL_TRACE_ID`.
4. `OTEL_EXPORTER_OTLP_HEADERS` **MUST** be registered in the added-gateway-env-var set so that downstream HTTP MCP env deduplication does not emit a duplicate `-e` flag for it.

### Gateway JSON Config

1. The rendered `opentelemetry` block in the gateway JSON config **MUST NOT** contain a `headers` key.
2. Implementations **MUST NOT** inject OTLP authentication credentials into any field of the stdin JSON config pipe.
3. The `opentelemetryConfig` definition in `mcp-gateway-config.schema.json` (both `pkg/workflow/schemas/` and `docs/public/schemas/` copies) **MUST NOT** define a `headers` property.

### Authentication Source of Truth

1. `mcpg` **MUST** obtain OTLP authentication headers exclusively from the `OTEL_EXPORTER_OTLP_HEADERS` environment variable, using the standard OTel SDK discovery path.
2. Implementations **MUST NOT** introduce additional channels (config files, alternative env vars, JSON fields) for supplying OTLP headers without superseding this ADR.

### Test Coverage

1. A test **MUST** assert that the substring `"headers"` does not appear in the rendered gateway JSON config when `OTLPHeaders` is set.
2. A test **MUST** assert that `-e OTEL_EXPORTER_OTLP_HEADERS` appears in the compiled docker command when `observability.otlp.headers` is configured.
3. A test **MUST** assert that `-e OTEL_EXPORTER_OTLP_HEADERS` is absent from the compiled docker command when no OTLP endpoint is configured.

### Conformance

An implementation is considered conformant with this ADR if it satisfies all **MUST** and **MUST NOT** requirements above. Failure to meet any **MUST** or **MUST NOT** requirement constitutes non-conformance.

---

*This is a DRAFT ADR generated by the [Design Decision Gate](https://github.com/github/gh-aw/actions/runs/25899764158) workflow. The PR author must review, complete, and finalize this document before the PR can merge.*
