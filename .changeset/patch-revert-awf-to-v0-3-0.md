---
"gh-aw": patch
---

Reverted the Agent Workflow Firewall (AWF) binary from v0.5.0 to v0.3.0 and recompiled workflow lock files so they reference v0.3.0.

This is an internal/tooling change (firewall binary version and compiled locks) and does not change CLI behavior.

