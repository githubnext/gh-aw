---
"gh-aw": patch
---

Revert the Agent Workflow Firewall (AWF) binary version from v0.5.0 to v0.4.0. This updates the default firewall version and recompiles workflow lock files to reference the v0.4.0 release.

Files changed include:
- `pkg/constants/constants.go` (DefaultFirewallVersion -> v0.4.0)
- `pkg/constants/constants_test.go` (updated test expectation)
- Recompiled workflow lock files referencing v0.4.0
