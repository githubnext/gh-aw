---
"gh-aw": patch
---

Upload firewall audit logs as a dedicated `firewall-audit-logs` artifact for firewall-enabled workflows.

This adds a compiler-generated upload step for firewall audit logs (with `if: always()` and `if-no-files-found: ignore`), removes these logs from the generic `agent` artifact, and extends coverage to all firewall-enabled engines including Gemini.
