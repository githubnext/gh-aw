# ADR-52924: Repository-Level Grype Config for Risk-Accepted CVEs

**Date**: 2026-08-15
**Status**: Draft
**Deciders**: pelikhan, copilot-swe-agent

---

### Context

The `gh aw compile --grype` scanner gates CI on `[Critical]` vulnerability findings. Three glibc CVEs (CVE-2026-5450, CVE-2026-5928, CVE-2026-5435) are present in the Debian base layer of `ghcr.io/github/github-mcp-server:v1.9.0`. Debian has published no fixed `libc6` version for any of these CVEs, so there is no upgrade path available. The daily container scan was failing consistently with findings that cannot be remediated, blocking the pipeline with no actionable next step. gh-aw only runs this image as an MCP server and does not redistribute glibc.

### Decision

We will add optional repository-level grype configuration support to `pkg/cli/grype.go`. When `.grype.yaml` exists at the repository root it is mounted read-only into the grype scanner container and passed via `--config`, applying its `ignore` rules to the scan. The `.grype.yaml` file is reserved for documented, CVE-scoped risk acceptances where no upstream fix exists, mirroring the existing pattern used by `--grant` with `.grant.yaml`. Each ignore rule is scoped to a specific CVE ID and package so newly disclosed vulnerabilities in the same package remain visible. Rules carry a `reason` field for auditability and must be removed as soon as Debian ships a patched base image.

### Alternatives Considered

#### Alternative 1: Disable the Critical gate for the affected image

Remove or relax the `[Critical]` severity gate for `ghcr.io/github/github-mcp-server` entirely so the scan passes without further changes. This was rejected because it would suppress all future Critical findings in that image, not just the three unfixable CVEs, removing meaningful signal for vulnerabilities that do have fixes available.

#### Alternative 2: Suppress findings via CLI flags at the call site (not checked in)

Pass grype `--ignore-wont-fix` or ad-hoc `--config` flags from the gh-aw CLI invocation code rather than committing a `.grype.yaml` file to the repository. This was rejected because the rules would not be visible to code review, would not be co-located with the codebase they protect, and would make it harder to audit which CVEs are accepted and why. Checked-in ignore rules surface through normal PR review.

#### Alternative 3: Pin to an older image that predates the CVEs

Roll back the pinned digest for `ghcr.io/github/github-mcp-server` to a tag unaffected by these CVEs. This was rejected because the CVEs affect the upstream Debian base layer across all current releases; no available tag is unaffected. Additionally, pinning to an older image would introduce other unpatched vulnerabilities and diverge from the upstream release track.

### Consequences

#### Positive
- Daily container scans pass again without manual intervention once Debian ships patches.
- Each risk acceptance is explicitly documented with a CVE ID, package scope, and `reason`, making the security posture auditable via normal code review.
- Newly disclosed `libc6` vulnerabilities are still reported because rules are scoped to specific CVE IDs, not to the package as a whole.
- The approach is consistent with the existing `.grant.yaml` pattern already used for license policy, reducing cognitive overhead.

#### Negative
- The three accepted CVEs are suppressed from scan output, reducing the visible finding count. Reviewers must consult `.grype.yaml` to understand the full risk picture.
- Ignore rules require discipline to remove: if Debian ships a fix and nobody cleans up the rule, the patched CVE continues to be silenced. The daily `--force-refresh-container-pins` scan mitigates this but does not enforce rule removal.

#### Neutral
- Docker arg construction was extracted into a testable `grypeDockerArgs` helper; the change is a refactor with no behaviour change when `.grype.yaml` is absent.
- The feature is opt-in: repositories without `.grype.yaml` are unaffected.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
