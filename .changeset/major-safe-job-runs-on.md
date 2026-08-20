---
"gh-aw": major
---

Support runner-group objects in custom safe-job `runs-on` fields.

The deprecated `safe-outputs.jobs.<job>.runner` alias has been removed. Run `gh aw fix` to migrate existing workflows to `safe-outputs.jobs.<job>.runs-on`.
