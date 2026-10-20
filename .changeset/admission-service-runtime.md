---
"gh-aw": minor
---

Add OIDC-authenticated activation admission configured through managed Actions variables. Valid approval supersedes daily-AIC validation; disabled admission and ignored outages retain daily validation as fallback. Keep required primary and optional threat-detection/evals policy allowances within activation, leaving downstream jobs and runtime budgets unchanged. Send caller-owned W3C trace context without requiring response trace IDs. Skip token generation and admission when no endpoint is configured.
