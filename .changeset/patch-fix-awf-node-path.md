---
"gh-aw": patch
---

Fixed AWF Copilot execution to use an absolute Node.js binary path when available so workflows no longer fail with `node: command not found` on runners where `sudo` resets `PATH`.

Also regenerated stale compiled workflow lock files after context propagation refactors.
