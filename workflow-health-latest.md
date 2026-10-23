# Workflow Health — 2026-10-23T04:40Z
- 19 workflows still at 0% (1 executed run each), mostly smoke-* plus agent-job-health, daily-doc-healer, unbloat-docs, semantic-function-refactor. Compile output clean.
- #70777 (API quota exhaustion) CLOSED 2026-10-22 and metrics unchanged -> attribution stale. smoke-claude run 39877179050 shows OTLP 401 + digest-mismatch in safe_outputs, no quota error. Related open: #71278 (AOAI no AI credits pricing), #69800 (Redact secrets step).
- Action: filed new P1 tracker issue for the 0% cluster.
