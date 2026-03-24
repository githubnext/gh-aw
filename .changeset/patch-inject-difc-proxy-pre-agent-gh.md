---
"gh-aw": patch
---

Inject a temporary DIFC proxy around pre-agent custom steps that use `GH_TOKEN` when GitHub guard policies are configured, so `gh` CLI traffic is filtered before the agent starts. Also add setup scripts and artifact wiring for proxy lifecycle and diagnostics.
