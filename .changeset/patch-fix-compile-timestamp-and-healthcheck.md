---
"gh-aw": patch
---

Fix compile timestamp handling and improve MCP gateway health check logging

Fixes handling of lock file timestamps in the compile command and enhances
gateway health check logging and validation order to check gateway readiness
before validating configuration files. Also includes minor workflow prompt
simplifications and safeinputs routing fixes when the sandbox agent is disabled.

This is an internal tooling and workflow change (patch).

