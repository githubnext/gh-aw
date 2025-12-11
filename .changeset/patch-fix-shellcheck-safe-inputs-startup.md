---
"gh-aw": patch
---

Fix shellcheck violations in `pkg/workflow/sh/start_safe_inputs_server.sh` that caused
issues in compiled workflow lock files. Changes include quoting variables, using compound
redirection, and replacing `ps | grep` with `pgrep`. Recompiled workflows were updated
to propagate the fixes to lock files.

