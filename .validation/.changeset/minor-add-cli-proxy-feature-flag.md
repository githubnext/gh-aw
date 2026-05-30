---
"gh-aw": minor
---

Add `cli-proxy` feature flag that injects `--difc-proxy-host` and `--difc-proxy-ca-cert` into the AWF command, starting `difc-proxy` on the host before AWF and giving agents secure read-only `gh` CLI access without exposing `GITHUB_TOKEN` (requires firewall v0.26.0+).
