---
# OpenTelemetry (OTel) shared import
# Provides OTLP observability telemetry for agentic workflows.
# Configures the OTLP endpoint and authentication headers via repository secrets.
#
# Required secrets:
#   GH_AW_OTEL_ENDPOINT — OTLP collector endpoint URL
#   GH_AW_OTEL_HEADERS  — OTLP authentication headers (e.g. "Authorization=Bearer <token>")
#
# Usage:
#   imports:
#     - shared/otel.md

imports:
  - shared/observability-otlp.md
---
