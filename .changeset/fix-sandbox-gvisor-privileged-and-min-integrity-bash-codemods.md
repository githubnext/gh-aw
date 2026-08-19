---
"gh-aw": patch
---

Fix two `gh aw fix --write` codemod gaps that left `--strict` compile failures unrepaired:

- `sandbox-runtime-profiles` no longer hard-errors when `sandbox.agent.runtime: gvisor` is combined with privileged security options (`sudo`/`legacy-security`). It now keeps `runtime: gvisor` and drops the no-longer-supported `sudo`/`legacy-security` fields, instead of leaving the file untouched.
- Added a new codemod that inserts `tools.bash: false` when `tools.github.min-integrity` is set to `none` and `tools.bash` is not already specified, satisfying the strict-mode requirement that shell access be explicit.
