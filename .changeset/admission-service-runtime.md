---
"gh-aw": minor
---

Add OIDC-authenticated activation admission configured through managed Actions variables. Valid approval supersedes daily-AIC validation and applies the required primary and independently optional threat-detection/evals allowances to their existing runtime credit limits, overriding configured limits when supplied. Omitted optional allowances, disabled admission and ignored outages retain existing component budgets; disabled admission and ignored outages also retain daily validation. Add redacted lifecycle logging and collapsible admission step summaries without exposing credentials, identity, policy amounts or service diagnostics. Send caller-owned W3C trace context without requiring response trace IDs. Skip token generation and admission when no endpoint is configured.

Include the compiled workflow's frontmatter/body hash in admission requests as a cache hint, without replacing authenticated invocation identity or authorization checks.
