---
"gh-aw": patch
---

Fix Claude engine tool-permission enforcement by using `--permission-mode acceptEdits` instead of `bypassPermissions`, so workflow `--allowed-tools` restrictions are honored in CI runs.
