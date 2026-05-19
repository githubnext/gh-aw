---
network:
  allowed:
    - "*.sentry.io"
observability:
  otlp:
    endpoint:
      - url: ${{ secrets.GH_AW_OTEL_SENTRY_ENDPOINT }}
        headers:
          Authorization: ${{ secrets.GH_AW_OTEL_SENTRY_AUTHORIZATION }}
---

## Required secrets

Consumers of this shared import must provision the following secrets:

- `GH_AW_OTEL_SENTRY_ENDPOINT`
- `GH_AW_OTEL_SENTRY_AUTHORIZATION`
