---
"gh-aw": patch
---

Temporarily move the default GitHub MCP server image tag from `v1.9.0` to `sha-eff4c3c` so newly compiled workflows pin a newer upstream container digest while waiting for the next stable release.

Also extend Grant’s license allowlist with Debian base-layer licenses used by runtime container images so container license scans stop flagging expected OS package licenses.
