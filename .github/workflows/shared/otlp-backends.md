---
network:
  allowed:
    - "*.sentry.io"
    - "*.grafana.net"
    - "*.datadoghq.com"
    - "*.datadoghq.eu"
    - "*.ddog-gov.com"
observability:
  otlp:
    endpoint:
      - url: ${{ secrets.GH_AW_OTEL_SENTRY_ENDPOINT }}
        headers:
          Authorization: ${{ secrets.GH_AW_OTEL_SENTRY_AUTHORIZATION }}
      - url: ${{ secrets.GH_AW_OTEL_GRAFANA_ENDPOINT }}
        headers:
          Authorization: ${{ secrets.GH_AW_OTEL_GRAFANA_AUTHORIZATION }}
      - url: ${{ secrets.GH_AW_OTEL_DATADOG_ENDPOINT || format('https://otlp-intake.{0}/v1/traces', secrets.DD_SITE || 'datadoghq.com') }}
        headers:
          DD-API-KEY: ${{ secrets.GH_AW_OTEL_DATADOG_API_KEY || secrets.DD_API_KEY }}
---

## Required secrets

Consumers of this shared import must provision the following secrets:

- `GH_AW_OTEL_SENTRY_ENDPOINT`
- `GH_AW_OTEL_SENTRY_AUTHORIZATION`
- `GH_AW_OTEL_GRAFANA_ENDPOINT`
- `GH_AW_OTEL_GRAFANA_AUTHORIZATION`
- `GH_AW_OTEL_DATADOG_ENDPOINT` (optional; defaults to `https://otlp-intake.${DD_SITE}/v1/traces`)
- `GH_AW_OTEL_DATADOG_API_KEY` (optional; falls back to `DD_API_KEY`)
