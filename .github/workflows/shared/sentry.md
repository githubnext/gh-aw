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

<!--
## Required secrets

Consumers of this shared import must provision the following secrets:

- `GH_AW_OTEL_SENTRY_ENDPOINT`
- `GH_AW_OTEL_SENTRY_AUTHORIZATION`
-->

Read `skills/otel-queries/SKILL.md` before telemetry analysis and follow its fixed query loop.

When producing reliability reports from Sentry telemetry:

1. Start by checking whether `spans`, `errors`, and `logs` datasets have recent data; treat empty datasets as an explicit observability finding.
2. Explicitly verify whether these attributes are present before claiming failures from traces:
   - `span.status`
   - `gen_ai.response.finish_reasons`
   - `gh_aw.workflow_name`
   - `release`
3. If those fields are missing, report the result as **inconclusive runtime outcome + confirmed instrumentation gap**, not as a confirmed timeout/failure.
4. For any latency or token outlier, include concrete evidence (count, max value, and at least one trace ID) rather than anecdotal descriptions.
