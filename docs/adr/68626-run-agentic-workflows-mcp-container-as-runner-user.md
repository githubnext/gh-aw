# ADR-68626: Run the Built-in agentic-workflows MCP Container as the Runner User

**Date**: 2026-10-14
**Status**: Draft
**Deciders**: Unknown (drafted by Design Decision Gate from PR #68626)

---

### Context

Compiled agentic workflows start the built-in `agentic-workflows` MCP server inside a nested Docker container launched by the MCP gateway. The gateway itself runs as the GitHub Actions runner user, but the nested container defaulted to `root`, so every file it produced (downloaded workflow logs, audit artifacts) was owned by `root` with owner-only permissions. The host-side secret-redaction step, which runs as the runner before artifact upload, could not read those files and failed with `EACCES` at the read stage (observed in the Safe Output Health Monitor run). Because the redactor is fail-closed, unsanitized artifacts were deleted rather than uploaded, so the workflow lost its diagnostic output. The constraint is that audit and log files must stay owner-only (no broad `chmod`) while still being readable by the runner's redaction step.

### Decision

The built-in `agentic-workflows` MCP container runs with the runner's numeric identity by appending `--user $(id -u):$(id -g)` to the Docker runtime args, resolved in the host-side configuration heredoc rather than inside the nested container. Because a numeric UID with no matching `/etc/passwd` entry would otherwise fall back to `/root`, the "Copy gh-aw binary for MCP Server" install step creates a fresh `0700` home with `mktemp -d` under `$RUNNER_TEMP/gh-aw/aw-mcp-home.*`. Setup rejects symlinked parent paths and publishes the path through `GITHUB_ENV` as `GH_AW_AGENTIC_WORKFLOWS_MCP_HOME`, without deleting, changing ownership, or reusing a previous home.

Both the JSON (`renderAgenticWorkflowsMCPConfigWithOptions`) and TOML (`renderAgenticWorkflowsTOML`) renderers resolve this variable in the runner-side configuration heredoc, setting `HOME` and a dedicated writable bind mount to the same absolute path in development and release modes. The gateway's mount policy permits writing only to that fresh home; the existing runtime-tree mount remains read-only. Keeping the host path unchanged also allows nested Docker to mount it from the daemon-visible runner-temp volume on ARC/DinD runners.

### Alternatives Considered

#### Alternative 1: Relax file permissions on produced artifacts

Add a `chmod -R a+r` (or equivalent) pass over the downloaded logs and audit directory before redaction. This would make the files readable by the runner without changing container identity and would be a one-line change. It was rejected because it widens access to potentially secret-bearing audit material on a shared runner filesystem, weakening the owner-only guarantee that the redaction pipeline depends on; it treats the symptom (unreadable files) rather than the cause (wrong file owner).

#### Alternative 2: Run the redaction step as root / with elevated privileges

Have the host redactor run under `sudo` so it can read root-owned output regardless of container identity. This was rejected because it escalates privileges for a step that handles untrusted agent output, and it would not generalise to self-hosted or container-based runners where passwordless `sudo` is unavailable. It also leaves root-owned files behind in the workspace, which breaks later runner-owned cleanup and artifact upload steps.

#### Alternative 3: Resolve the UID/GID inside the nested container

Compute the identity at container start instead of in the host heredoc. Rejected because the nested container cannot observe the host runner's identity, so the value would be wrong by construction; resolving `$(id -u):$(id -g)` while the host writes the configuration is the only point where the correct identity is known.

### Consequences

#### Positive
- The host secret-redaction step can read MCP-produced logs and audit files, avoiding ownership-related failures while retaining fail-closed handling of any remaining redaction errors.
- Owner-only (`0600`/`0700`) permissions and fail-closed removal of unsanitized artifacts are preserved — no broadening of access.
- Files left in the workspace are runner-owned, so subsequent cleanup, upload, and cache steps behave normally.
- JSON and TOML renderers share the same home environment-variable and mount contract, keeping development and release modes in sync.
- Each setup uses a fresh home, so stale, public, or root-owned legacy homes cannot affect the current job.

#### Negative
- The container now runs with a numeric UID that has no `/etc/passwd` entry; any tooling inside the image that resolves the current user by name may misbehave and requires the explicit `HOME` workaround.
- Processes inside the container lose root privileges, so image paths that expect to write outside the mounted workspace or install packages at runtime will break.
- The behaviour could not be verified end-to-end in CI because the Docker daemon is unavailable in the agent environment; correctness currently rests on compiled-output tests rather than a live container start.
- All 44 affected workflow lock files had to be regenerated, enlarging the diff and creating a rebase/merge-conflict surface for concurrent PRs.

#### Neutral
- `HOME` is job-scoped under `$RUNNER_TEMP/gh-aw/` and cleaned by the runner's normal job cleanup; no shared fixed home or whole-runner-temp writable mount is introduced.
- Pricing and detection configuration are unchanged; this is strictly a runtime-identity and filesystem-ownership change.
- Regression coverage was added at three levels: Go tests for configuration expansion and private-home preparation, compiled MCP/import behaviour tests, and a private-file case in `redact_secrets.test.cjs`.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
