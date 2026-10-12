---
"gh-aw": patch
---

Narrow the opt-in `claude` network domain set. Telemetry, certificate/OCSP services, package mirrors, container registries, and Playwright downloads now require explicit domains or ecosystem opt-ins such as `defaults`, `containers`, and `playwright`. The direct Anthropic API is included only when Claude uses its default Anthropic target; workflows configured with `ANTHROPIC_BASE_URL` or `engine.api-target` no longer get a direct `api.anthropic.com` bypass. Add `api.anthropic.com` explicitly if a workflow intentionally needs direct Anthropic access alongside a custom target. The broader `anthropic.com` suffix remains excluded because it would also allow `statsig.anthropic.com`.

If an agent still needs one of the removed hosts, add the specific hostname or matching ecosystem to `network.allowed`. Configuring the Playwright tool continues to opt in to its download domains.
