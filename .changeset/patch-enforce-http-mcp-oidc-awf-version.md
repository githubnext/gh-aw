---
"gh-aw": patch
---

Compilation now rejects workflows that configure `mcp-servers.<name>.auth.type: github-oidc` when `firewall.version` is pinned below v0.25.3. AWF v0.25.3+ is required so `--exclude-env` can keep Actions OIDC credentials out of the agent container.
