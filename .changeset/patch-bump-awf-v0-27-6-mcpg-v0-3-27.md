---
"gh-aw": patch
---

Bump the default `gh-aw-firewall` version to `v0.27.6` and `gh-aw-mcpg` version to `v0.3.27`, then regenerate pinned workflow artifacts.

Firewall v0.27.6 notably fixes the api-proxy AIC=0 token-usage regression (the `token-tracker-shared.js` / OTEL modules were missing from the api-proxy Docker image COPY list, silently disabling all token tracking) and the Copilot cache-write token fidelity accounting.
