---
"gh-aw": minor
---

Add `cli-proxy` and `cli-proxy-writable` feature flags that inject `--enable-cli-proxy`, `--cli-proxy-writable`, and `--cli-proxy-policy` into the AWF command, giving agents secure `gh` CLI access without exposing `GITHUB_TOKEN` (requires firewall v0.25.14+).
