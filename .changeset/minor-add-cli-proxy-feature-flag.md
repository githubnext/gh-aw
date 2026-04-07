---
"gh-aw": minor
---

Add `cli-proxy` feature flag that injects `--enable-cli-proxy` and `--cli-proxy-policy` into the AWF command, giving agents secure read-only `gh` CLI access without exposing `GITHUB_TOKEN` (requires firewall v0.25.14+).
