---
"gh-aw": patch
---

Narrow the opt-in `claude` network domain set. Telemetry, certificate/OCSP services, package mirrors, container registries, and Playwright downloads now require explicit domains or ecosystem opt-ins such as `defaults`, `containers`, and `playwright`. The direct Anthropic API is included when Claude uses its default Anthropic target. `engine.api-target` configures the Copilot target and does not replace Claude's Anthropic endpoint; set `ANTHROPIC_BASE_URL` to configure a custom Anthropic target, which does not automatically allow direct access to `api.anthropic.com`. Add `api.anthropic.com` explicitly if a workflow intentionally needs direct Anthropic access alongside a custom target. The broader `anthropic.com` suffix remains excluded because it would also allow `statsig.anthropic.com`.

If an agent still needs one of the removed hosts, add the specific hostname or matching ecosystem to `network.allowed`. Configuring the Playwright tool continues to opt in to its download domains.
