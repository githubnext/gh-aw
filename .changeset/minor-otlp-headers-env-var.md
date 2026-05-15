---
"gh-aw": minor
---

Pass `OTEL_EXPORTER_OTLP_HEADERS` as a container environment variable to the mcpg container when `observability.otlp` is configured, and remove the `headers` field from the `gateway.opentelemetry` JSON config.

## What changed

- When `observability.otlp` is configured in a workflow, `OTEL_EXPORTER_OTLP_HEADERS` is now forwarded to the mcpg container via `-e OTEL_EXPORTER_OTLP_HEADERS`, following the standard [OTel SDK environment variable convention](https://opentelemetry.io/docs/specs/otel/protocol/exporter/#configuration-options).
- The `headers` field is removed from the `gateway.opentelemetry` stdin JSON config and from the MCP gateway config JSON schema. Auth credentials must not flow through the JSON config pipe.

## Why

OTLP auth headers (e.g., `Authorization: Bearer <token>`) are security-sensitive. The correct mechanism is the `OTEL_EXPORTER_OTLP_HEADERS` environment variable, which is:
- Not expanded into the stdin JSON config pipe or workflow logs where credentials could be inadvertently exposed
- The standard OTel convention used by all OTel SDKs and collectors
- Provided via standard container environment variable configuration, keeping credentials out of config artifacts

## Migration

If you previously relied on the `headers` field in your gateway config JSON (not via `observability.otlp` frontmatter), set the `OTEL_EXPORTER_OTLP_HEADERS` environment variable instead:

```
OTEL_EXPORTER_OTLP_HEADERS="Authorization=Bearer my-token,X-Custom=value"
```

For workflows using `observability.otlp.headers`, no action is required — gh-aw automatically forwards the value as the `OTEL_EXPORTER_OTLP_HEADERS` container env var.
