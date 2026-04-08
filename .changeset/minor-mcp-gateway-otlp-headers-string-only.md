---
"gh-aw": minor
---

Changed MCP gateway OpenTelemetry `headers` configuration to accept only a string value and pass it through unchanged.

## Codemod

If you currently configure OTLP headers as an object in workflow frontmatter:

```yaml
mcp-gateway:
  opentelemetry:
    headers:
      Authorization: "Bearer ${OTLP_TOKEN}"
      X-Scope-OrgID: "my-tenant"
```

Update it to a string:

```yaml
mcp-gateway:
  opentelemetry:
    headers: "Authorization=Bearer ${OTLP_TOKEN},X-Scope-OrgID=my-tenant"
```

This applies to workflows using the `mcp-gateway.opentelemetry.headers` setting.
