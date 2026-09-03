# Formal Notes: security-architecture-spec-validation.md

**Last formalized**: 2026-09-03-15-35-21
**Notation**: TLA+-style bash-guard predicates / Z3-style version-gate conjunction
**Issue**: created via safe-output (number assigned post-processing)

## Predicates

| ID | Predicate | Description |
|---|---|---|
| P1 | `ChrootPatchInjectedOnlyWhenDockerHostMatchesTCP` | AWF chroot config patch (§8c T-SI-002/003/005) only injected when `DOCKER_HOST` matches the `tcp://` arc-dind regex |
| P2 | `ExcludeEnvFlagGatedByAWFVersion` | `--exclude-env` flags only emitted when effective AWF version >= `AWFExcludeEnvMinVersion` (v0.25.3) |
| P3 | `EnvAllAlwaysPrecedesExcludeEnv` | `--env-all` is unconditional and always precedes any `--exclude-env` pairs |
| P4 | `ToolCacheMountReadOnlyOutsideOptPrefix` | Tool-cache mount is read-only and skipped when path is under `/opt/` (image-baked) or missing |
| P5 | `MountRestrictedToStagedGhAwTree` | Sandbox mounts must be scoped under `${RUNNER_TEMP}/gh-aw`, never arbitrary host paths |
| P6 | `ChrootPatchUsesBashVariantForDetectionRuns` | Detection runs use the bash/jq chroot-patch variant; agent runs use the Node.js variant (T-SI-007 composed sandboxing) |
| P7 | `VersionGateMonotonicity` | Version-gate helper is monotonic: version >= X and X >= min implies version >= min |

## Key Invariants

- Sandbox chroot patching and `--exclude-env` filtering are both conditionally gated (by DOCKER_HOST value and AWF version respectively) rather than unconditional — absence of the gate must never silently degrade to an insecure default.
- All sandbox mounts observed in compiled workflows are scoped under the staged `${RUNNER_TEMP}/gh-aw` tree; no arbitrary host path (e.g. `/`, `/var/run/docker.sock`) should ever be mountable.
- Detection-job and agent-job invocations share the same underlying chroot-patch mechanism but select different script variants (bash vs Node.js) — sandboxing and detection composed, not exclusive (T-SI-007).

## Edge Cases Identified

- Malformed/partial `tcp` prefixes (`tcp:/`, `tcp127.0.0.1`, case variations, leading whitespace) must NOT match the DOCKER_HOST arc-dind regex.
- An empty (but non-nil) exclude-env list must produce zero `--exclude-env` pairs, not an empty/bare flag.
- Non-semver / unparseable version strings (branch names, missing "v" prefix) must conservatively fail version gates — except empty string, which falls back to the default version.

## Notes for Future Runs

- This run targeted §8c "Sandbox Isolation Supplemental Evidence" (T-SI-001 to T-SI-007), flagged as the top-priority partial-evidence gap in prior runs' notes.
- The generated test suite is written against a **stub** re-implementation of `versionAtLeast`, `awfSupportsExcludeEnv`, DOCKER_HOST regex matching, and mount/patch-variant selection logic (all unexported in `pkg/workflow`). A follow-up PR should move this suite into `package workflow` (or add exported test-only wrappers) so it exercises the real `pkg/workflow/awf_command_builder.go`, `pkg/workflow/awf_arc_dind.go`, and `pkg/workflow/version_gate.go` implementations directly.
- The direct runtime host/socket-visibility probe from inside the AWF sandbox (tracked in gh-aw#48686) remains unaddressed — it requires an actual running sandbox, not unit-level Go tests, and is out of scope for this formalization.
- Next-highest-priority target per the compliance matrix: Threat Detection (T-TD-002 to T-TD-007) — only TD-01 (job presence) has been verified so far.
