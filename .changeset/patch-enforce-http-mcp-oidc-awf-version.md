---
"gh-aw": patch
---

Require `permissions.id-token: write` for HTTP MCP GitHub OIDC authentication and reject `firewall.version` pins below v0.25.3. AWF v0.25.3+ is required so `--exclude-env` can keep Actions OIDC credentials out of the agent container.
