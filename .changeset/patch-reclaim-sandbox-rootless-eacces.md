---
"gh-aw": patch
---

Reclaim root-owned `/tmp/gh-aw/sandbox` before AWF `writeConfigs()` to prevent `EACCES: mkdir /tmp/gh-aw/sandbox/firewall/logs` on runners with rootless-container residue.
