---
# OpenTelemetry (OTel) shared import
# Provides OTLP observability telemetry for agentic workflows.
# Configures OTLP endpoints and authentication headers via repository secrets.
#
# Required secrets:
#   GH_AW_OTEL_SENTRY_ENDPOINT  — Sentry OTLP endpoint URL
#   GH_AW_OTEL_SENTRY_HEADERS   — Sentry OTLP authentication headers
#   GH_AW_OTEL_GRAFANA_ENDPOINT — Grafana OTLP endpoint URL
#   GH_AW_OTEL_GRAFANA_HEADERS  — Grafana OTLP authentication headers
#
# Usage:
#   imports:
#     - shared/otel.md

imports:
  - shared/observability-otlp.md
---
