# ADR-46128: Auto-Detect GitHub Enterprise Host from Git Remote When GH_HOST Is Unset

**Date**: 2026-07-17
**Status**: Draft
**Deciders**: mnkiefer

---

### Context

`gh-aw` commands that create PATs or verify authentication (e.g., `doctor`, `setup auth`, `setup repository`) previously hardcoded `github.com` as the target host unless the user explicitly set `GH_HOST`. Users operating in GitHub Enterprise Server (GHES) checkouts who have not set `GH_HOST` would receive PAT creation URLs and auth guidance pointing to the wrong host, causing silent failures or confusing redirects. A reliable, zero-configuration signal for the GHES host already exists in the local git repository's `origin` remote URL.

### Decision

We will implement implicit GHES host detection by parsing the `origin` remote URL at command runtime. When `GH_HOST` is unset and the detected host is not `github.com`, the tool will call `workflow.SetDefaultGHHost(detectedHost)` to configure the session host before executing auth checks or constructing PAT creation URLs. PAT URL construction is refactored into a shared `buildPATCreationURL` helper that applies this detection for both Copilot and generic PAT flows.

### Alternatives Considered

#### Alternative 1: Require Explicit GH_HOST Configuration

Users must set `GH_HOST` (or `GITHUB_HOST` / `GITHUB_ENTERPRISE_HOST` / `GITHUB_SERVER_URL`) before using GHES-targeting commands. The tool documents this requirement and returns a clear error when the host cannot be determined. This approach is simpler and deterministic but imposes manual setup friction on every GHES user and breaks the zero-configuration experience the tool aims for.

#### Alternative 2: Env-Var-Only Expansion Without Git Remote Inspection

Expand host detection to check a prioritized list of environment variables (`GITHUB_SERVER_URL`, `GITHUB_ENTERPRISE_HOST`, `GITHUB_HOST`) in addition to `GH_HOST`, without inspecting the git remote. This avoids any filesystem I/O and is safe in all working-directory contexts but fails when none of these variables are set — which is the common case for developers running locally outside CI/CD pipelines.

### Consequences

#### Positive
- GHES users in a checkout receive correct PAT creation URLs and auth commands without manual host configuration.
- The `buildPATCreationURL` helper consolidates host-aware URL construction, eliminating the hardcoded `github.com` string in both Copilot PAT and system PAT flows.
- New integration-style tests verify host detection end-to-end by creating real temporary git repos with remote URLs.

#### Negative
- Auto-detection couples command behavior to the current working directory's git state; commands run outside a git repo or in a repo with a non-GitHub remote will silently fall back to `github.com` rather than surfacing a clear error.
- Git remote inspection adds I/O at command startup for `setup auth` and `setup repository check`, which could slow down commands in environments with sluggish filesystem access.
- If the `origin` remote points to a mirror or proxy (not the canonical GHES host), the detected host will be wrong, and the error may not be obvious to the user.

#### Neutral
- The `doctor` command long description is updated to document the auto-detection behavior and the manual fallback (`gh auth login --hostname <host>` or `GH_HOST`), keeping user-facing guidance in sync with the implementation.
- The change is scoped to host detection at command entry; downstream `gh` invocations already use the configured default host once `workflow.SetDefaultGHHost` is called.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
