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
5. When core fields are missing, inspect the emit-side mapping in `actions/setup/js/send_otlp_span.cjs` before final recommendations:
   - workflow identity is emitted as `gh-aw.workflow.name`
   - runtime outcome is emitted via OTLP `status.code` / `status.message` and `gh-aw.run.status`
   - finish reasons are emitted only when runtime metrics include `stopReason` (`gen_ai.response.finish_reasons`)
   - release correlation is emitted as resource attribute `service.version` (Sentry `release` mapping may be backend-dependent)
