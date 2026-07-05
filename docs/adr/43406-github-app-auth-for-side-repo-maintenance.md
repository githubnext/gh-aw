# ADR-43406: GitHub App Authentication Support for SideRepoOps Maintenance Workflow Generator

**Date**: 2026-07-05
**Status**: Draft
**Deciders**: Unknown (copilot-swe-agent, pelikhan)

---

### Context

The maintenance workflow generator (`pkg/workflow/side_repo_maintenance.go`) produces scheduled cross-repo maintenance YAML files (`agentics-maintenance-<owner>-<repo>.yml`) for every repository a source workflow checks out with `current: true`. Prior to this change, the generator only understood `github-token:` as an authentication mechanism. When a source workflow used `github-app:` for cross-repo checkout auth, the generator silently fell back to `${{ secrets.GH_AW_GITHUB_TOKEN }}` — a secret that App-only consumers never configure — causing 100% failure on every scheduled maintenance run with no visible error at generation time.

### Decision

We will extend `SideRepoTarget` with a `GitHubApp *GitHubAppConfig` field (mutually exclusive with `GitHubToken`, mirroring `CheckoutConfig`) and update `collectSideRepoTargets` to propagate app config alongside token config using first-seen-wins semantics. When a target carries a `GitHubApp`, `generateSideRepoMaintenanceWorkflow` will inject a `create-github-app-token` mint step as the first step of each cross-repo job (`close-expired-entities`, `apply_safe_outputs`, `create_labels`, `activity_report`), and `effectiveSideRepoToken` will return the minted token reference (`${{ steps.side-repo-app-token.outputs.token }}`). The `validate_workflows` job is intentionally excluded because it operates on the repository's own content and does not require cross-repo access.

### Alternatives Considered

#### Alternative 1: Require a separate PAT alongside App auth

Users who configure `github-app:` for checkout could also be required to supply a `github-token:` in their source workflow specifically for maintenance jobs. The token would flow through the existing code path unchanged. This avoids the mint-step injection complexity but forces consumers to manage two distinct auth mechanisms — the very problem App auth is meant to eliminate. It also silently ignores the App config the user already provided, which was the root cause of the original bug.

#### Alternative 2: Single shared token-minting job with `needs:` dependencies

A dedicated `mint-side-repo-token` job could be generated at the top of the maintenance workflow, and all cross-repo jobs could reference its output via `needs: [mint-side-repo-token]`. This avoids duplicating the mint step in every job. However, it introduces job-level dependency edges that complicate the YAML structure, requires all cross-repo jobs to change their `needs:` list, and makes the generated file harder to read and debug. The per-job duplication cost (a single ~10-line step) is low enough that the structural simplicity of the per-job approach outweighs the YAML verbosity.

### Consequences

#### Positive
- App-authenticated cross-repo maintenance workflows now work end-to-end without requiring any additional secrets beyond the App credentials already declared in the source workflow.
- First-seen-wins semantics are explicit and logged via `maintenanceLog.Printf`, making conflict resolution deterministic and diagnosable when the same repo appears in multiple workflows with different auth configurations.
- The existing `buildGitHubAppTokenMintStepWithMeta` infrastructure is reused without modification, keeping the mint-step generation consistent with the compiler's other GitHub App auth paths.

#### Negative
- The `create-github-app-token` mint step is duplicated verbatim into each cross-repo job rather than being generated once. For a workflow with four cross-repo jobs, this adds approximately 40 lines of repeated YAML.
- The `validate_workflows` job deliberately does not receive the mint step, creating an asymmetry in the generated file that is not immediately obvious. Contributors unfamiliar with this decision may attempt to add App auth there, not realizing it is intentional.

#### Neutral
- `SideRepoTarget.GitHubToken` and `SideRepoTarget.GitHubApp` are mutually exclusive by convention only; no runtime enforcement prevents both from being set simultaneously. This matches the existing `CheckoutConfig` pattern but relies on callers to uphold the invariant.
- `collectSideRepoTargets` changes its internal accumulator from `map[string]string` to `map[string]sideRepoAuth`, which is a breaking change to the function's internal structure but has no effect on callers since the function is unexported.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
