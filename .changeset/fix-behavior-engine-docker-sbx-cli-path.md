---
"gh-aw": patch
---

Fix `ENOENT` failures for behavior-defined engines (such as Crush) running on the `docker-sbx` and `cloud-hypervisor` sandbox runtimes. These runtimes do not mount the runner tool cache into the sandbox, so an engine CLI installed with `npm install -g` was invisible to the agent process even though the host-side install and verify steps succeeded. The engine CLI is now also staged into `${RUNNER_TEMP}/gh-aw/engine-cli` and that `bin` directory is prepended to the sandbox `PATH`, matching the Claude and Codex engines.

The Crush harness now resolves its CLI on `PATH` before spawning it and fails with `crush not found in sandbox PATH: <path>` instead of a bare `spawnSync crush ENOENT`.
