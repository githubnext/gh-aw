---
network:
  allowed:
    - "*.sentry.io"
    - "*.grafana.net"
observability:
  otlp:
    endpoint:
      - url: ${{ secrets.GH_AW_OTEL_SENTRY_ENDPOINT }}
        headers:
          Authorization: ${{ secrets.GH_AW_OTEL_SENTRY_AUTHORIZATION }}
      - url: ${{ secrets.GH_AW_OTEL_GRAFANA_ENDPOINT }}
        headers:
          Authorization: ${{ secrets.GH_AW_OTEL_GRAFANA_AUTHORIZATION }}
---

## Required secrets

Consumers of this shared import must provision the following secrets:

- `GH_AW_OTEL_SENTRY_ENDPOINT`
- `GH_AW_OTEL_SENTRY_AUTHORIZATION` - authored as the `Authorization` value; gh-aw rewrites it to `x-sentry-auth` for Sentry endpoints. Set this secret to the header value only, typically `Sentry sentry_version=7, sentry_key=<key>, sentry_client=gh-aw`, where `sentry_version=7`, `sentry_key=<key>`, and `sentry_client=gh-aw` are the Sentry auth components.
- `GH_AW_OTEL_GRAFANA_ENDPOINT`
- `GH_AW_OTEL_GRAFANA_AUTHORIZATION` - value for the `Authorization` header, typically `Bearer <token>`

This shared import configures only the authentication header for each backend.
If you need additional per-endpoint headers, define `observability.otlp.endpoint`
directly in your workflow using the object or array form and add those headers
alongside `Authorization`.
