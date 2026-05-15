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
