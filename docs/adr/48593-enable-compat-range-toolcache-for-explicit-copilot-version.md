# ADR-48593: Enable Compat-Range Toolcache Matching for Explicit Copilot CLI Version Requests

**Date**: 2026-07-28
**Status**: Draft
**Deciders**: pelikhan, copilot-swe-agent

---

### Context

The Copilot CLI install script (`install_copilot_cli.sh`) has two code paths for locating a compatible binary: (1) range-based toolcache lookup driven by `compat.json`, and (2) exact-version toolcache lookup when an explicit version is passed. Path (1) allows a cached runner binary within the compat window to satisfy the install without a network download; path (2) requires an exact version match and falls back to a network download on any mismatch.

Because `pkg/workflow/copilot_installer.go` always passes an explicit version (defaulting to `DefaultCopilotVersion` when unset), path (2) was the only path ever exercised in practice. When `DefaultCopilotVersion` (1.0.75) drifted past the `compat.json` `max-agent` (1.0.56), the cached runner toolcache entry (1.0.56) was always rejected by exact-match, forcing a network download on every agentic job. A CDN blip of ~30 s was sufficient to fail the job entirely—twice per run (once in the `agent` job, once in the `detection` job). Additionally, `GH_AW_COMPILED_VERSION` was never emitted in compiled job environments, so the install script had no compiled version to resolve a compat window even if it wanted to.

### Decision

We will run compat-matrix resolution unconditionally—including when an explicit version is provided—so that `find_cached_copilot_bin` receives a populated compat range and can fall through to range-based matching when the exact version is not cached. Specifically:

- `install_copilot_cli.sh`: call `resolve_version_from_compat` in the explicit-version branch and populate `COMPAT_MATCHED_MIN_AGENT`/`COMPAT_MATCHED_MAX_AGENT`; in `find_cached_copilot_bin`, fall through to the range check when no exact match is found and a compat range is available, rather than unconditionally `continue`-ing.
- `compiler_main_job_helpers.go` / `threat_detection_job.go`: emit `GH_AW_COMPILED_VERSION` in job-level env for release builds so the script can resolve the compat window.
- `.github/aw/compat.json`: update `max-agent` from 1.0.56 to 1.0.75 to include the current `DefaultCopilotVersion`.
- `pkg/constants/version_constants_test.go`: add `TestDefaultCopilotVersionWithinCompatWindow` as a CI gate to assert `min-agent ≤ DefaultCopilotVersion ≤ max-agent`, preventing this class of drift from recurring.

### Alternatives Considered

#### Alternative 1: Remove explicit version pinning and always resolve via compat matrix

The install script could drop the explicit-version argument and always use compat-matrix resolution to select the best available version. This would give full range-based toolcache matching without any bypass. It was rejected because the explicit version pin provides a stable, testable contract: callers can assert exactly which CLI version is installed. Losing that contract complicates debugging and prevents version-specific rollouts.

#### Alternative 2: Only bump compat.json max-agent without changing the install script

Updating `max-agent` to 1.0.75 would fix the immediate drift and allow the range check to pass in theory, but the install script's explicit-version branch still skips compat resolution and calls `find_cached_copilot_bin` with an empty compat range. The toolcache entry would continue to be rejected by exact-match. This approach fixes the data without fixing the logic, leaving the underlying bypass latent for the next drift cycle.

### Consequences

#### Positive
- Runner toolcache satisfies Copilot CLI installs for any version within the compat window, eliminating unnecessary network downloads on every agentic job.
- CDN failures during Copilot CLI download no longer cause hard job failures when a compatible cached binary exists.
- `TestDefaultCopilotVersionWithinCompatWindow` creates a CI gate that blocks future `DefaultCopilotVersion`/`compat.json` drift before it reaches production.

#### Negative
- Range-based toolcache matching may serve a cached binary version that differs from the exact requested version; behavioral equivalence relies on the compat window being accurately defined. An overly broad window could accept an incompatible binary.
- `GH_AW_COMPILED_VERSION` is now present in compiled job-level environments for release builds, exposing compiler version metadata in workflow logs and to any downstream tool that reads job env vars.

#### Neutral
- The compat window is now a load-bearing correctness invariant: `compat.json` must be updated alongside `DefaultCopilotVersion` on every bump, enforced by the new test.
- The change in `find_cached_copilot_bin` is a fall-through rather than a new code path; the existing range comparison logic below the exact-match block is unchanged.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
