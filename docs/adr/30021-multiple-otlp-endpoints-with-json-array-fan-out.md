# ADR-30021: Multiple OTLP Endpoints with Polymorphic `endpoint` Field and JSON-Array Fan-Out

**Date**: 2026-05-03
**Status**: Draft
**Deciders**: Unknown (copilot-swe-agent, PR #30021)

---

## Part 1 — Narrative (Human-Friendly)

### Context

The gh-aw observability pipeline exports OpenTelemetry (OTLP) spans from GitHub Actions jobs. Until this point the system supported exactly one collector endpoint, configured via a scalar `observability.otlp.endpoint` string and propagated as `OTEL_EXPORTER_OTLP_ENDPOINT` in the environment. Users operating redundant or vendor-differentiated observability stacks (e.g., primary collector + hot-standby, or two different SaaS vendors simultaneously) had no way to fan out spans to more than one destination. Because the Go compiler and the JavaScript action steps run in separate processes, a compact, self-contained serialization format is needed to pass the full endpoint list across the process boundary without requiring code changes in every consuming script.

### Decision

We will make the `observability.otlp.endpoint` field polymorphic — accepting a plain URL string (backward compat), a single `{url, headers}` object, or an array of `{url, headers}` objects — and propagate all endpoint configurations as a JSON-encoded array in the environment variable `GH_AW_OTLP_ENDPOINTS`. The separate `endpoints` array field is not introduced; a single `endpoint` field covers all three forms. JavaScript action scripts will read exclusively from `GH_AW_OTLP_ENDPOINTS` and fan out spans to every listed endpoint concurrently via `Promise.allSettled`. The legacy `OTEL_EXPORTER_OTLP_ENDPOINT` variable is still injected for backward compatibility with the MCP gateway but is no longer used by the JavaScript fan-out logic.

### Alternatives Considered

#### Alternative 1: Separate `endpoints` Array Field (alongside scalar `endpoint`)

Introducing a separate `endpoints` array field alongside the existing scalar `endpoint` was the first iteration of this feature. This dual-field approach required merging two fields at compile time and was surprising for users who expected a single, unified configuration point. It was replaced by the polymorphic single-field design.

#### Alternative 2: Separate Indexed Environment Variables (`OTEL_EXPORTER_OTLP_ENDPOINT_0`, `_1`, …)

Expanding each endpoint into an indexed set of env-vars (e.g., `OTEL_EXPORTER_OTLP_ENDPOINT_0`, `OTEL_EXPORTER_OTLP_HEADERS_0`, …) would stay closer to the OpenTelemetry SDK convention. However, it requires every consuming script to perform a scan-loop over an open-ended index range rather than a single JSON parse, and the number of variables grows linearly with the number of endpoints × the number of per-endpoint fields. This approach also makes it harder to add new per-endpoint fields in the future without updating every loop. It was not chosen because the JSON array form is simpler for the consumer side and more extensible.

#### Alternative 3: Retain OTEL_EXPORTER_OTLP_ENDPOINT as the Primary Signal and Add a Separate Multi-Endpoint Variable

A hybrid approach would keep JavaScript reading `OTEL_EXPORTER_OTLP_ENDPOINT` for the single-endpoint case and only switch to a multi-endpoint variable when more than one endpoint is configured. This preserves maximum backward compatibility but introduces divergent code paths in every consumer (always check single-endpoint first, then fall back to the multi-endpoint var, or vice-versa). It was not chosen because the divergent logic is error-prone; normalizing at the Go compiler layer and having JavaScript always read the array form is simpler and more consistent.

### Consequences

#### Positive
- Workflow authors can now send spans to multiple OTLP collectors simultaneously using the array form of `observability.otlp.endpoint`.
- A single field (`endpoint`) covers all use cases: scalar URL, single-endpoint-with-headers object, and multi-endpoint array — no separate `endpoints` field to remember.
- JavaScript action code is simplified: a single code path (`parseOTLPEndpoints` + `sendOTLPToAllEndpoints`) handles all three forms.
- All endpoint secrets are transparently masked via the expanded `GH_AW_OTLP_ALL_HEADERS` mechanism, maintaining parity with the single-endpoint masking behavior.
- Static endpoint hostnames are automatically added to the network firewall allowlist at compile time.

#### Negative
- JavaScript action scripts now unconditionally depend on `GH_AW_OTLP_ENDPOINTS` being a valid JSON array; if a third-party tool injects a malformed value the parse will fail silently (the value is absent in that case).
- `OTEL_EXPORTER_OTLP_ENDPOINT` is maintained only for MCP gateway backward compatibility. Any consumer that still reads the standard env-var directly will miss additional endpoints beyond the first.
- GitHub Actions expression-based endpoint URLs (e.g., `${{ secrets.OTLP_ENDPOINT }}`) cannot be added to the firewall allowlist at compile time; those hostnames must be manually allowlisted.

#### Neutral
- The JSON array is always injected even for single-endpoint configurations; downstream tooling must parse JSON to determine the endpoint count.
- The `GH_AW_OTLP_ALL_HEADERS` variable is only set when at least one endpoint includes headers; consumers that rely on it being present must guard against it being absent.

---

## Part 2 — Normative Specification (RFC 2119)

> The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this section are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### Frontmatter Configuration

The `observability.otlp.endpoint` field **MUST** accept three forms:

1. **String form** (backward compat): a plain URL string. The optional top-level `observability.otlp.headers` field **MAY** supply headers for this single endpoint; the compiler **MUST** associate those headers with the endpoint entry.
2. **Object form**: a single `{url, headers}` object. The `headers` key inside the object takes precedence; the top-level `observability.otlp.headers` field **MUST** be ignored when the endpoint is in object form.
3. **Array form**: a list of `{url, headers}` objects, each defining its own URL and optional per-endpoint headers. The top-level `observability.otlp.headers` field **MUST** be ignored when the endpoint is in array form.

### Environment Variable Contract

1. The Go compiler **MUST** inject `GH_AW_OTLP_ENDPOINTS` as a JSON-encoded array of endpoint objects for every workflow that contains at least one OTLP endpoint configuration, regardless of which form is used.
2. The Go compiler **MUST** continue to inject `OTEL_EXPORTER_OTLP_ENDPOINT` (set to the first endpoint URL) for backward compatibility with the MCP gateway; this variable **MUST NOT** be used as the primary source of endpoint configuration by any new consumer.
3. The Go compiler **MUST** inject `GH_AW_OTLP_ALL_HEADERS` (containing the combined headers of all endpoints) when two or more endpoints include headers; this variable **MAY** be absent when zero or one endpoint has headers.
4. Static hostname values extracted from the `endpoint` field (in any form) **MUST** be added to the network firewall allowlist at compile time; expression-based URL values (`${{ … }}`) **MUST** be skipped.

### JavaScript Fan-Out

1. JavaScript action scripts **MUST** read `GH_AW_OTLP_ENDPOINTS` exclusively to determine the set of active OTLP endpoints; they **MUST NOT** fall back to `OTEL_EXPORTER_OTLP_ENDPOINT` for span-export decisions.
2. JavaScript action scripts **MUST** send each span to all configured endpoints concurrently using `Promise.allSettled`, so that a failure on one endpoint does not prevent export to others.
3. If `GH_AW_OTLP_ENDPOINTS` is absent or empty, JavaScript action scripts **MUST** skip OTLP export without error.

### Conformance

An implementation is considered conformant with this ADR if it satisfies all **MUST** and **MUST NOT** requirements above. Specifically: the Go compiler must always serialize endpoint configurations into `GH_AW_OTLP_ENDPOINTS` as a JSON array (handling all three forms of the `endpoint` field); JavaScript consumers must read exclusively from that array and fan out concurrently; and the legacy `OTEL_EXPORTER_OTLP_ENDPOINT` must remain injected for MCP gateway backward compatibility but must not be used as the primary endpoint source by new code.

---

*This is a DRAFT ADR generated by the [Design Decision Gate](https://github.com/github/gh-aw/actions/runs/25293323836) workflow. The PR author must review, complete, and finalize this document before the PR can merge.*
