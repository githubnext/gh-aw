---
"gh-aw": patch
---

Refactor workflow compilation to replace hardcoded `/opt/gh-aw` paths with `GH_AW_HOME`-based constants, enabling self-hosted runners to relocate the installation directory via the `GH_AW_HOME` environment variable.
