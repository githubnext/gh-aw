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
- `GH_AW_OTEL_SENTRY_AUTHORIZATION`
- `GH_AW_OTEL_GRAFANA_ENDPOINT`
- `GH_AW_OTEL_GRAFANA_AUTHORIZATION`

Do not use the older header secret names `GH_AW_OTEL_SENTRY_HEADERS` and
`GH_AW_OTEL_GRAFANA_HEADERS`; they have been replaced by the
`*_AUTHORIZATION` secrets above.
